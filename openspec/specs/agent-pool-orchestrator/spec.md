# Agent Pool Orchestrator Specification

## Purpose

Orchestre un pool d'agents autonomes qui exécutent en parallèle les changements OpenSpec de la colonne To Do d'un workspace, chacun isolé dans son propre git worktree, avec une boucle d'auto-correction sur échec de validation.

## Requirements

### Requirement: Configuration du pool d'agents par workspace
Le backend SHALL exposer un mécanisme de configuration du pool d'agents propre à chaque workspace. Cette configuration comprend le nombre maximum d'agents parallèles (`size`, de 1 à 5), le mode de délégation (`delegation_mode`, `full-autonomy` ou `hitl-review`) et le nombre maximal de tentatives d'auto-correction (`max_attempts`), appliqués uniquement aux workers lancés pour ce workspace. Cette configuration SHALL être persistée : des défauts globaux sont définis dans Configuration et chaque workspace peut les surcharger champ par champ dans Settings. Au démarrage d'un pool, la configuration effective SHALL être résolue en prenant, pour chaque champ, la valeur du workspace, sinon le défaut global, sinon la valeur par défaut de la plateforme ; une configuration passée explicitement dans la requête de démarrage SHALL l'emporter sur cette résolution pour ce lancement.

#### Scenario: Configuration valide du pool d'un workspace
- **WHEN** le client demande à configurer le pool du workspace `A` avec une taille de 3 et le mode `hitl-review`
- **THEN** le backend stocke et applique cette configuration pour tous les futurs workers lancés par le pool de ce workspace, sans affecter la configuration ou l'exécution du pool d'un autre workspace

#### Scenario: Résolution sans requête explicite
- **WHEN** un pool est démarré pour le workspace `A` sans configuration explicite, que le défaut global fixe la taille à 2 et que `A` surcharge le mode à `full-autonomy`
- **THEN** le pool de `A` démarre avec une taille de 2 et le mode `full-autonomy`

#### Scenario: Configuration explicite dans la requête
- **WHEN** un pool est démarré avec une taille de 5 dans la requête alors que la configuration résolue donnerait 2
- **THEN** le pool démarre avec une taille de 5 pour ce lancement sans modifier les valeurs persistées

### Requirement: Exécution concurrente de pools sur plusieurs workspaces
Le backend SHALL permettre l'exécution simultanée d'un pool d'agents par workspace, chaque pool étant démarré, arrêté et suivi indépendamment des pools des autres workspaces.

#### Scenario: Démarrage de pools sur deux workspaces différents
- **WHEN** un pool est déjà en cours d'exécution sur le workspace `A` et que le client demande le démarrage d'un pool sur le workspace `B`
- **THEN** le backend démarre le pool du workspace `B` avec succès, sans interrompre ni modifier le pool en cours sur le workspace `A`

#### Scenario: Refus de double démarrage sur le même workspace
- **WHEN** un pool est déjà en cours d'exécution sur le workspace `A` et que le client redemande le démarrage d'un pool sur ce même workspace `A`
- **THEN** le backend refuse la demande et retourne une erreur indiquant qu'un pool est déjà actif pour ce workspace

### Requirement: Visibilité globale des pools actifs
Le backend SHALL exposer un moyen de lister, en une seule requête, l'ensemble des pools actuellement actifs à travers tous les workspaces, regroupés par pool d'origine. Pour chaque pool, cette information SHALL inclure l'identifiant et le nom du workspace, la taille configurée du pool (`size`), et le mode de délégation. Pour chaque worker actif d'un pool, cette information SHALL inclure le nom du changement (tâche Kanban) en cours de traitement, son statut, un aperçu de l'activité en cours (dernière sortie produite par sa session), et sa date de démarrage.

#### Scenario: Liste des pools actifs sur plusieurs workspaces
- **WHEN** un pool de taille 3 avec 2 workers actifs tourne sur le workspace `A` et un pool de taille 1 avec 1 worker actif tourne sur le workspace `B`
- **THEN** la liste globale retourne les deux pools regroupés, chacun associé à l'identifiant et au nom de son workspace d'origine, à sa taille configurée et à son mode de délégation, avec le détail de ses workers actifs (changement traité, statut, aperçu d'activité, date de démarrage)

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

### Requirement: Application des réglages de rôle aux workers
Le pool SHALL lancer chaque worker avec l'agent, le modèle et l'effort résolus pour le rôle `implementer` du workspace, et SHALL utiliser le rôle `fixer` pour le subprocess démarré lors d'une relance après « Demander des corrections ». Les tours d'auto-guérison d'un worker partageant son subprocess d'implémentation, ils SHALL conserver les réglages de l'`implementer`.

#### Scenario: Worker avec réglages de rôle
- **WHEN** le rôle `implementer` du workspace `A` définit l'agent `claude`, le modèle `sonnet` et l'effort `medium`
- **THEN** le subprocess d'un worker de `A` est lancé avec ces valeurs

#### Scenario: Relance après demande de corrections
- **WHEN** un worker est relancé après « Demander des corrections »
- **THEN** le nouveau subprocess utilise les réglages du rôle `fixer`

#### Scenario: Tours d'auto-guérison
- **WHEN** la validation échoue et que le worker réinjecte l'erreur dans son subprocess
- **THEN** aucun nouveau subprocess n'est démarré et le modèle reste celui de l'`implementer`

### Requirement: Journalisation de l'exécution de chaque worker
Chaque worker du pool SHALL consigner son exécution dans un run persistant de kind `pool` (voir `agent-run-detail`) tout au long de la vie de son subprocess d'agent, y compris les tours de correction envoyés lors de la boucle d'auto-guérison. Cette journalisation ne SHALL PAS modifier le déroulement de l'exécution : un échec d'écriture du journal SHALL NOT interrompre ni mettre en pause le worker.

#### Scenario: Tours de guérison consignés
- **WHEN** la validation échoue et que le worker renvoie les erreurs à l'agent par un tour de suivi
- **THEN** ce tour et la réponse de l'agent sont ajoutés au même run que la tentative initiale

#### Scenario: Échec d'écriture du journal
- **WHEN** l'écriture d'une ligne dans le run du worker échoue
- **THEN** le worker poursuit son exécution normalement

### Requirement: État des workers cohérent sous concurrence
Le statut et l'aperçu d'activité d'un worker SHALL être lus et écrits de façon synchronisée avec les instantanés retournés par le statut du pool, de sorte qu'un instantané ne présente jamais un état partiellement mis à jour ni ne provoque de course de données.

#### Scenario: Instantané pendant une mise à jour de l'activité
- **WHEN** le statut du pool est demandé pendant que le worker met à jour son aperçu d'activité
- **THEN** la réponse contient un état cohérent du worker et le détecteur de courses de données ne signale rien
