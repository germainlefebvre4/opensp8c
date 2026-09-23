# Spec Delta

## REMOVED Requirements

### Requirement: Configuration globale du pool d'agents
**Reason**: La configuration du pool n'est plus un mécanisme global unique partagé par tous les workspaces ; elle devient une configuration indépendante par workspace pour permettre l'exécution concurrente de plusieurs pools.
**Migration**: Voir le nouveau requirement "Configuration du pool d'agents par workspace", qui applique la même configuration (`size`, `delegation_mode`) mais de façon isolée à chaque workspace.

## ADDED Requirements

### Requirement: Configuration du pool d'agents par workspace
Le backend SHALL exposer un mécanisme de configuration du pool d'agents propre à chaque workspace. Cette configuration comprend le nombre maximum d'agents parallèles (`size`, de 1 à 5) et le mode de délégation (`delegation_mode`, `full-autonomy` ou `hitl-review`), appliqués uniquement aux workers lancés pour ce workspace.

#### Scenario: Configuration valide du pool d'un workspace
- **WHEN** le client demande à configurer le pool du workspace `A` avec une taille de 3 et le mode `hitl-review`
- **THEN** le backend stocke et applique cette configuration pour tous les futurs workers lancés par le pool de ce workspace, sans affecter la configuration ou l'exécution du pool d'un autre workspace

### Requirement: Exécution concurrente de pools sur plusieurs workspaces
Le backend SHALL permettre l'exécution simultanée d'un pool d'agents par workspace, chaque pool étant démarré, arrêté et suivi indépendamment des pools des autres workspaces.

#### Scenario: Démarrage de pools sur deux workspaces différents
- **WHEN** un pool est déjà en cours d'exécution sur le workspace `A` et que le client demande le démarrage d'un pool sur le workspace `B`
- **THEN** le backend démarre le pool du workspace `B` avec succès, sans interrompre ni modifier le pool en cours sur le workspace `A`

#### Scenario: Refus de double démarrage sur le même workspace
- **WHEN** un pool est déjà en cours d'exécution sur le workspace `A` et que le client redemande le démarrage d'un pool sur ce même workspace `A`
- **THEN** le backend refuse la demande et retourne une erreur indiquant qu'un pool est déjà actif pour ce workspace

### Requirement: Visibilité globale des pools actifs
Le backend SHALL exposer un moyen de lister, en une seule requête, l'ensemble des pools actuellement actifs à travers tous les workspaces, ainsi que leurs workers. Pour chaque worker actif, cette information SHALL inclure l'identifiant et le nom du workspace auquel il appartient, le nom du changement (tâche Kanban) en cours de traitement, et son statut.

#### Scenario: Liste des pools actifs sur plusieurs workspaces
- **WHEN** un pool avec 2 workers actifs tourne sur le workspace `A` et un pool avec 1 worker actif tourne sur le workspace `B`
- **THEN** la liste globale retourne les 3 workers, chacun associé à l'identifiant et au nom de son workspace d'origine et au changement qu'il traite

#### Scenario: Aucun pool actif
- **WHEN** aucun pool n'est en cours d'exécution sur aucun workspace
- **THEN** la liste globale retourne une liste vide
