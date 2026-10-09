# change-review-actions Specification

## Purpose

Permet à l'utilisateur d'agir sur un change en revue (statut To Review) : l'approuver pour fusionner son travail, ou demander une correction qui relance le worker, depuis l'onglet Actions du DetailPanel ou par glisser-déposer, sans laisser de restes orphelins quand le change est supprimé.

## Requirements

### Requirement: Approuver et fusionner un change en revue
Le backend SHALL exposer `POST /api/workspaces/{id}/changes/{name}/review/approve`. Pour un change dont le statut est `to-review` et sans worker actif, il SHALL fusionner `feature/<change>` dans la branche actuellement extraite du dépôt (fusion `--no-ff`), supprimer ensuite le worktree puis la branche, et répondre `200` avec la branche cible. Le marqueur de revue disparaissant avec la branche, le change SHALL passer en `done` (le `tasks.md` fusionné est coché) et un événement `change_updated` SHALL être publié. L'approbation SHALL NOT supposer que le pool d'agents tourne et SHALL NOT lancer de session d'agent. Elle SHALL refuser avec `409` un change qui n'est pas en revue (code `not_in_review`) ou sur lequel un worker est actif (`worker_active`). En cas de refus ou d'échec survenant avant la fusion, la branche, le worktree, le travail committé et le marqueur de revue SHALL être conservés tels qu'avant l'appel, et la réponse SHALL porter un code d'erreur et un message lisibles. Si la fusion a eu lieu mais que la suppression du worktree ou de la branche échoue, le backend SHALL NOT présenter cette situation comme un échec : il SHALL lever le marqueur de revue (le change passe en `done`), publier `change_updated` et répondre `200` avec la branche cible et un avertissement `cleanup_incomplete` portant un message lisible et la liste `remaining` de ce qui reste à nettoyer à la main (parmi `worktree`, `branch` et `marker`). Si le marqueur ne peut lui-même pas être levé, `remaining` SHALL contenir `marker`, et une nouvelle approbation de ce change SHALL reconnaître que sa branche est déjà fusionnée dans la branche cible, sans nouvelle intégration, validation ni fusion, puis terminer le nettoyage.

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

#### Scenario: Fusion réussie mais worktree non supprimable
- **WHEN** l'utilisateur approuve `add-user-auth` alors que son worktree contient un fichier non suivi qui fait refuser la suppression du worktree, et que la fusion réussit
- **THEN** la réponse est `200` avec la branche cible et un avertissement `cleanup_incomplete` dont `remaining` contient `worktree`, le marqueur de revue est levé et le change passe en `done`

#### Scenario: Fusion réussie mais branche non supprimable
- **WHEN** la fusion réussit mais que la suppression de la branche `feature/add-user-auth` échoue
- **THEN** la réponse est `200` avec un avertissement `cleanup_incomplete` dont `remaining` contient `branch`, le marqueur est levé et le change passe en `done`

#### Scenario: Marqueur non levable après la fusion
- **WHEN** la fusion réussit puis que la levée du marqueur échoue
- **THEN** la réponse est `200` avec un avertissement `cleanup_incomplete` dont `remaining` contient `marker`, et une nouvelle approbation du même change reconnaît la branche comme déjà fusionnée, sans rejouer la validation ni fusionner, puis termine le nettoyage et lève le marqueur

### Requirement: Intégration et revalidation à l'approbation
À l'approbation, si la branche cible contient un commit absent de `feature/<change>`, le backend SHALL l'intégrer dans le worktree du change, puis SHALL rejouer la commande de validation du workspace sur le résultat, sans session d'agent ni tentative de guérison ; la fusion SHALL n'avoir lieu que si la validation réussit. Lorsque la branche cible ne contient aucun commit absent de `feature/<change>`, la validation SHALL NOT être rejouée. Le backend SHALL refuser de fusionner : si la branche de base enregistrée n'est pas la branche courante (code `base_branch_mismatch`), si un merge est déjà en cours dans le dépôt (code `merge_in_progress`), si l'intégration provoque un conflit (code `integration_conflict`, intégration annulée et worktree retrouvé tel qu'avant ; la réponse SHALL porter la branche cible `target` et la liste `files` des fichiers en conflit, relevée avant l'annulation, chemins relatifs à la racine du dépôt et triés, et SHALL NOT être vide de fichiers lorsque git en signale) ou si la validation échoue (code `validation_failed`, avec la sortie de validation). La fusion SHALL être sérialisée avec celles des workers du même workspace (verrou de fusion partagé) et SHALL NOT démarrer tant que la branche cible contient un commit absent du change au moment où le verrou est détenu : dans ce cas elle SHALL libérer le verrou et recommencer l'intégration, dans la limite de 3 intégrations successives, au-delà de laquelle elle répond `409` (code `target_moving`).

#### Scenario: Branche cible avancée sans conflit
- **WHEN** `main` a reçu un commit depuis la création de `feature/add-user-auth` et que l'utilisateur approuve le change
- **THEN** `main` est intégrée dans le worktree, la validation est rejouée, puis la fusion a lieu si elle réussit

#### Scenario: Branche cible inchangée
- **WHEN** `main` n'a reçu aucun commit depuis la création de la branche et que l'utilisateur approuve
- **THEN** aucune intégration ni validation n'a lieu et la fusion suit son cours

#### Scenario: Conflit d'intégration
- **WHEN** les commits de `main` entrent en conflit avec le travail de `feature/add-user-auth`
- **THEN** la réponse est `409` avec le code `integration_conflict`, l'intégration est annulée, la branche, le worktree et le marqueur sont inchangés et le change reste en To Review

#### Scenario: Fichiers en conflit exposés
- **WHEN** l'intégration de `main` dans `feature/add-user-auth` entre en conflit sur `src/a.tsx` et `src/b.tsx`
- **THEN** la réponse `409` porte `target` valant `main` et `files` valant `["src/a.tsx", "src/b.tsx"]`, et le worktree ne contient ni marqueur de conflit ni merge en cours

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
Le backend SHALL exposer `POST /api/workspaces/{id}/changes/{name}/review/request-correction` avec un corps `{ "feedback": "<texte>", "reopen_human_tasks": <booléen optionnel, faux par défaut> }`. Pour un change en revue et sans worker actif, il SHALL ajouter exactement une tâche décochée au `tasks.md` du worktree du change, dans une section `## Corrections` (créée si absente, réutilisée sinon), contenant l'intégralité du retour (y compris s'il est multi-lignes) sans créer d'autres tâches, puis committer ce fichier dans `feature/<change>` (lorsque `reopen_human_tasks` est vrai, la réouverture des tâches de validation humaine cochées, voir `human-review-tasks`, SHALL faire partie du même commit et SHALL NOT toucher les autres tâches), lever le marqueur de revue, publier `change_updated` et répondre `204`. Le retour SHALL être non vide après suppression des espaces, sinon la réponse est `400` et rien n'est modifié. Le retour de l'utilisateur SHALL survivre à l'arrêt du pool et du backend puisqu'il est versionné dans la branche. Si le worktree du change n'existe plus, le backend SHALL le recréer à partir de la branche avant d'écrire. En cas d'échec d'écriture ou de commit, le marqueur SHALL être conservé et la réponse porter un message lisible.

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

#### Scenario: Correction avec réouverture des tâches humaines
- **WHEN** l'utilisateur demande une correction avec `reopen_human_tasks` vrai pour un change en revue dont `tasks.md` contient `- [x] 4.2 Parcours manuel <!-- human review required -->` et `- [x] 1.1 Implémentation`
- **THEN** un seul commit ajoute la tâche de correction et repasse la ligne 4.2 en `- [ ]` avec son marqueur, la ligne 1.1 reste cochée, le marqueur de revue est levé et la réponse est `204`

#### Scenario: Correction sans réouverture
- **WHEN** la correction est demandée sans `reopen_human_tasks` ou avec `reopen_human_tasks` faux
- **THEN** aucune tâche existante n'est modifiée, comme avant

#### Scenario: Réouverture sans tâche humaine cochée
- **WHEN** `reopen_human_tasks` est vrai et qu'aucune tâche marquée n'est cochée
- **THEN** la correction est enregistrée normalement et aucune tâche n'est modifiée

#### Scenario: Échec du commit avec réouverture
- **WHEN** le commit échoue alors que `reopen_human_tasks` est vrai
- **THEN** le `tasks.md` du worktree retrouve son contenu d'origine, tâches humaines cochées comprises, et le marqueur de revue est conservé

### Requirement: Reprise du change après une demande de correction
Une fois le marqueur levé par une demande de correction, le change SHALL redevenir éligible au dispatcher (statut dérivé de l'avancement et de l'état « lancé »). Le worker qui le reprend SHALL réutiliser la branche et le worktree existants, appliquer les tâches restantes, dont la correction, avec le même flux que pour toute reprise (`/opsx:apply`, triage, validation, contrôle de complétion exigeant que toutes les tâches à faire par l'agent, corrections comprises, soient cochées : les tâches de validation humaine restées décochées ne comptent pas, voir `agent-pool-orchestrator`), puis repasser le change en revue en mode `hitl-review`. Si le pool est arrêté au moment de la demande, le change SHALL être repris au prochain démarrage du pool.

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

#### Scenario: Tâches de validation humaine conservées après correction
- **WHEN** une correction est demandée pour un change en revue dont des tâches de validation humaine sont décochées, et que le worker applique la correction
- **THEN** ces tâches restent décochées, le contrôle de complétion les ignore et le change repasse en To Review

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
Le bouton « Approuver & Fusionner » de l'onglet Actions et le drag d'une carte de To Review vers Done SHALL demander une confirmation nommant le change et la branche cible avant d'appeler l'approbation. Pendant l'appel, le bouton SHALL être désactivé avec un indicateur de chargement et les actions de revue du change SHALL NOT pouvoir être déclenchées à nouveau. En cas de succès, la carte SHALL passer en Done et le DetailPanel SHALL se mettre à jour. Si la réponse de succès porte un avertissement `cleanup_incomplete`, le dialogue SHALL se fermer comme pour un succès, aucune erreur SHALL être affichée, et un message d'avertissement, dans la langue de l'interface, SHALL indiquer ce qui reste à nettoyer à la main d'après `remaining`. En cas d'échec, la carte SHALL rester (ou revenir) dans To Review et un message lisible, selon le code d'erreur renvoyé (conflit, validation en échec avec sa sortie, branche de base, merge en cours), SHALL être affiché sans fermer le DetailPanel.

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

#### Scenario: Approbation réussie avec nettoyage incomplet
- **WHEN** le backend répond `200` avec un avertissement `cleanup_incomplete`
- **THEN** le dialogue se ferme, la carte apparaît dans la colonne Done, aucune erreur n'est affichée et un message d'avertissement, dans la langue de l'interface, précise ce qui reste à nettoyer d'après `remaining`

### Requirement: Suppression d'un change en revue sans restes orphelins
Lorsque l'utilisateur supprime un change en statut `to-review` (après la confirmation existante), le backend SHALL aussi supprimer son worktree, sa branche `feature/<change>` et son marqueur de revue, en plus du dossier du change. Si cette suppression échoue, le change SHALL être conservé et l'erreur remontée. La suppression reste refusée tant qu'un worker est actif sur ce change.

#### Scenario: Suppression d'un change en revue
- **WHEN** l'utilisateur confirme la suppression de `add-user-auth`, en revue
- **THEN** le dossier du change, le worktree, la branche `feature/add-user-auth` et le marqueur de revue n'existent plus

#### Scenario: Échec du nettoyage
- **WHEN** le nettoyage du worktree ou de la branche échoue
- **THEN** le dossier du change n'est pas supprimé et le message d'erreur est affiché

### Requirement: Approbation refusée tant que des tâches restent à valider
`POST /api/workspaces/{id}/changes/{name}/review/approve` SHALL refuser l'approbation avec `409` et le code `tasks_pending` tant que le `tasks.md` de `feature/<change>` contient au moins une tâche non cochée, qu'elle porte ou non le marqueur de validation humaine. La réponse SHALL porter le nombre de tâches non cochées (`remaining`) et un message lisible. Le refus SHALL être décidé avant toute intégration, validation ou fusion et SHALL conserver la branche, le worktree, le travail committé et le marqueur de revue tels qu'avant l'appel. Le contrôle SHALL être fait sous le verrou des actions de revue du change, de sorte qu'une coche concurrente soit prise en compte. Il SHALL NE PAS s'appliquer à une nouvelle approbation d'un change dont la branche est déjà fusionnée (reprise du nettoyage), ni à un change dont la branche n'a pas de `tasks.md` exploitable.

#### Scenario: Tâches restantes à valider
- **WHEN** l'approbation est demandée pour `add-user-auth`, en revue, dont la branche a 2 tâches décochées (dont une marquée)
- **THEN** la réponse est `409` avec le code `tasks_pending` et `remaining` valant 2, et rien n'est fusionné ni modifié

#### Scenario: Tâches toutes cochées
- **WHEN** l'utilisateur a coché toutes les tâches de la branche en revue puis demande l'approbation
- **THEN** l'approbation suit son déroulement habituel (intégration, validation, fusion)

#### Scenario: Tâche décochée par l'utilisateur
- **WHEN** l'utilisateur décoche une tâche faite par l'agent d'un change en revue, puis demande l'approbation
- **THEN** la réponse est `409` avec le code `tasks_pending` et `remaining` valant 1

#### Scenario: Branche déjà fusionnée
- **WHEN** l'approbation est rejouée pour un change dont la branche est déjà fusionnée dans la branche cible
- **THEN** le contrôle des tâches n'est pas appliqué et le nettoyage se termine

### Requirement: Approbation désactivée dans l'interface tant que des tâches restent à valider
Lorsque la liste des tâches d'un change en To Review contient au moins une tâche non cochée, le bouton « Approuver & Fusionner » de l'onglet Actions du DetailPanel SHALL rester visible mais désactivé, avec une info-bulle indiquant le nombre de tâches à valider (« N tâche(s) à valider » en français, « N task(s) to validate » en anglais). Le bouton SHALL redevenir actif dès que toutes les tâches sont cochées, sans rechargement de la page. Si le backend répond `409` avec le code `tasks_pending`, l'interface SHALL afficher un message lisible indiquant que des tâches restent à valider, sans fermer le DetailPanel ni déplacer la carte. Les libellés SHALL être disponibles en français et en anglais.

#### Scenario: Bouton désactivé avec tâches restantes
- **WHEN** l'utilisateur ouvre l'onglet Actions d'un change en revue qui a 2 tâches non cochées
- **THEN** « Approuver & Fusionner » est visible et désactivé, et son info-bulle indique « 2 tâches à valider »

#### Scenario: Bouton actif après avoir tout coché
- **WHEN** l'utilisateur coche la dernière tâche depuis l'onglet Tâches
- **THEN** « Approuver & Fusionner » devient actif sans rechargement

#### Scenario: Refus du backend
- **WHEN** le backend répond `409` avec le code `tasks_pending`
- **THEN** un message indique que des tâches restent à valider, la carte reste dans To Review et le DetailPanel reste ouvert

### Requirement: Résolution guidée d'un conflit d'intégration
Lorsque l'approbation d'un change en revue échoue avec `integration_conflict`, que ce soit depuis le bouton « Approuver & Fusionner » du DetailPanel ou depuis le glisser-déposer d'une carte vers Done, le dialogue d'approbation SHALL rester ouvert avec le message d'erreur, afficher sous ce message la liste `files` des fichiers en conflit et la branche cible, et proposer l'action « Demander la résolution au worker ». Cette action SHALL ouvrir le dialogue de correction existant, avec un texte initial **prérempli et modifiable** nommant la branche cible, demandant d'intégrer cette branche dans la branche du change, de résoudre les conflits listés en conservant les évolutions des deux côtés et de relancer la validation. Elle SHALL NE RIEN envoyer au backend avant la confirmation de l'utilisateur dans ce dialogue ; l'annulation du dialogue SHALL laisser le change en revue et ne rien modifier. La confirmation SHALL suivre le circuit de la demande de correction (tâche committée, marqueur levé, reprise par un worker). L'action SHALL NOT être proposée pour les autres codes d'erreur de l'approbation. Les libellés SHALL être disponibles en français et en anglais.

#### Scenario: Conflit depuis l'onglet Actions
- **WHEN** l'utilisateur confirme « Approuver & Fusionner » sur `add-user-auth` et que le backend répond `409` `integration_conflict` avec `target` `main` et `files` `["src/a.tsx", "src/b.tsx"]`
- **THEN** le dialogue reste ouvert, affiche le message d'erreur, les fichiers `src/a.tsx` et `src/b.tsx`, la branche `main` et l'action « Demander la résolution au worker »

#### Scenario: Conflit depuis le glisser-déposer vers Done
- **WHEN** l'utilisateur dépose une carte To Review sur Done, confirme, et que l'approbation répond `integration_conflict`
- **THEN** le même dialogue affiche les fichiers et l'action, et la carte reste en To Review

#### Scenario: Dialogue de correction prérempli
- **WHEN** l'utilisateur clique sur « Demander la résolution au worker »
- **THEN** le dialogue de correction s'ouvre avec un texte initial qui nomme `main` et les fichiers en conflit, le champ est modifiable et rien n'a été envoyé au backend

#### Scenario: Texte ajusté par l'utilisateur
- **WHEN** l'utilisateur ajoute une consigne au texte prérempli puis confirme
- **THEN** le retour envoyé est le texte modifié, intégralement

#### Scenario: Annulation du dialogue prérempli
- **WHEN** l'utilisateur annule le dialogue de correction ouvert depuis un conflit
- **THEN** aucune requête de correction n'est envoyée et le change reste en To Review

#### Scenario: Autres erreurs d'approbation
- **WHEN** l'approbation échoue avec `validation_failed`, `target_moving` ou `base_branch_mismatch`
- **THEN** l'action « Demander la résolution au worker » n'est pas proposée

### Requirement: Réouverture des tâches de validation humaine dans le dialogue de correction
Le dialogue de correction SHALL afficher la case « Rouvrir les N tâche(s) de validation humaine » (« Reopen the N human validation task(s) » en anglais) lorsque la liste des tâches du change contient au moins une tâche de validation humaine cochée, N étant leur nombre ; il SHALL NE PAS l'afficher sinon. La valeur de la case SHALL être envoyée au backend sous la forme `reopen_human_tasks`. Elle SHALL être **cochée par défaut** lorsque le dialogue est ouvert par la résolution guidée d'un conflit d'intégration et **décochée par défaut** pour toute autre ouverture (bouton « Demander correction », glisser-déposer vers In Progress). L'utilisateur SHALL pouvoir modifier la case avant de confirmer. Une fois la correction traitée et le change revenu en To Review, les tâches rouvertes SHALL apparaître décochées avec leur badge « Validation humaine » et « Approuver & Fusionner » SHALL rester désactivé jusqu'à ce qu'elles soient cochées (voir « Approbation désactivée dans l'interface tant que des tâches restent à valider »).

#### Scenario: Case cochée par défaut pour un conflit
- **WHEN** le dialogue de correction s'ouvre depuis la résolution guidée d'un change qui a une tâche de validation humaine cochée
- **THEN** la case « Rouvrir la tâche de validation humaine » est affichée et cochée

#### Scenario: Case décochée par défaut pour une correction ordinaire
- **WHEN** l'utilisateur clique sur « Demander correction » pour le même change
- **THEN** la case est affichée et décochée, et une confirmation sans y toucher n'envoie pas `reopen_human_tasks` à vrai

#### Scenario: Pas de tâche humaine cochée
- **WHEN** aucune tâche de validation humaine du change n'est cochée
- **THEN** la case n'est pas affichée

#### Scenario: Case modifiée par l'utilisateur
- **WHEN** l'utilisateur décoche la case ouverte par un conflit puis confirme
- **THEN** la demande est envoyée avec `reopen_human_tasks` faux et les tâches humaines cochées restent cochées

#### Scenario: Revalidation exigée après la reprise
- **WHEN** la correction a été envoyée avec la case cochée, que le worker l'a appliquée et que le change est revenu en To Review
- **THEN** la tâche rouverte est décochée avec son badge et « Approuver & Fusionner » est désactivé avec l'info-bulle « 1 tâche à valider »
