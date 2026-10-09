# Workspace Agent Pool Tab Specification

## Purpose

Donne, depuis la navigation principale, une vue en lecture seule limitée au workspace actuellement sélectionné de son agent pool actif, pour savoir en un coup d'œil qui y travaille sans ouvrir le Kanban.

## Requirements

### Requirement: Onglet Agents scopé au workspace actif
L'onglet "Agents" du sous-menu du workspace SHALL être lié au workspace actuellement sélectionné, au même titre que les onglets Kanban, Specs et Timeline. Lorsqu'aucun workspace n'est sélectionné, il SHALL afficher le même état vide "aucun workspace" que les autres onglets liés au workspace, plutôt qu'une liste tous workspaces confondus.

#### Scenario: Changement de workspace met à jour la liste affichée
- **WHEN** l'utilisateur navigue du Kanban du workspace `A` vers l'onglet Agents, puis sélectionne le workspace `B` dans la sidebar
- **THEN** l'onglet Agents affiche uniquement les workers du pool actif du workspace `B`, sans aucun worker appartenant au workspace `A`

#### Scenario: Aucun workspace sélectionné
- **WHEN** aucun workspace n'est configuré ou sélectionné et que l'utilisateur ouvre l'onglet Agents
- **THEN** l'application affiche l'état vide "aucun workspace", comme pour les onglets Kanban, Specs et Timeline

### Requirement: Détail des workers du pool du workspace actif
Le sous-onglet « Workers » de l'onglet Agents scopé au workspace actif SHALL afficher, pour chaque worker actuellement actif du pool de ce workspace, le nom du changement (tâche Kanban) traité, son statut, un aperçu de l'activité en cours de l'agent, sa durée d'exécution depuis son démarrage, et la raison du blocage lorsque son statut est `paused`. Chaque ligne de worker SHALL être sélectionnable pour ouvrir le détail de son run (voir `agent-run-detail`).

#### Scenario: Affichage des workers actifs du workspace sélectionné
- **WHEN** le pool du workspace actif a une taille de 3 et que 2 workers y sont actuellement actifs
- **THEN** le sous-onglet Workers affiche ces 2 workers, chacun avec sa tâche, son statut, un aperçu de son activité en cours et sa durée d'exécution

#### Scenario: Aucun pool actif sur le workspace sélectionné
- **WHEN** aucun pool n'est en cours d'exécution sur le workspace actuellement sélectionné
- **THEN** la liste des workers actifs affiche un état vide indiquant qu'aucun agent n'est actif sur ce workspace, et le sous-onglet « Runs » reste disponible avec les runs passés du workspace

#### Scenario: Worker en pause avec raison affichée
- **WHEN** un worker du pool du workspace actif passe au statut `paused` après avoir épuisé ses tentatives de guérison
- **THEN** le sous-onglet Workers affiche, pour ce worker, la raison lisible de son blocage

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

### Requirement: Rafraîchissement automatique et runs récents dans l'onglet Agents
L'onglet Agents SHALL se mettre à jour automatiquement lorsque le pool ou l'un de ses workers change d'état, et SHALL afficher la section « Runs récents » ainsi que le panneau de détail définis par la capacité `agent-run-detail`.

#### Scenario: Statut de worker mis à jour sans navigation préalable vers le Kanban
- **WHEN** l'utilisateur ouvre directement l'onglet Agents et qu'un worker passe de `working` à `testing`
- **THEN** le statut affiché pour ce worker passe à `testing` sans rechargement de la page
### Requirement: Sous-onglets Workers et Runs de l'onglet Agents
L'onglet Agents d'un workspace SHALL afficher la barre de sous-onglets commune des pages du workspace (voir `workspace-page-subtabs`) avec deux sous-onglets : « Workers », qui affiche le résumé du pool et la table des workers actifs, et « Runs », qui affiche la section des runs récents (voir `agent-run-detail`). Le sous-onglet « Workers » SHALL être actif par défaut. Le sous-onglet actif SHALL être reflété dans le paramètre d'URL `tab` (valeurs `workers` et `runs`), l'absence de paramètre valant « Workers ». Lorsque l'URL contient un paramètre `run` valide et aucun paramètre `tab` valide, le sous-onglet « Runs » SHALL être actif. Un paramètre `tab` explicite SHALL toujours l'emporter sur cette règle.

#### Scenario: Ouverture de l'onglet Agents
- **WHEN** l'utilisateur ouvre l'onglet Agents sans paramètre `tab` ni `run`
- **THEN** les sous-onglets « Workers » et « Runs » sont affichés, « Workers » est actif et la liste des workers du workspace est visible

#### Scenario: Bascule vers Runs
- **WHEN** l'utilisateur clique sur le sous-onglet « Runs »
- **THEN** la section des runs récents est affichée à la place de la table des workers et l'URL porte `tab=runs`

#### Scenario: Retour à Workers
- **WHEN** l'utilisateur est sur « Runs » puis clique sur « Workers »
- **THEN** la table des workers est affichée et l'URL ne porte plus de paramètre `tab` ni `tab=runs`

#### Scenario: Rechargement de la page sur Runs
- **WHEN** l'utilisateur recharge une page dont l'URL contient `tab=runs`
- **THEN** le sous-onglet « Runs » est actif après le rechargement

#### Scenario: Lien direct vers un run
- **WHEN** l'URL contient `run=<change>/<ts>` sans paramètre `tab`
- **THEN** le sous-onglet « Runs » est actif et le panneau de détail s'ouvre sur ce run

#### Scenario: Sélection d'un worker reste sur Workers
- **WHEN** l'utilisateur, sur le sous-onglet « Workers », clique sur la ligne d'un worker actif
- **THEN** le panneau de détail s'ouvre sur son run et le sous-onglet « Workers » reste actif

#### Scenario: Valeur de tab inconnue
- **WHEN** l'URL contient un paramètre `tab` dont la valeur n'est ni `workers` ni `runs`
- **THEN** ce paramètre est ignoré et le sous-onglet est déterminé comme s'il était absent

#### Scenario: Pas de titre de page
- **WHEN** l'onglet Agents est affiché
- **THEN** aucun titre « Agents » n'est affiché au-dessus de la barre de sous-onglets
