# Spec Delta

## MODIFIED Requirements

### Requirement: Visibilité globale des pools actifs
Le backend SHALL exposer un moyen de lister, en une seule requête, l'ensemble des pools actuellement actifs à travers tous les workspaces, regroupés par pool d'origine. Pour chaque pool, cette information SHALL inclure l'identifiant et le nom du workspace, la taille configurée du pool (`size`), et le mode de délégation. Pour chaque worker actif d'un pool, cette information SHALL inclure le nom du changement (tâche Kanban) en cours de traitement, son statut, un aperçu de l'activité en cours (dernière sortie produite par sa session), et sa date de démarrage.

#### Scenario: Liste des pools actifs sur plusieurs workspaces
- **WHEN** un pool de taille 3 avec 2 workers actifs tourne sur le workspace `A` et un pool de taille 1 avec 1 worker actif tourne sur le workspace `B`
- **THEN** la liste globale retourne les deux pools regroupés, chacun associé à l'identifiant et au nom de son workspace d'origine, à sa taille configurée et à son mode de délégation, avec le détail de ses workers actifs (changement traité, statut, aperçu d'activité, date de démarrage)

#### Scenario: Aucun pool actif
- **WHEN** aucun pool n'est en cours d'exécution sur aucun workspace
- **THEN** la liste globale retourne une liste vide

## ADDED Requirements

### Requirement: Raison de blocage lisible sur les workers en pause
Lorsqu'un worker passe au statut `paused`, le backend SHALL renseigner sur ce worker un message lisible décrivant la cause du blocage. Ce message SHALL être exposé à la fois par l'endpoint de statut de pool par workspace et par l'endpoint de liste globale des pools.

#### Scenario: Échec de démarrage du subprocess agent
- **WHEN** un worker ne parvient pas à démarrer le subprocess de l'agent CLI configuré
- **THEN** son statut passe à `paused` et sa raison de blocage décrit l'échec de démarrage du subprocess

#### Scenario: Échec de l'invocation d'application
- **WHEN** l'invocation de l'agent pour appliquer les tâches restantes de `tasks.md` échoue
- **THEN** son statut passe à `paused` et sa raison de blocage décrit l'échec de cette invocation

#### Scenario: Épuisement des tentatives de guérison
- **WHEN** l'agent n'arrive pas à résoudre l'erreur de build/tests après le nombre maximal de tentatives configuré (`max_attempts`)
- **THEN** son statut passe à `paused` et sa raison de blocage indique que les tentatives de guérison sont épuisées

#### Scenario: Tâches restantes malgré une validation réussie
- **WHEN** la validation (build/tests) réussit pour un changement mais que `tasks.md` contient encore des tâches non cochées
- **THEN** son statut passe à `paused` et sa raison de blocage indique que des tâches restent incomplètes malgré la validation réussie
