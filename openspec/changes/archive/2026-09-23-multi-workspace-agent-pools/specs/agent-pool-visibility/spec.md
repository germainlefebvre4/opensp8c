# Spec Delta

## Purpose

Donne une vue centralisée et en lecture seule de tous les agents et pools d'agents actuellement actifs à travers l'ensemble des workspaces, pour savoir en un coup d'œil qui travaille sur quoi et où.

## ADDED Requirements

### Requirement: Onglet de navigation Agents
L'interface utilisateur SHALL inclure un onglet "Agents" dans la navigation principale, au même niveau que les onglets Kanban, Specs et Timeline, accessible indépendamment du workspace actuellement sélectionné.

#### Scenario: Accès à l'onglet depuis n'importe quel workspace
- **WHEN** l'utilisateur clique sur l'onglet "Agents" alors qu'il consulte le Kanban ou les Specs d'un workspace quelconque
- **THEN** l'application affiche la vue Agents, indépendamment du workspace précédemment actif

### Requirement: Liste des pools et workers actifs tous workspaces confondus
La vue Agents SHALL afficher, pour chaque worker actuellement actif sur n'importe quel workspace, le nom du workspace auquel il appartient, le nom du changement (tâche Kanban) qu'il traite, son statut, le mode de délégation du pool auquel il appartient, et sa durée d'exécution depuis son démarrage. Cette liste SHALL se limiter aux pools et workers actuellement actifs, sans historique des exécutions passées.

#### Scenario: Plusieurs pools actifs sur des workspaces différents
- **WHEN** un pool avec 2 workers actifs tourne sur le workspace `A` et un pool avec 1 worker actif tourne sur le workspace `B`
- **THEN** la vue Agents affiche les 3 workers, chacun avec le nom de son workspace, le changement en cours, son statut, son mode de délégation, et sa durée d'exécution

#### Scenario: Aucun pool actif
- **WHEN** aucun pool n'est en cours d'exécution sur aucun workspace
- **THEN** la vue Agents affiche un état vide indiquant qu'aucun agent n'est actuellement actif

#### Scenario: Mise à jour de la liste
- **WHEN** un pool démarre, s'arrête, ou qu'un de ses workers change de statut ou de changement traité pendant que l'utilisateur consulte la vue Agents
- **THEN** la vue Agents reflète ce changement sans que l'utilisateur ait à recharger la page manuellement

### Requirement: Navigation vers le workspace depuis la liste
Depuis la vue Agents, l'utilisateur SHALL pouvoir cliquer sur une ligne de la liste pour naviguer directement vers le Kanban du workspace correspondant. La vue Agents SHALL rester une vue de consultation : elle ne SHALL PAS permettre de démarrer ou d'arrêter un pool directement.

#### Scenario: Navigation vers le Kanban d'un workspace listé
- **WHEN** l'utilisateur clique sur la ligne d'un worker appartenant au workspace `A` dans la vue Agents
- **THEN** l'application affiche le Kanban du workspace `A`
