# Agent Pool Orchestrator Specification

## Purpose

Orchestre un pool d'agents autonomes qui exécutent en parallèle les changements OpenSpec de la colonne To Do d'un workspace, chacun isolé dans son propre git worktree, avec une boucle d'auto-correction sur échec de validation.

## Requirements

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

### Requirement: Isolation par Git Worktree pour les workers du pool
Pour chaque changement en cours d'exécution en parallèle, le backend SHALL provisionner un répertoire de travail temporaire isolé en utilisant la commande `git worktree`. Chaque worker de l'agent pool SHALL exécuter son processus d'agent de façon autonome et isolée au sein de ce worktree afin d'éviter tout conflit de fichiers.

#### Scenario: Provisionnement et exécution isolée
- **WHEN** un worker prend en charge le changement `add-user-auth`
- **THEN** le backend crée une branche `feature/add-user-auth`, l'associe à un nouveau git worktree temporaire sous `.opensp8c/worktrees/wt-add-user-auth`, et lance le subprocess de l'agent en définissant son répertoire de travail sur ce dossier

#### Scenario: Nettoyage après merge ou rejet
- **WHEN** le changement `add-user-auth` est approuvé ou entièrement annulé
- **THEN** le backend supprime le git worktree correspondant de l'arborescence et supprime la branche locale si demandé

### Requirement: Boucle d'auto-correction (Self-Healing Loop) des workers
Chaque worker exécutant un changement SHALL invoquer un agent CLI réel (et non simulé) au sein d'une session non-interactive unique par tentative, chargée d'exécuter les tâches restantes du `tasks.md`. Après cette invocation, le worker SHALL exécuter la commande de validation de test ou de compilation. En cas d'échec, le worker SHALL ré-injecter les logs d'erreurs dans le contexte du modèle en poursuivant la même session d'agent par un tour de suivi, jusqu'à un maximum configurable de tentatives.

#### Scenario: Invocation réelle de l'agent sur un changement Todo
- **WHEN** un worker prend en charge le changement `add-user-auth`
- **THEN** le backend lance une session non-interactive de l'agent configuré pour ce workspace/changement, dans le worktree isolé du changement, avec pour instruction d'implémenter les tâches restantes de `tasks.md`

#### Scenario: Auto-correction réussie sur erreur de compilation
- **WHEN** l'agent modifie un fichier provoquant une erreur de compilation Go, et que la commande de build échoue
- **THEN** le worker extrait l'erreur de compilation, la transmet à l'agent dans un message système, et l'agent génère un correctif qui réussit la compilation au deuxième essai

#### Scenario: Échec de correction après tentatives maximales
- **WHEN** l'agent n'arrive pas à résoudre l'erreur de build après 3 tentatives consécutives
- **THEN** le worker marque l'état comme bloqué, arrête l'exécution de ce changement pour demander l'aide de l'utilisateur, et libère le worker pour d'autres tâches indépendantes

### Requirement: Vérification de complétion avant finalisation
Après une validation réussie (build/tests OK), le worker SHALL vérifier que `tasks.md` ne contient plus aucune tâche non cochée avant de finaliser le changement (fusion automatique en mode `full-autonomy`, ou transition vers l'état de revue en mode `hitl-review`). Si des tâches restent non cochées malgré une validation réussie, le worker SHALL NOT finaliser le changement et SHALL passer son propre statut à `paused` pour signaler qu'une intervention est nécessaire.

#### Scenario: Finalisation refusée si des tâches restent ouvertes
- **WHEN** la validation (tests/build) réussit pour le changement `add-user-auth` mais que `tasks.md` contient encore des tâches non cochées
- **THEN** le worker ne fusionne pas la branche et ne fait pas transitionner le changement vers l'état suivant ; il passe son propre statut à `paused`

#### Scenario: Finalisation autorisée quand toutes les tâches sont cochées
- **WHEN** la validation réussit et que toutes les tâches de `tasks.md` sont cochées
- **THEN** le worker finalise le changement selon le mode de délégation configuré (fusion en `full-autonomy`, transition vers revue en `hitl-review`)
