# Spec Delta

## MODIFIED Requirements

### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher les changements OpenSpec répartis en six slots horizontaux d'égale largeur : **To Explore**, **Ready**, **To Do**, **In Progress**, **To Review**, et **Done/Archived**. Le slot **Done/Archived** contient verticalement la colonne Done (en haut, `flex-1 min-h-0`, prioritaire sur l'espace vertical) et la colonne Archived (en bas, hauteur plafonnée à 40 % du slot via `max-h`, avec scroll interne). Les colonnes actives lisent depuis `openspec/changes/` (hors `archive/`). La colonne Archived lit depuis `openspec/changes/archive/` via un endpoint dédié. L'endpoint `/changes` SHALL inclure les champs `days_since_activity` (int), `is_stale` (bool), et `tags` (objet optionnel `{ type, complexity, components[] }`) pour chaque change actif. Un changement portant un marqueur de revue (voir `change-review-state`) SHALL être affiché dans la colonne **To Review**, quel que soit l'avancement de ses tasks dans le workspace principal.

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
- **WHEN** toutes les tasks d'un changement sont cochées ET que le pool d'agents s'exécute en mode `full-autonomy` (ou hors exécution du pool)
- **THEN** il est affiché dans la colonne **Done**

#### Scenario: Changement entièrement complété en mode Review Humaine
- **WHEN** un worker du pool d'agents exécuté en mode `hitl-review` termine un changement avec l'issue `awaiting-review` (travail committé dans `feature/<change>`), alors que le `tasks.md` du workspace principal n'est pas encore coché
- **THEN** il est affiché dans la colonne **To Review** (et non dans **To Do**) et n'est déplacé vers **Done** qu'après approbation de l'utilisateur et fusion de sa branche

#### Scenario: Changement en revue après redémarrage
- **WHEN** le pool d'agents ou le backend est arrêté puis redémarré alors qu'un changement est en attente de revue
- **THEN** il reste affiché dans la colonne **To Review**
