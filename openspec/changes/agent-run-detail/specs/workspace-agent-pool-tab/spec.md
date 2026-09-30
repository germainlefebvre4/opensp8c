# Spec Delta

## MODIFIED Requirements

### Requirement: Détail des workers du pool du workspace actif
L'onglet Agents scopé au workspace actif SHALL afficher, pour chaque worker actuellement actif du pool de ce workspace, le nom du changement (tâche Kanban) traité, son statut, un aperçu de l'activité en cours de l'agent, sa durée d'exécution depuis son démarrage, et la raison du blocage lorsque son statut est `paused`. Chaque ligne de worker SHALL être sélectionnable pour ouvrir le détail de son run (voir `agent-run-detail`).

#### Scenario: Affichage des workers actifs du workspace sélectionné
- **WHEN** le pool du workspace actif a une taille de 3 et que 2 workers y sont actuellement actifs
- **THEN** l'onglet Agents affiche ces 2 workers, chacun avec sa tâche, son statut, un aperçu de son activité en cours et sa durée d'exécution

#### Scenario: Aucun pool actif sur le workspace sélectionné
- **WHEN** aucun pool n'est en cours d'exécution sur le workspace actuellement sélectionné
- **THEN** la liste des workers actifs affiche un état vide indiquant qu'aucun agent n'est actif sur ce workspace, et la section des runs récents reste affichée avec les runs passés du workspace

#### Scenario: Worker en pause avec raison affichée
- **WHEN** un worker du pool du workspace actif passe au statut `paused` après avoir épuisé ses tentatives de guérison
- **THEN** l'onglet Agents affiche, pour ce worker, la raison lisible de son blocage

#### Scenario: Sélection d'une ligne de worker
- **WHEN** l'utilisateur clique sur la ligne d'un worker actif
- **THEN** la ligne est marquée comme sélectionnée et le panneau de détail de son run s'ouvre

### Requirement: Vue de consultation sans contrôle de pool
L'onglet Agents scopé au workspace actif SHALL rester une vue de consultation : il ne SHALL PAS permettre de démarrer ou d'arrêter le pool, ni d'agir individuellement sur un worker (arrêt, relance, envoi de message). La sélection d'un worker ou d'un run pour en consulter le détail en lecture seule n'est pas une action sur le worker. Les actions de pool restent réservées au panneau de contrôle du pool accessible depuis le Kanban du workspace.

#### Scenario: Aucune action de pool proposée
- **WHEN** l'utilisateur consulte l'onglet Agents du workspace actif, qu'un pool y soit actif ou non
- **THEN** aucun bouton de démarrage, d'arrêt de pool, ou d'action individuelle sur un worker n'est proposé sur cette vue ni dans le panneau de détail

#### Scenario: Consultation du détail sans effet sur le worker
- **WHEN** l'utilisateur sélectionne un worker actif puis ferme le panneau de détail
- **THEN** l'état du worker et du pool est inchangé

## ADDED Requirements

### Requirement: Rafraîchissement automatique et runs récents dans l'onglet Agents
L'onglet Agents SHALL se mettre à jour automatiquement lorsque le pool ou l'un de ses workers change d'état, et SHALL afficher la section « Runs récents » ainsi que le panneau de détail définis par la capacité `agent-run-detail`.

#### Scenario: Statut de worker mis à jour sans navigation préalable vers le Kanban
- **WHEN** l'utilisateur ouvre directement l'onglet Agents et qu'un worker passe de `working` à `testing`
- **THEN** le statut affiché pour ce worker passe à `testing` sans rechargement de la page
