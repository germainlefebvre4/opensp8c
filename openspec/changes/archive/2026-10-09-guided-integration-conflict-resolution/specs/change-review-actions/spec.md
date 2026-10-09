# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

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
