# Spec Delta

## MODIFIED Requirements

### Requirement: Onglet Agents scopé au workspace actif
L'onglet "Agents" du sous-menu du workspace SHALL être lié au workspace actuellement sélectionné, au même titre que les onglets Kanban, Specs et Timeline. Lorsqu'aucun workspace n'est sélectionné, il SHALL afficher le même état vide "aucun workspace" que les autres onglets liés au workspace, plutôt qu'une liste tous workspaces confondus.

#### Scenario: Changement de workspace met à jour la liste affichée
- **WHEN** l'utilisateur navigue du Kanban du workspace `A` vers l'onglet Agents, puis sélectionne le workspace `B` dans la sidebar
- **THEN** l'onglet Agents affiche uniquement les workers du pool actif du workspace `B`, sans aucun worker appartenant au workspace `A`

#### Scenario: Aucun workspace sélectionné
- **WHEN** aucun workspace n'est configuré ou sélectionné et que l'utilisateur ouvre l'onglet Agents
- **THEN** l'application affiche l'état vide "aucun workspace", comme pour les onglets Kanban, Specs et Timeline
