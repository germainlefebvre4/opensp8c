# Spec Delta

## MODIFIED Requirements

### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher les changements OpenSpec répartis en six slots horizontaux d'égale largeur : **To Explore**, **Ready**, **To Do**, **In Progress**, **To Review**, et **Done/Archived**. Le slot **Done/Archived** contient verticalement la colonne Done (en haut, `flex-1 min-h-0`, prioritaire sur l'espace vertical) et la colonne Archived (en bas, hauteur plafonnée à 40 % du slot via `max-h`, avec scroll interne). Les colonnes actives lisent depuis `openspec/changes/` (hors `archive/`). La colonne Archived lit depuis `openspec/changes/archive/` via un endpoint dédié. L'endpoint `/changes` SHALL inclure les champs `days_since_activity` (int), `is_stale` (bool), et `tags` (objet optionnel `{ type, complexity, components[] }`) pour chaque change actif. Pour un change dont un worker du pool est actif (y compris en pause), la progression (`tasks_done`, `tasks_total`) et la colonne SHALL être dérivées du `tasks.md` du worktree du worker, et non de celui du dépôt principal ; la colonne SHALL alors être plafonnée à **In Progress**.

#### Scenario: Chargement du Kanban avec changements archivés
- **WHEN** l'utilisateur ouvre le Kanban Board ou change de workspace actif
- **THEN** l'application charge les changements actifs depuis `/workspaces/{id}/changes` et les changements archivés depuis `/workspaces/{id}/archived-changes`, et les affiche dans leurs colonnes respectives — Done en haut du slot (priorité flex), Archived en bas (hauteur plafonnée), et To Review entre In Progress et Done/Archived

#### Scenario: Réponse API avec champs stale et tags
- **WHEN** l'endpoint `/workspaces/{id}/changes` retourne la liste des changes
- **THEN** chaque change inclut `days_since_activity` (entier, -1 si pas de tasks.md), `is_stale` (booléen), et `tags` (objet optionnel ou `null`)

#### Scenario: Rétrécissement vertical — Done conserve son espace
- **WHEN** la hauteur disponible du Kanban diminue (bottom panel ouvert, fenêtre réduite)
- **THEN** la colonne Archived se compresse en premier (dans la limite de son plafond), Done conserve l'espace résiduel disponible

#### Scenario: Chargement du Kanban sans changements archivés
- **WHEN** le répertoire `openspec/changes/archive/` est vide ou absent
- **THEN** la colonne Archived est affichée vide en bas du slot Done, sans erreur

#### Scenario: Changement sans tasks.md
- **WHEN** un changement n'a pas de fichier `tasks.md` (ou tasks_total == 0)
- **THEN** il est affiché dans la colonne **To Explore**

#### Scenario: Changement avec tasks non démarrées et non lancé
- **WHEN** un changement a un `tasks.md` avec des tasks mais aucune cochée (`tasks_done == 0`), et n'est pas marqué "lancé" (voir `kanban-ready-column`)
- **THEN** il est affiché dans la colonne **Ready**

#### Scenario: Changement avec tasks non démarrées et lancé
- **WHEN** un changement a un `tasks.md` avec des tasks mais aucune cochée (`tasks_done == 0`), et est marqué "lancé"
- **THEN** il est affiché dans la colonne **To Do**

#### Scenario: Changement partiellement complété
- **WHEN** un changement a au moins une task cochée mais pas toutes
- **THEN** il est affiché dans la colonne **In Progress**

#### Scenario: Changement entièrement complété en mode Autonomie Totale
- **WHEN** toutes les tasks d'un changement sont cochées ET qu'aucun worker du pool n'est actif sur ce change (merge effectué en mode `full-autonomy`, ou changement hors exécution du pool)
- **THEN** il est affiché dans la colonne **Done**

#### Scenario: Changement entièrement complété en mode Review Humaine
- **WHEN** toutes les tasks d'un changement ont été complétées par un worker du pool d'agents exécuté en mode `hitl-review`

#### Scenario: Progression en direct pendant le run d'un worker
- **WHEN** un worker est actif sur un change et que l'agent a coché 2 tâches sur 5 dans le `tasks.md` du worktree, alors que celui du dépôt principal n'en a aucune cochée
- **THEN** le change est affiché dans la colonne **In Progress** avec `tasks_done = 2` et `tasks_total = 5`, et son badge worker reste visible

#### Scenario: Change entièrement coché dans le worktree avant le merge
- **WHEN** un worker est actif sur un change dont toutes les tâches sont cochées dans le `tasks.md` du worktree, la validation ou le merge n'étant pas terminé
- **THEN** le change reste dans la colonne **In Progress** avec une progression de 100 %, et ne passe à **Done** qu'une fois le worker libéré après le merge

#### Scenario: Worker actif sans tâche cochée dans le worktree
- **WHEN** un worker est actif sur un change lancé dont aucune tâche n'est cochée dans le worktree
- **THEN** le change est affiché dans la colonne **To Do** avec son badge worker

#### Scenario: Worktree sans tasks.md exploitable
- **WHEN** un worker est actif sur un change mais que le worktree n'est pas encore provisionné, ou que son `tasks.md` est absent ou ne contient aucune tâche
- **THEN** la progression et la colonne sont dérivées du `tasks.md` du dépôt principal

#### Scenario: Fin du worker sans merge
- **WHEN** un worker est annulé ou libéré sans que son travail ait été fusionné
- **THEN** la progression et la colonne du change redeviennent celles du `tasks.md` du dépôt principal
