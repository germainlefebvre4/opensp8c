# change-review-actions Specification

## Purpose

Permet à l'utilisateur d'agir sur un change en revue (statut To Review) : l'approuver pour fusionner son travail, ou demander une correction qui relance le worker, depuis l'onglet Actions du DetailPanel ou par glisser-déposer, sans laisser de restes orphelins quand le change est supprimé.

## Requirements

### Requirement: Approuver et fusionner un change en revue
Le backend SHALL exposer `POST /api/workspaces/{id}/changes/{name}/review/approve`. Pour un change dont le statut est `to-review` et sans worker actif, il SHALL fusionner `feature/<change>` dans la branche actuellement extraite du dépôt (fusion `--no-ff`), supprimer ensuite le worktree puis la branche, et répondre `200` avec la branche cible. Le marqueur de revue disparaissant avec la branche, le change SHALL passer en `done` (le `tasks.md` fusionné est coché) et un événement `change_updated` SHALL être publié. L'approbation SHALL NOT supposer que le pool d'agents tourne et SHALL NOT lancer de session d'agent. Elle SHALL refuser avec `409` un change qui n'est pas en revue (code `not_in_review`) ou sur lequel un worker est actif (`worker_active`). En cas de refus ou d'échec, la branche, le worktree, le travail committé et le marqueur de revue SHALL être conservés tels qu'avant l'appel, et la réponse SHALL porter un code d'erreur et un message lisibles.

#### Scenario: Approbation réussie
- **WHEN** l'utilisateur approuve `add-user-auth`, en revue, alors que `main` n'a pas avancé
- **THEN** `feature/add-user-auth` est fusionnée dans `main`, le worktree puis la branche sont supprimés, la réponse est `200` avec la branche `main`, et le change passe en `done`

#### Scenario: Approbation avec le pool arrêté
- **WHEN** l'utilisateur approuve un change en revue alors que le pool du workspace est arrêté
- **THEN** l'approbation aboutit comme si le pool tournait

#### Scenario: Change qui n'est pas en revue
- **WHEN** l'utilisateur appelle l'approbation d'un change au statut `todo`
- **THEN** la réponse est `409` avec le code `not_in_review` et rien n'est modifié

#### Scenario: Worker actif sur le change
- **WHEN** l'approbation est demandée alors qu'un worker est actif sur ce change
- **THEN** la réponse est `409` avec le code `worker_active` et rien n'est modifié

#### Scenario: Approbations concurrentes
- **WHEN** deux approbations du même change arrivent simultanément
- **THEN** une seule fusionne ; l'autre reçoit `409` avec le code `not_in_review` ou un refus de fusion en cours, sans second merge

### Requirement: Intégration et revalidation à l'approbation
À l'approbation, si la branche cible contient un commit absent de `feature/<change>`, le backend SHALL l'intégrer dans le worktree du change, puis SHALL rejouer la commande de validation du workspace sur le résultat, sans session d'agent ni tentative de guérison ; la fusion SHALL n'avoir lieu que si la validation réussit. Lorsque la branche cible ne contient aucun commit absent de `feature/<change>`, la validation SHALL NOT être rejouée. Le backend SHALL refuser de fusionner : si la branche de base enregistrée n'est pas la branche courante (code `base_branch_mismatch`), si un merge est déjà en cours dans le dépôt (code `merge_in_progress`), si l'intégration provoque un conflit (code `integration_conflict`, intégration annulée et worktree retrouvé tel qu'avant) ou si la validation échoue (code `validation_failed`, avec la sortie de validation). La fusion SHALL être sérialisée avec celles des workers du même workspace (verrou de fusion partagé) et SHALL NOT démarrer tant que la branche cible contient un commit absent du change au moment où le verrou est détenu : dans ce cas elle SHALL libérer le verrou et recommencer l'intégration, dans la limite de 3 intégrations successives, au-delà de laquelle elle répond `409` (code `target_moving`).

#### Scenario: Branche cible avancée sans conflit
- **WHEN** `main` a reçu un commit depuis la création de `feature/add-user-auth` et que l'utilisateur approuve le change
- **THEN** `main` est intégrée dans le worktree, la validation est rejouée, puis la fusion a lieu si elle réussit

#### Scenario: Branche cible inchangée
- **WHEN** `main` n'a reçu aucun commit depuis la création de la branche et que l'utilisateur approuve
- **THEN** aucune intégration ni validation n'a lieu et la fusion suit son cours

#### Scenario: Conflit d'intégration
- **WHEN** les commits de `main` entrent en conflit avec le travail de `feature/add-user-auth`
- **THEN** la réponse est `409` avec le code `integration_conflict`, l'intégration est annulée, la branche, le worktree et le marqueur sont inchangés et le change reste en To Review

#### Scenario: Validation en échec
- **WHEN** la validation rejouée après intégration échoue
- **THEN** la réponse est `422` avec le code `validation_failed` et la sortie de validation, aucune fusion n'a lieu et le change reste en To Review

#### Scenario: Dépôt hors de la branche de base
- **WHEN** le dépôt n'est plus sur la branche de base enregistrée du change
- **THEN** la réponse est `409` avec le code `base_branch_mismatch` et aucune fusion n'a lieu

#### Scenario: Merge déjà en cours
- **WHEN** un merge est déjà en cours dans le dépôt au moment de l'approbation
- **THEN** la réponse est `409` avec le code `merge_in_progress`, sans annuler ce merge

#### Scenario: Fusion d'un worker en cours sur le même workspace
- **WHEN** un worker `full-autonomy` du même workspace détient le verrou de fusion pendant une approbation
- **THEN** l'approbation attend le verrou, puis intègre le résultat de ce worker si nécessaire avant de fusionner

### Requirement: Demander une correction par des tâches dans tasks.md
Le backend SHALL exposer `POST /api/workspaces/{id}/changes/{name}/review/request-correction` avec un corps `{ "feedback": "<texte>" }`. Pour un change en revue et sans worker actif, il SHALL ajouter exactement une tâche décochée au `tasks.md` du worktree du change, dans une section `## Corrections` (créée si absente, réutilisée sinon), contenant l'intégralité du retour (y compris s'il est multi-lignes) sans créer d'autres tâches, puis committer ce fichier dans `feature/<change>`, lever le marqueur de revue, publier `change_updated` et répondre `204`. Le retour SHALL être non vide après suppression des espaces, sinon la réponse est `400` et rien n'est modifié. Le retour de l'utilisateur SHALL survivre à l'arrêt du pool et du backend puisqu'il est versionné dans la branche. Si le worktree du change n'existe plus, le backend SHALL le recréer à partir de la branche avant d'écrire. En cas d'échec d'écriture ou de commit, le marqueur SHALL être conservé et la réponse porter un message lisible.

#### Scenario: Correction demandée
- **WHEN** l'utilisateur demande une correction sur `add-user-auth` avec le retour « Le bouton Annuler ne ferme pas le dialogue »
- **THEN** le `tasks.md` du worktree contient, sous `## Corrections`, une tâche décochée reprenant ce texte, elle est committée dans `feature/add-user-auth`, le marqueur de revue est levé et la réponse est `204`

#### Scenario: Retour multi-lignes
- **WHEN** le retour comporte plusieurs lignes dont certaines commencent par `- [ ]`
- **THEN** une seule tâche décochée est ajoutée (le compteur de tâches du change augmente de 1)

#### Scenario: Deuxième demande de correction
- **WHEN** le change repasse en revue puis l'utilisateur demande une nouvelle correction
- **THEN** une nouvelle tâche décochée est ajoutée dans la même section `## Corrections`, sans dupliquer la section

#### Scenario: Retour vide
- **WHEN** le retour est vide ou ne contient que des espaces
- **THEN** la réponse est `400` et ni le `tasks.md`, ni la branche, ni le marqueur ne sont modifiés

#### Scenario: Change qui n'est pas en revue ou worker actif
- **WHEN** la correction est demandée pour un change qui n'est pas en revue, ou sur lequel un worker est actif
- **THEN** la réponse est `409` avec le code `not_in_review` ou `worker_active` et rien n'est modifié

#### Scenario: Retour conservé après redémarrage
- **WHEN** le backend est redémarré après la demande de correction, avant que le worker ne reprenne
- **THEN** la tâche de correction est toujours présente dans `feature/<change>` et sera traitée à la reprise

### Requirement: Reprise du change après une demande de correction
Une fois le marqueur levé par une demande de correction, le change SHALL redevenir éligible au dispatcher (statut dérivé de l'avancement et de l'état « lancé »). Le worker qui le reprend SHALL réutiliser la branche et le worktree existants, appliquer les tâches restantes, dont la correction, avec le même flux que pour toute reprise (`/opsx:apply`, validation, contrôle de complétion exigeant que toutes les tâches, corrections comprises, soient cochées), puis repasser le change en revue en mode `hitl-review`. Si le pool est arrêté au moment de la demande, le change SHALL être repris au prochain démarrage du pool.

#### Scenario: Worker relancé sur la correction
- **WHEN** la correction est demandée et que le pool tourne
- **THEN** le dispatcher assigne un worker au change, qui réutilise `feature/<change>` et son worktree et traite la tâche de correction

#### Scenario: Retour en revue après correction
- **WHEN** le worker termine les tâches, corrections incluses, et que la validation réussit
- **THEN** le change repasse en To Review avec un nouveau marqueur de revue

#### Scenario: Correction non appliquée
- **WHEN** le worker termine sans avoir coché la tâche de correction
- **THEN** il ne finalise pas le changement et passe à `paused` (contrôle de complétion existant)

#### Scenario: Pool arrêté à la demande de correction
- **WHEN** la correction est demandée alors que le pool est arrêté
- **THEN** le change n'est plus en To Review et sera repris par un worker au prochain démarrage du pool

### Requirement: Dialogue de saisie du retour de correction
Le bouton « Demander correction » de l'onglet Actions du DetailPanel et le drag d'une carte de To Review vers In Progress SHALL ouvrir le même dialogue : un champ de texte obligatoire pour décrire ce qui ne va pas, un bouton de confirmation désactivé tant que le champ est vide, et un bouton d'annulation. La confirmation SHALL envoyer le retour au backend, fermer le dialogue et le DetailPanel selon le succès de l'appel ; en cas d'erreur, le message SHALL être affiché dans le dialogue, le texte saisi conservé. L'annulation (bouton, fermeture ou Échap) SHALL ne rien envoyer et, après un drag, remettre la carte dans To Review.

#### Scenario: Ouverture depuis le bouton
- **WHEN** l'utilisateur clique sur « Demander correction » dans l'onglet Actions d'un change en revue
- **THEN** le dialogue de saisie s'ouvre, le champ est vide et la confirmation est désactivée

#### Scenario: Ouverture depuis le drag
- **WHEN** l'utilisateur dépose une carte To Review sur la colonne In Progress
- **THEN** le même dialogue s'ouvre et la carte reste en To Review en attendant la confirmation

#### Scenario: Confirmation d'un retour valide
- **WHEN** l'utilisateur saisit un retour et confirme
- **THEN** la demande est envoyée, le dialogue se ferme et la carte quitte la colonne To Review

#### Scenario: Annulation après un drag
- **WHEN** l'utilisateur annule le dialogue ouvert par un drag
- **THEN** aucune requête n'est envoyée et la carte reste dans la colonne To Review

#### Scenario: Erreur à l'envoi
- **WHEN** le backend répond par une erreur à la demande de correction
- **THEN** le message d'erreur est affiché dans le dialogue et le texte saisi n'est pas perdu

### Requirement: Confirmation et suivi de l'approbation dans l'interface
Le bouton « Approuver & Fusionner » de l'onglet Actions et le drag d'une carte de To Review vers Done SHALL demander une confirmation nommant le change et la branche cible avant d'appeler l'approbation. Pendant l'appel, le bouton SHALL être désactivé avec un indicateur de chargement et les actions de revue du change SHALL NOT pouvoir être déclenchées à nouveau. En cas de succès, la carte SHALL passer en Done et le DetailPanel SHALL se mettre à jour. En cas d'échec, la carte SHALL rester (ou revenir) dans To Review et un message lisible, selon le code d'erreur renvoyé (conflit, validation en échec avec sa sortie, branche de base, merge en cours), SHALL être affiché sans fermer le DetailPanel.

#### Scenario: Confirmation avant fusion
- **WHEN** l'utilisateur clique sur « Approuver & Fusionner » ou dépose une carte To Review sur Done
- **THEN** une confirmation nommant le change et la branche cible s'affiche avant toute fusion

#### Scenario: Annulation de la confirmation
- **WHEN** l'utilisateur annule la confirmation
- **THEN** rien n'est fusionné et la carte reste dans To Review

#### Scenario: Fusion en cours
- **WHEN** l'approbation est en cours
- **THEN** le bouton est désactivé avec un indicateur de chargement

#### Scenario: Approbation réussie
- **WHEN** le backend répond `200`
- **THEN** la carte apparaît dans la colonne Done

#### Scenario: Approbation en échec
- **WHEN** le backend répond `409` avec le code `integration_conflict` ou `422` avec le code `validation_failed`
- **THEN** la carte reste dans To Review et l'onglet Actions affiche un message correspondant au code, avec la sortie de validation le cas échéant

### Requirement: Suppression d'un change en revue sans restes orphelins
Lorsque l'utilisateur supprime un change en statut `to-review` (après la confirmation existante), le backend SHALL aussi supprimer son worktree, sa branche `feature/<change>` et son marqueur de revue, en plus du dossier du change. Si cette suppression échoue, le change SHALL être conservé et l'erreur remontée. La suppression reste refusée tant qu'un worker est actif sur ce change.

#### Scenario: Suppression d'un change en revue
- **WHEN** l'utilisateur confirme la suppression de `add-user-auth`, en revue
- **THEN** le dossier du change, le worktree, la branche `feature/add-user-auth` et le marqueur de revue n'existent plus

#### Scenario: Échec du nettoyage
- **WHEN** le nettoyage du worktree ou de la branche échoue
- **THEN** le dossier du change n'est pas supprimé et le message d'erreur est affiché
