# Spec Delta

## MODIFIED Requirements

### Requirement: Badges de statut Kanban dans le sidebar
Le `WorkspaceSidebar` SHALL afficher, pour chaque workspace, une barre segmentée indiquant la répartition de ses changes par statut Kanban, avec le nombre total de changes à son extrémité. Chaque statut dont le compteur est supérieur à 0 (`to-explore`, `ready`, `todo`, `in-progress`, `verifying`, `to-review`, `done`) SHALL y figurer comme un segment de largeur proportionnelle à son compteur, dans l'ordre du Kanban, avec la couleur du dot de sa colonne Kanban ; les statuts à 0 SHALL être omis. Chaque segment SHALL exposer son libellé et son nombre en infobulle. Les couleurs de statut de la sidebar et du Kanban SHALL provenir d'une même source.

#### Scenario: Workspace avec changes actifs
- **WHEN** un workspace a des changes dans plusieurs statuts
- **THEN** la barre affiche un segment par statut non nul, dans l'ordre to-explore, ready, todo, in-progress, verifying, to-review, done, avec le total à son extrémité

#### Scenario: Workspace sans changes actifs
- **WHEN** un workspace n'a aucun change
- **THEN** la barre est affichée vide, avec un total de 0, et aucun segment n'est affiché (statuts à 0 masqués)

#### Scenario: Statuts ready et verifying
- **WHEN** un workspace a des changes `ready` ou `verifying`
- **THEN** la barre affiche un segment indigo pour `ready` et un segment teal pour `verifying`, les couleurs des colonnes correspondantes du Kanban

#### Scenario: Infobulle d'un segment
- **WHEN** l'utilisateur survole le segment `in-progress` d'un workspace qui a 5 changes en cours
- **THEN** une infobulle indique le libellé du statut et le nombre 5

#### Scenario: Mise à jour des badges
- **WHEN** un changement de statut Kanban survient dans un workspace
- **THEN** la barre de la sidebar se met à jour dans les 15 secondes suivantes
