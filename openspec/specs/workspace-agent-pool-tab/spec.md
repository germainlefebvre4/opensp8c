# Workspace Agent Pool Tab Specification

## Purpose

Donne, depuis la navigation principale, une vue en lecture seule limitée au workspace actuellement sélectionné de son agent pool actif, pour savoir en un coup d'œil qui y travaille sans ouvrir le Kanban.

## Requirements

### Requirement: Onglet Agents scopé au workspace actif
L'onglet "Agents" de la navigation principale SHALL être lié au workspace actuellement sélectionné, au même titre que les onglets Kanban, Specs et Timeline. Lorsqu'aucun workspace n'est sélectionné, il SHALL afficher le même état vide "aucun workspace" que les autres onglets liés au workspace, plutôt qu'une liste tous workspaces confondus.

#### Scenario: Changement de workspace met à jour la liste affichée
- **WHEN** l'utilisateur navigue du Kanban du workspace `A` vers l'onglet Agents, puis sélectionne le workspace `B` dans la sidebar
- **THEN** l'onglet Agents affiche uniquement les workers du pool actif du workspace `B`, sans aucun worker appartenant au workspace `A`

#### Scenario: Aucun workspace sélectionné
- **WHEN** aucun workspace n'est configuré ou sélectionné et que l'utilisateur ouvre l'onglet Agents
- **THEN** l'application affiche l'état vide "aucun workspace", comme pour les onglets Kanban, Specs et Timeline

### Requirement: Détail des workers du pool du workspace actif
L'onglet Agents scopé au workspace actif SHALL afficher, pour chaque worker actuellement actif du pool de ce workspace, le nom du changement (tâche Kanban) traité, son statut, un aperçu de l'activité en cours de l'agent, sa durée d'exécution depuis son démarrage, et la raison du blocage lorsque son statut est `paused`.

#### Scenario: Affichage des workers actifs du workspace sélectionné
- **WHEN** le pool du workspace actif a une taille de 3 et que 2 workers y sont actuellement actifs
- **THEN** l'onglet Agents affiche ces 2 workers, chacun avec sa tâche, son statut, un aperçu de son activité en cours et sa durée d'exécution

#### Scenario: Aucun pool actif sur le workspace sélectionné
- **WHEN** aucun pool n'est en cours d'exécution sur le workspace actuellement sélectionné
- **THEN** l'onglet Agents affiche un état vide indiquant qu'aucun agent n'est actif sur ce workspace

#### Scenario: Worker en pause avec raison affichée
- **WHEN** un worker du pool du workspace actif passe au statut `paused` après avoir épuisé ses tentatives de guérison
- **THEN** l'onglet Agents affiche, pour ce worker, la raison lisible de son blocage

### Requirement: Vue de consultation sans contrôle de pool
L'onglet Agents scopé au workspace actif SHALL rester une vue de consultation : il ne SHALL PAS permettre de démarrer ou d'arrêter le pool, ni d'agir individuellement sur un worker. Ces actions restent réservées au panneau de contrôle du pool accessible depuis le Kanban du workspace.

#### Scenario: Aucune action de pool proposée
- **WHEN** l'utilisateur consulte l'onglet Agents du workspace actif, qu'un pool y soit actif ou non
- **THEN** aucun bouton de démarrage, d'arrêt de pool, ou d'action individuelle sur un worker n'est proposé sur cette vue
