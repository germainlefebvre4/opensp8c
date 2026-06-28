## MODIFIED Requirements

### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher les changements OpenSpec répartis en quatre slots horizontaux d'égale largeur : **To Explore**, **To Do**, **In Progress**, et **Done/Archived**. Le slot **Done/Archived** contient verticalement la colonne Done (en haut, prenant l'espace restant) et la colonne Archived (en bas, hauteur auto limitée à ses cartes visibles). Les colonnes actives lisent depuis `openspec/changes/` (hors `archive/`). La colonne Archived lit depuis `openspec/changes/archive/` via un endpoint dédié. La règle de calcul du statut est inchangée pour les colonnes actives.

#### Scenario: Chargement du Kanban avec changements archivés
- **WHEN** l'utilisateur ouvre le Kanban Board ou change de workspace actif
- **THEN** l'application charge les changements actifs depuis `/workspaces/{id}/changes` et les changements archivés depuis `/workspaces/{id}/archived-changes`, et les affiche dans leurs colonnes respectives — Done en haut du slot, Archived en bas

#### Scenario: Chargement du Kanban sans changements archivés
- **WHEN** le répertoire `openspec/changes/archive/` est vide ou absent
- **THEN** la colonne Archived est affichée vide en bas du slot Done, sans erreur

#### Scenario: Changement sans tasks.md
- **WHEN** un changement n'a pas de fichier `tasks.md` (ou tasks_total == 0)
- **THEN** il est affiché dans la colonne **To Explore**

#### Scenario: Changement avec tasks non démarrées
- **WHEN** un changement a un `tasks.md` avec des tasks mais aucune cochée (`tasks_done == 0`)
- **THEN** il est affiché dans la colonne **To Do**

#### Scenario: Changement partiellement complété
- **WHEN** un changement a au moins une task cochée mais pas toutes
- **THEN** il est affiché dans la colonne **In Progress**

#### Scenario: Changement entièrement complété
- **WHEN** toutes les tasks d'un changement sont cochées (`tasks_done == tasks_total > 0`)
- **THEN** il est affiché dans la colonne **Done**

### Requirement: Séparateur visuel entre Done et Archived
Un séparateur visuel horizontal SHALL être rendu entre les colonnes **Done** et **Archived** à l'intérieur du slot partagé, pour signaler la frontière entre changements actifs et archivés.

#### Scenario: Rendu du séparateur horizontal
- **WHEN** le Kanban est affiché
- **THEN** une ligne horizontale tenue sépare visuellement la section Done de la section Archived dans le même slot de colonne
