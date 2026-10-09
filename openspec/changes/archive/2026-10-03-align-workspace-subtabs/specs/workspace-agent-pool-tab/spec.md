# Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

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
