# Spec Delta

## MODIFIED Requirements

### Requirement: Dispatcher de dépendances basé sur un DAG
Le dispatcher de tâches du backend SHALL lire les dépendances déclarées dans le fichier `.openspec.yaml` de chaque changement situé dans la colonne **Todo** du Kanban. Il SHALL construire un Graphe Dirigé Acyclique (DAG) et distribuer en parallèle uniquement les changements n'ayant pas de dépendances actives en attente de traitement. Lorsque plusieurs changements runnables sont disponibles pour un nombre de workers libres inférieur au nombre de changements éligibles, le dispatcher SHALL les distribuer par ordre croissant de leur rang de priorité persisté (`order`, voir `kanban-ready-column`), les rangs les plus bas étant distribués en premier.

#### Scenario: Distribution parallèle sans dépendance commune
- **WHEN** les changements A et B sont dans la colonne Todo et n'ont aucune dépendance l'un envers l'autre
- **THEN** le dispatcher distribue A et B en parallèle à deux workers libres différents

#### Scenario: Ordonnancement séquentiel avec dépendance déclarée
- **WHEN** le changement B déclare dépendre de A, et que les deux sont dans la colonne Todo
- **THEN** le dispatcher lance uniquement le changement A, et n'ordonnance le changement B qu'après la transition réussie de A vers son état final de validation

#### Scenario: Priorité entre changements runnables sans dépendance commune
- **WHEN** les changements A (rang de priorité 2) et B (rang de priorité 1) sont tous deux runnables dans la colonne Todo et qu'un seul worker est disponible
- **THEN** le dispatcher lance B en premier, celui-ci ayant le rang de priorité le plus bas
