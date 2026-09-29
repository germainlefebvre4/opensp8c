# Spec Delta

## MODIFIED Requirements

### Requirement: Onglet de navigation Agents
La vue de visibilité globale, tous workspaces confondus, des agents et pools d'agents actifs SHALL être exposée dans la section "Agent Pool" de la page Configuration, indépendante du workspace actuellement sélectionné, et non plus comme un onglet séparé de la navigation principale liée au workspace.

#### Scenario: Accès à l'onglet depuis n'importe quel workspace
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool alors qu'il consultait le Kanban ou les Specs d'un workspace quelconque
- **THEN** l'application affiche la vue Agent Pool, listant les pools actifs de tous les workspaces, indépendamment du workspace précédemment actif

### Requirement: Liste des pools et workers actifs tous workspaces confondus
La vue Agent Pool SHALL regrouper les workers actifs par pool d'origine. Pour chaque pool, l'affichage SHALL indiquer le nom du workspace auquel il appartient, sa taille configurée sous la forme "X/Y workers actifs", et son mode de délégation. Pour chaque worker actif de ce pool, l'affichage SHALL indiquer le nom du changement (tâche Kanban) qu'il traite, son statut, un aperçu de l'activité en cours de l'agent, sa durée d'exécution depuis son démarrage, et la raison du blocage lorsque son statut est `paused`. Cette liste SHALL se limiter aux pools et workers actuellement actifs, sans historique des exécutions passées.

#### Scenario: Plusieurs pools actifs sur des workspaces différents
- **WHEN** un pool de taille 3 avec 2 workers actifs tourne sur le workspace `A` et un pool de taille 1 avec 1 worker actif tourne sur le workspace `B`
- **THEN** la vue Agent Pool affiche les deux pools regroupés, chacun avec le nom de son workspace, sa taille ("2/3" et "1/1"), son mode de délégation, et le détail de ses workers actifs (changement en cours, statut, aperçu d'activité, durée)

#### Scenario: Aucun pool actif
- **WHEN** aucun pool n'est en cours d'exécution sur aucun workspace
- **THEN** la vue Agent Pool affiche un état vide indiquant qu'aucun agent n'est actuellement actif

#### Scenario: Mise à jour de la liste
- **WHEN** un pool démarre, s'arrête, ou qu'un de ses workers change de statut, d'activité ou de changement traité pendant que l'utilisateur consulte la vue Agent Pool
- **THEN** la vue Agent Pool reflète ce changement sans que l'utilisateur ait à recharger la page manuellement

#### Scenario: Worker en pause avec raison affichée
- **WHEN** un worker actif d'un pool passe au statut `paused` (par exemple après épuisement des tentatives de guérison)
- **THEN** la vue Agent Pool affiche, pour ce worker, la raison lisible de son blocage

### Requirement: Navigation vers le workspace depuis la liste
Depuis la vue Agent Pool, l'utilisateur SHALL pouvoir cliquer sur la ligne d'un worker pour naviguer directement vers le Kanban du workspace correspondant. La vue Agent Pool SHALL rester une vue de consultation : elle ne SHALL PAS permettre de démarrer ou d'arrêter un pool directement.

#### Scenario: Navigation vers le Kanban d'un workspace listé
- **WHEN** l'utilisateur clique sur la ligne d'un worker appartenant au workspace `A` dans la vue Agent Pool
- **THEN** l'application affiche le Kanban du workspace `A`
