## ADDED Requirements

### Requirement: Configuration globale du pool d'agents
Le backend SHALL exposer un mécanisme de configuration globale pour le pool d'agents. Cette configuration comprend le nombre maximum d'agents parallèles (`size`) et le mode de délégation unifié (`delegation_mode`), qui peut être `full-autonomy` ou `hitl-review`.

#### Scenario: Configuration valide du pool
- **WHEN** le client demande à configurer le pool avec une taille de 3 et le mode `hitl-review`
- **THEN** le backend stocke et applique cette configuration pour tous les futurs workers lancés par le pool

### Requirement: Dispatcher de dépendances basé sur un DAG
Le dispatcher de tâches du backend SHALL lire les dépendances déclarées dans le fichier `.openspec.yaml` de chaque changement situé dans la colonne **Todo** du Kanban. Il SHALL construire un Graphe Dirigé Acyclique (DAG) et distribuer en parallèle uniquement les changements n'ayant pas de dépendances actives en attente de traitement.

#### Scenario: Distribution parallèle sans dépendance commune
- **WHEN** les changements A et B sont dans la colonne Todo et n'ont aucune dépendance l'un envers l'autre
- **THEN** le dispatcher distribue A et B en parallèle à deux workers libres différents

#### Scenario: Ordonnancement séquentiel avec dépendance déclarée
- **WHEN** le changement B déclare dépendre de A, et que les deux sont dans la colonne Todo
- **THEN** le dispatcher lance uniquement le changement A, et n'ordonnance le changement B qu'après la transition réussie de A vers son état final de validation

### Requirement: Isolation par Git Worktree pour les workers du pool
Pour chaque changement en cours d'exécution en parallèle, le backend SHALL provisionner un répertoire de travail temporaire isolé en utilisant la commande `git worktree`. Chaque worker de l'agent pool SHALL exécuter son processus d'agent de façon autonome et isolée au sein de ce worktree afin d'éviter tout conflit de fichiers.

#### Scenario: Provisionnement et exécution isolée
- **WHEN** un worker prend en charge le changement `add-user-auth`
- **THEN** le backend crée une branche `feature/add-user-auth`, l'associe à un nouveau git worktree temporaire sous `.opensp8c/worktrees/wt-add-user-auth`, et lance le subprocess de l'agent en définissant son répertoire de travail sur ce dossier

#### Scenario: Nettoyage après merge ou rejet
- **WHEN** le changement `add-user-auth` est approuvé ou entièrement annulé
- **THEN** le backend supprime le git worktree correspondant de l'arborescence et supprime la branche locale si demandé

### Requirement: Boucle d'auto-correction (Self-Healing Loop) des workers
Chaque worker exécutant un changement SHALL suivre une boucle continue d'exécution des tâches du `tasks.md`. Après chaque modification de code pour une tâche donnée, le worker SHALL exécuter la commande de validation de test ou de compilation. En cas d'échec, le worker SHALL ré-injecter les logs d'erreurs dans le contexte du modèle pour lui permettre de s'auto-corriger, jusqu'à un maximum configurable de tentatives.

#### Scenario: Auto-correction réussie sur erreur de compilation
- **WHEN** l'agent modifie un fichier provoquant une erreur de compilation Go, et que la commande de build échoue
- **THEN** le worker extrait l'erreur de compilation, la transmet à l'agent dans un message système, et l'agent génère un correctif qui réussit la compilation au deuxième essai

#### Scenario: Échec de correction après tentatives maximales
- **WHEN** l'agent n'arrive pas à résoudre l'erreur de build après 3 tentatives consécutives
- **THEN** le worker marque l'état comme bloqué, arrête l'exécution de ce changement pour demander l'aide de l'utilisateur, et libère le worker pour d'autres tâches indépendantes
