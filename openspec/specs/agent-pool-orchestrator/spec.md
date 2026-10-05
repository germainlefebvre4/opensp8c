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
Pour chaque changement en cours d'exécution en parallèle, le backend SHALL provisionner un répertoire de travail temporaire isolé en utilisant la commande `git worktree`. Chaque worker de l'agent pool SHALL exécuter son processus d'agent de façon autonome et isolée au sein de ce worktree afin d'éviter tout conflit de fichiers. L'emplacement du worktree SHALL être propre au workspace (`.opensp8c/worktrees/<identifiant du workspace>/wt-<change>`) de sorte que deux workspaces ayant un changement de même nom n'utilisent jamais le même répertoire. Le backend SHALL NOT supprimer un worktree contenant des modifications non committées, sauf annulation explicite par l'utilisateur, et SHALL NOT supprimer un répertoire qui n'est pas un worktree enregistré du dépôt du workspace.

#### Scenario: Provisionnement et exécution isolée
- **WHEN** un worker prend en charge le changement `add-user-auth` du workspace `A`
- **THEN** le backend crée une branche `feature/add-user-auth`, l'associe à un nouveau git worktree temporaire sous `.opensp8c/worktrees/<identifiant de A>/wt-add-user-auth`, et lance le subprocess de l'agent en définissant son répertoire de travail sur ce dossier

#### Scenario: Changements de même nom dans deux workspaces
- **WHEN** les workspaces `A` et `B` ont chacun un changement `shared-change` pris en charge par un worker
- **THEN** chaque worker dispose de son propre worktree, sans collision de répertoire, et le nettoyage de l'un n'affecte jamais l'autre

#### Scenario: Nettoyage après merge ou rejet
- **WHEN** le changement `add-user-auth` a été fusionné avec succès, ou que l'utilisateur l'annule explicitement
- **THEN** le backend supprime le git worktree correspondant et la branche locale ; hors annulation explicite, un worktree contenant des modifications non committées n'est pas supprimé

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
Après une validation réussie (build/tests OK), le worker SHALL vérifier que `tasks.md` existe dans le worktree, contient au moins une tâche et ne contient plus aucune tâche non cochée avant de finaliser le changement (fusion automatique en mode `full-autonomy`, ou transition vers l'état de revue en mode `hitl-review`). Si `tasks.md` est absent ou ne contient aucune tâche, ou si des tâches restent non cochées malgré une validation réussie, le worker SHALL NOT finaliser le changement et SHALL passer son propre statut à `paused` pour signaler qu'une intervention est nécessaire.

#### Scenario: Finalisation refusée si des tâches restent ouvertes
- **WHEN** la validation (tests/build) réussit pour le changement `add-user-auth` mais que `tasks.md` contient encore des tâches non cochées
- **THEN** le worker ne fusionne pas la branche et ne fait pas transitionner le changement vers l'état suivant ; il passe son propre statut à `paused`

#### Scenario: Finalisation refusée si tasks.md est absent ou vide
- **WHEN** la validation réussit mais que `tasks.md` est absent du worktree ou ne contient aucune tâche
- **THEN** le worker ne finalise pas, passe à `paused` et sa raison de blocage indique que la liste des tâches est absente ou vide

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

### Requirement: Commande de validation résolue par workspace
Après chaque invocation de l'agent, le worker SHALL exécuter la commande de validation du workspace dans le worktree de son changement. Cette commande SHALL être résolue ainsi : le réglage `validationCommand` du workspace s'il est défini, sinon le défaut global de Configuration, sinon le résultat de l'auto-détection. L'auto-détection SHALL examiner la racine du worktree puis chacun de ses sous-répertoires directs (hors répertoires cachés et `node_modules`) et SHALL retenir, pour chaque répertoire concerné, `go test ./...` lorsqu'il contient un `go.mod` et `npm test` lorsque son `package.json` déclare un script `test` et que ses dépendances sont installées (répertoire `node_modules` présent). Un répertoire dont le `package.json` déclare un script `test` mais dont les dépendances ne sont pas installées SHALL être considéré comme un projet détecté mais non validable : l'auto-détection SHALL NOT l'ignorer silencieusement, et la validation SHALL être traitée comme une erreur d'environnement (voir « Pause immédiate sur erreur d'environnement de validation »), y compris lorsque d'autres commandes ont été détectées par ailleurs. Chaque commande détectée SHALL être exécutée dans le répertoire où elle a été détectée, et la validation SHALL réussir uniquement si toutes les commandes réussissent. Une commande explicitement configurée SHALL être exécutée à la racine du worktree et SHALL l'emporter sur toute détection, y compris sur le constat de projet non validable.

#### Scenario: Module Go dans un sous-répertoire
- **WHEN** le worktree du changement `add-user-auth` contient `backend/go.mod` et `frontend/package.json` avec un script `test` et un répertoire `frontend/node_modules`, et qu'aucune commande de validation n'est configurée
- **THEN** le worker exécute `go test ./...` dans `backend/` et `npm test` dans `frontend/`, et la validation réussit uniquement si les deux réussissent

#### Scenario: Projet Node sans dépendances installées
- **WHEN** `frontend/package.json` déclare un script `test` mais que `frontend/node_modules` n'existe pas dans le worktree, que `backend/go.mod` existe et qu'aucune commande de validation n'est configurée
- **THEN** le worker n'exécute aucune commande, passe à `paused` sans tour de guérison, et sa raison de blocage nomme `frontend` comme projet non validable faute de dépendances installées et invite à configurer une commande de validation

#### Scenario: Projet Node sans dépendances et commande configurée
- **WHEN** `frontend/package.json` déclare un script `test`, `frontend/node_modules` n'existe pas, et le workspace définit `validationCommand` par `cd backend && go test ./... && cd ../frontend && npm ci && npm test`
- **THEN** le worker exécute uniquement cette commande à la racine du worktree et ne met pas le worker en pause pour projet non validable

#### Scenario: Module Go à la racine
- **WHEN** la racine du worktree contient un `go.mod` et qu'aucune commande de validation n'est configurée
- **THEN** le worker exécute `go test ./...` à la racine du worktree

#### Scenario: Commande configurée
- **WHEN** le workspace surcharge `validationCommand` par `make test` et que le worktree contient aussi un `go.mod`
- **THEN** le worker exécute `make test` à la racine du worktree et n'exécute aucune commande auto-détectée

#### Scenario: Défaut global de la commande
- **WHEN** Configuration définit `validationCommand` et que le workspace n'a aucune surcharge
- **THEN** le worker de ce workspace exécute la commande définie dans Configuration

#### Scenario: Échec de tests d'un module
- **WHEN** l'une des commandes de validation retourne un code d'erreur après avoir démarré
- **THEN** la validation échoue et sa sortie est renvoyée à l'agent dans un tour de guérison

### Requirement: Pause immédiate sur erreur d'environnement de validation
Lorsque la validation ne peut pas être exécutée pour une raison d'environnement, c'est-à-dire lorsque l'auto-détection ne trouve aucune commande, que l'auto-détection trouve un projet Node dont les dépendances ne sont pas installées, qu'une commande configurée est vide ou introuvable, ou que son répertoire d'exécution n'existe pas, le worker SHALL passer immédiatement au statut `paused` avec une raison de blocage lisible, sans lancer de tour d'auto-guérison ni consommer de tentative. Un échec de test ou de compilation d'une commande qui a démarré SHALL continuer à suivre la boucle d'auto-guérison.

#### Scenario: Aucune commande de validation détectable
- **WHEN** aucune commande n'est configurée et que l'auto-détection ne trouve ni `go.mod` ni `package.json` avec script `test` dans le worktree
- **THEN** le worker passe à `paused` sans tour de guérison, et sa raison de blocage indique qu'aucune commande de validation n'a été détectée et invite à en configurer une

#### Scenario: Projet détecté mais non validable
- **WHEN** aucune commande n'est configurée et que `frontend/package.json` déclare un script `test` sans `frontend/node_modules` dans le worktree
- **THEN** le worker passe à `paused` sans tour de guérison et sans consommer de tentative, avant toute exécution de commande de validation, et sa raison de blocage nomme le répertoire concerné

#### Scenario: Commande configurée introuvable
- **WHEN** la commande de validation configurée référence un exécutable absent de la machine
- **THEN** le worker passe à `paused` sans tour de guérison, et sa raison de blocage nomme la commande introuvable

#### Scenario: Échec de tests toujours guérissable
- **WHEN** une commande de validation démarre puis échoue parce qu'un test est rouge
- **THEN** le worker lance un tour de guérison et n'est pas mis en pause avant l'épuisement de `max_attempts`

### Requirement: Reprise de la branche et du worktree existants
Lorsqu'un worker prend en charge un changement dont la branche `feature/<change>` existe déjà, le backend SHALL réutiliser cette branche et, si le worktree du changement existe déjà, ce worktree tel quel, sans recréer la branche ni écraser les modifications non commitées qu'il contient. Un worktree créé à l'ancien emplacement `.opensp8c/worktrees/wt-<change>` (sans segment de workspace) qui est un worktree enregistré du dépôt du workspace sur cette branche SHALL être réutilisé tel quel. L'existence de la branche SHALL être déterminée par le résultat de git et non par le contenu de sa sortie. Ce comportement s'applique à toute reprise d'un changement : reprise après pause, relance après « Demander des corrections », redémarrage du pool.

Par exception, une branche périmée SHALL être recréée : lorsque le worktree réutilisé ne contient pas `tasks.md` du changement alors que celui-ci est committé dans la branche courante du dépôt, et que la branche `feature/<change>` ne porte aucun travail (worktree sans modification et aucun commit absent de la branche courante), le backend SHALL supprimer ce worktree et cette branche sans forcer, les recréer à partir de la branche courante et ré-enregistrer la branche de base. Une branche qui porte du travail SHALL NOT être supprimée ni réécrite.

#### Scenario: Reprise après pause
- **WHEN** un worker précédemment mis en pause pour le changement `add-user-auth` est repris alors que `feature/add-user-auth` et son worktree existent
- **THEN** le worker réutilise ce worktree avec ses modifications non commitées et n'échoue pas sur l'existence de la branche

#### Scenario: Branche existante sans worktree
- **WHEN** `feature/add-user-auth` existe mais que son worktree a été supprimé
- **THEN** le backend crée un nouveau worktree sur cette branche existante, sans tenter de recréer la branche

#### Scenario: Première prise en charge
- **WHEN** aucune branche `feature/add-user-auth` n'existe
- **THEN** le backend crée la branche et le worktree comme pour un nouveau changement

#### Scenario: Worktree à l'ancien emplacement
- **WHEN** `feature/add-user-auth` est extraite dans un worktree enregistré du dépôt à l'ancien emplacement `.opensp8c/worktrees/wt-add-user-auth`
- **THEN** le backend réutilise ce worktree tel quel avec ses modifications non commitées, sans créer de second worktree ni échouer parce que la branche est déjà extraite

#### Scenario: Branche périmée sans travail recréée
- **WHEN** `feature/add-user-auth` a été créée avant que le changement soit committé, que son worktree est propre sans commit propre, et que le changement est depuis committé dans la branche courante
- **THEN** le backend recrée la branche et le worktree à partir de la branche courante, le worktree contient `openspec/changes/add-user-auth/tasks.md`, et le worker lance l'agent normalement

#### Scenario: Branche périmée avec travail conservée
- **WHEN** `feature/add-user-auth` ne contient pas le changement mais porte des modifications non commitées ou des commits propres
- **THEN** le backend ne supprime ni ne réécrit la branche ni le worktree et le worker passe à `paused` (voir « Présence du change dans le worktree »)

### Requirement: Échec de provisionnement visible
Lorsque le provisionnement de la branche ou du worktree d'un changement échoue, le worker SHALL passer au statut `paused` avec une raison de blocage lisible reprenant la cause de l'échec, au lieu d'échouer silencieusement.

#### Scenario: Provisionnement en échec
- **WHEN** `git worktree add` échoue pour le changement `add-user-auth`
- **THEN** le worker passe à `paused` et sa raison de blocage, exposée par l'endpoint de statut et par la liste globale des pools, décrit l'échec du provisionnement

### Requirement: Changes en pause exclues du dispatcher
Tant qu'un worker est en pause pour un changement, le dispatcher SHALL NOT réassigner ce changement à un worker, même s'il reste dans la colonne Todo et sans dépendance en attente. Le changement SHALL redevenir éligible uniquement lorsque sa pause est levée par une reprise explicite, par la rétrogradation du changement vers Ready (voir `kanban-ready-column`) ou par l'arrêt du pool. Les autres changements éligibles SHALL continuer à être distribués normalement. Un worker dont l'exécution est interrompue par l'arrêt du pool ou par l'annulation explicite de son changement SHALL NOT apparaître comme en pause, et un pool redémarré SHALL partir sans aucune pause héritée d'un run précédent. L'identifiant d'un worker terminé SHALL NOT pouvoir supprimer l'état d'un worker plus récent qui a réutilisé cet identifiant.

#### Scenario: Change en pause non relancée
- **WHEN** le worker du changement `add-user-auth` passe en pause alors que ce changement reste éligible dans la colonne Todo
- **THEN** le dispatcher ne lui assigne aucun nouveau worker aux ticks suivants et aucune nouvelle tentative de provisionnement n'a lieu

#### Scenario: Autres changements non bloqués
- **WHEN** le changement A est en pause et que le changement B est éligible avec un worker libre
- **THEN** le dispatcher distribue B normalement

#### Scenario: Arrêt du pool
- **WHEN** l'utilisateur arrête le pool alors que le changement A est en pause, puis le redémarre
- **THEN** le changement A redevient éligible à la distribution

#### Scenario: Arrêt du pool pendant l'exécution d'un worker
- **WHEN** l'utilisateur arrête le pool alors que le worker du changement A exécute un tour d'agent, puis redémarre le pool
- **THEN** le worker interrompu n'apparaît pas comme en pause, le changement A est éligible dès le redémarrage et son issue enregistrée est `stopped`

#### Scenario: Annulation explicite d'un changement
- **WHEN** l'utilisateur désarme de force le changement A alors que son worker est actif
- **THEN** le worker est annulé sans apparaître comme en pause

#### Scenario: Pause levée par la rétrogradation du changement
- **WHEN** l'utilisateur rétrograde le changement A vers Ready alors que son worker est en pause, puis le promeut de nouveau vers Todo
- **THEN** le worker en pause a disparu de la liste des workers du pool dès la rétrogradation, et le dispatcher redistribue A aux ticks suivants sans reprise explicite

#### Scenario: Libération d'une pause sans effet sur les autres workers
- **WHEN** la pause du changement A est levée par sa rétrogradation alors que le worker du changement B s'exécute
- **THEN** le worker de B n'est ni interrompu ni modifié, et son identifiant reste inchangé

### Requirement: Reprise explicite d'un worker en pause
Le backend SHALL exposer une action de reprise d'un worker en pause, par workspace et par identifiant de worker (`POST /api/workspaces/{id}/pool/workers/{workerId}/resume`). La reprise SHALL retirer le worker de la liste des workers en pause, ce qui rend son changement de nouveau éligible ; le dispatcher SHALL alors le reprendre selon ses règles habituelles en réutilisant la branche et le worktree existants. La reprise SHALL NOT interrompre les autres workers du pool. Le corps de la requête SHALL être optionnel : sans corps, ou avec `{"finalize_only": false}`, le worker repris relance l'agent comme avant ; avec `{"finalize_only": true}`, la reprise SHALL suivre l'exigence « Reprise en finalisant sans tour d'agent ». L'action SHALL retourner une erreur `404` si le workspace est inconnu ou si aucun worker en pause n'a cet identifiant, et `409` si aucun pool n'est actif pour ce workspace.

#### Scenario: Reprise d'un worker en pause
- **WHEN** le client demande la reprise du worker 1, en pause sur le changement `add-user-auth`, dans un pool actif
- **THEN** le worker 1 disparaît des workers en pause, le changement redevient éligible, un worker le reprend dans son worktree existant, et les autres workers du pool ne sont pas interrompus

#### Scenario: Worker non en pause
- **WHEN** le client demande la reprise d'un worker en cours d'exécution ou d'un identifiant inconnu
- **THEN** le backend retourne `404` sans modifier l'état du pool

#### Scenario: Pool arrêté
- **WHEN** le client demande la reprise d'un worker alors qu'aucun pool n'est actif pour ce workspace
- **THEN** le backend retourne `409`

#### Scenario: Reprise sans corps
- **WHEN** le client demande la reprise d'un worker en pause sans corps de requête
- **THEN** le worker repris relance un tour d'agent sur les tâches restantes, comme avant l'introduction de la reprise en finalisant

### Requirement: Présence du change dans le worktree
Avant de provisionner la branche et le worktree d'un changement dont la branche `feature/<change>` n'existe pas encore, le worker SHALL vérifier que le fichier `tasks.md` du changement est committé dans la branche courante du dépôt ; sinon il SHALL ne créer ni branche ni worktree, ne lancer aucun agent et passer à `paused` avec une raison de blocage lisible indiquant que le changement doit être committé dans le dépôt avant d'être lancé. Après le provisionnement, le worker SHALL vérifier que `tasks.md` est présent dans le worktree : s'il est absent alors que le changement est committé dans la branche courante, le comportement de recréation ou de pause de la branche périmée décrit dans « Reprise de la branche et du worktree existants » s'applique ; la raison de pause d'une branche périmée portant du travail SHALL se distinguer de celle d'un changement non committé et indiquer que la branche `feature/<change>` ne contient pas le changement et que l'utilisateur doit y intégrer la branche courante ou la supprimer. Si le changement n'est pas committé dans la branche courante, la raison de pause indiquant qu'il doit être committé SHALL être utilisée.

#### Scenario: Change committé
- **WHEN** un worker prend en charge le changement `add-user-auth` dont le dossier est committé dans le dépôt
- **THEN** le worktree contient `openspec/changes/add-user-auth/tasks.md` et le worker lance l'agent normalement

#### Scenario: Change non committé
- **WHEN** un worker prend en charge le changement `add-user-auth` dont les fichiers n'ont jamais été committés
- **THEN** aucun agent n'est lancé, le worker passe à `paused` et sa raison de blocage indique que le changement doit être committé avant d'être lancé

#### Scenario: Change non committé sans branche ni worktree orphelins
- **WHEN** un worker prend en charge le changement `add-user-auth` non committé alors qu'aucune branche `feature/add-user-auth` n'existe
- **THEN** le worker passe à `paused` et ni branche `feature/add-user-auth` ni worktree n'ont été créés

#### Scenario: Change committé après un premier échec
- **WHEN** le changement `add-user-auth` a été committé dans la branche courante après une pause pour « non committé », puis le worker est repris
- **THEN** le worker lance l'agent normalement sans que l'utilisateur ait à supprimer la branche ni le worktree

#### Scenario: Branche périmée portant du travail
- **WHEN** `feature/add-user-auth` porte du travail, ne contient pas le changement committé depuis dans la branche courante, et le worker est repris
- **THEN** aucun agent n'est lancé, la branche et le worktree sont intacts, et le worker passe à `paused` avec une raison de blocage distincte de « doit être committé », indiquant que la branche ne contient pas le changement

### Requirement: Commit du travail de l'agent avant finalisation
Après une validation réussie et une vérification de complétion réussie, le worker SHALL committer dans la branche `feature/<change>`, depuis le worktree, toutes les modifications produites par l'agent qui ne sont pas encore committées, dans les deux modes de délégation, avant toute fusion, transition vers la revue ou suppression du worktree. Si l'échec du commit empêche de conserver le travail (par exemple identité git absente), le worker SHALL passer à `paused` avec la cause dans sa raison de blocage et SHALL conserver le worktree tel quel. Si l'agent n'a produit aucun changement (aucune modification non committée et aucun commit propre à la branche par rapport à la branche courante du dépôt), le worker SHALL NOT finaliser le changement et SHALL passer à `paused` avec une raison de blocage indiquant qu'aucun travail n'a été produit.

#### Scenario: Modifications non committées de l'agent
- **WHEN** la validation et la vérification de complétion réussissent pour `add-user-auth` et que le worktree contient des modifications non committées
- **THEN** le worker committe ces modifications dans `feature/add-user-auth` avant de finaliser

#### Scenario: Agent ayant déjà committé
- **WHEN** l'agent a lui-même committé tout son travail et que le worktree est propre
- **THEN** le worker ne crée pas de commit supplémentaire vide et poursuit la finalisation

#### Scenario: Aucun travail produit
- **WHEN** la validation réussit mais que la branche ne contient ni modification non committée ni commit propre par rapport à la branche courante du dépôt
- **THEN** le worker ne finalise pas, passe à `paused` et sa raison de blocage indique qu'aucun travail n'a été produit

#### Scenario: Commit impossible
- **WHEN** le commit échoue parce que l'identité git n'est pas configurée
- **THEN** le worker passe à `paused`, sa raison de blocage reprend la cause de l'échec et le worktree est conservé avec ses modifications

### Requirement: Fusion sûre en mode full-autonomy
En mode `full-autonomy`, le worker SHALL fusionner la branche `feature/<change>` dans la branche actuellement extraite du dépôt, une fois le travail committé, et SHALL supprimer le worktree puis la branche uniquement après une fusion réussie. Le worker SHALL NOT supprimer de worktree contenant des modifications non committées. Les fusions d'un même workspace SHALL être sérialisées. Le worker SHALL NOT démarrer de fusion lorsqu'un merge est déjà en cours dans le dépôt et SHALL NOT annuler un merge qu'il n'a pas lui-même démarré. Si la fusion échoue, le worker SHALL annuler son propre merge, conserver la branche et le worktree, passer à `paused` avec une raison de blocage lisible reprenant la cause, et le changement SHALL NOT être redistribué tant que la pause n'est pas levée.

Le worker SHALL NOT démarrer de fusion, ni de commit du travail de l'agent, lorsqu'il a été annulé (arrêt du pool, rétrogradation forcée du change) : l'annulation SHALL être vérifiée avant le commit du travail et immédiatement après l'obtention du verrou de fusion du workspace, y compris lorsque ce verrou a été attendu. Un worker annulé avant le début de sa fusion SHALL libérer le verrou, conserver la branche, le worktree et le travail déjà committé, et terminer avec l'issue `stopped`. Une fusion déjà démarrée SHALL NOT être interrompue : elle aboutit ou échoue de façon atomique, puis le worker poursuit sa finalisation normale.

#### Scenario: Fusion réussie
- **WHEN** le travail de `add-user-auth` est committé et que la fusion dans la branche courante réussit
- **THEN** le worktree puis la branche `feature/add-user-auth` sont supprimés et le run se termine avec l'issue `completed`

#### Scenario: Fusion en conflit
- **WHEN** la fusion de `feature/add-user-auth` provoque un conflit
- **THEN** le merge est annulé, la branche et le worktree sont conservés avec le travail committé, le worker passe à `paused` avec une raison indiquant l'échec de la fusion, et le dispatcher ne relance pas ce changement

#### Scenario: Merge déjà en cours dans le dépôt
- **WHEN** l'utilisateur a un merge en cours dans le dépôt au moment où le worker veut fusionner
- **THEN** le worker ne lance pas de fusion, n'annule pas le merge de l'utilisateur, conserve la branche et le worktree, et passe à `paused` avec une raison indiquant qu'un merge est en cours

#### Scenario: Deux workers qui finalisent en même temps
- **WHEN** deux workers du même workspace atteignent la fusion simultanément
- **THEN** les fusions s'exécutent l'une après l'autre et aucune n'échoue à cause de l'autre

#### Scenario: Annulation pendant l'attente du verrou de fusion
- **WHEN** un worker a terminé sa validation et attend le verrou de fusion pendant le merge d'un autre worker, puis est annulé (arrêt du pool ou rétrogradation forcée) avant d'obtenir le verrou
- **THEN** une fois le verrou obtenu le worker ne lance aucune fusion, la branche `feature/<change>` n'est pas fusionnée, la branche et le worktree sont conservés, le verrou est libéré pour les autres workers et le run se termine avec l'issue `stopped`

#### Scenario: Annulation avant le commit du travail
- **WHEN** un worker est annulé après une validation réussie et avant le commit du travail de l'agent
- **THEN** le worker ne committe rien, ne fusionne rien, conserve le worktree tel quel et le run se termine avec l'issue `stopped`

#### Scenario: Arrêt puis redémarrage rapide du pool
- **WHEN** le pool est arrêté puis redémarré alors qu'un ancien worker de `add-user-auth` attend encore le verrou de fusion, et qu'un nouveau worker reprend `add-user-auth`
- **THEN** l'ancien worker ne fusionne pas la branche et ne supprime ni la branche ni le worktree utilisés par le nouveau worker

#### Scenario: Fusion déjà démarrée au moment de l'annulation
- **WHEN** l'annulation survient alors que la fusion du worker a déjà démarré
- **THEN** la fusion n'est pas interrompue, elle aboutit ou échoue selon le cas, et si elle aboutit le run se termine avec l'issue `completed`

### Requirement: Changes en attente de revue exclues du dispatcher
En mode `hitl-review`, lorsqu'un worker transmet un changement en revue, le travail étant committé dans la branche `feature/<change>`, le changement SHALL porter un marqueur de revue persistant (voir `change-review-state`) et avoir le statut `to-review`. Le dispatcher SHALL NOT réassigner ce changement à un worker tant que son marqueur existe, y compris après l'arrêt puis le redémarrage du pool ou du backend. Les autres changements éligibles SHALL continuer à être distribués normalement. Un changement en revue SHALL être de nouveau éligible uniquement lorsque son marqueur est levé par une action explicite de l'utilisateur (voir `change-review-actions`). Un changement qui en dépend SHALL rester non éligible tant qu'il est en revue.

#### Scenario: Change transmis en revue non relancé
- **WHEN** le worker du changement `add-user-auth` termine en mode `hitl-review` avec l'issue `awaiting-review`
- **THEN** le dispatcher ne lui assigne aucun nouveau worker aux ticks suivants et le travail reste committé dans `feature/add-user-auth`

#### Scenario: Autres changements non bloqués par une revue
- **WHEN** le changement A est en attente de revue et que le changement B est éligible avec un worker libre
- **THEN** le dispatcher distribue B normalement

#### Scenario: Redémarrage du pool avec un change en revue
- **WHEN** le pool est arrêté puis redémarré alors que `add-user-auth` est en revue
- **THEN** le dispatcher ne lui assigne aucun worker et `add-user-auth` reste en To Review

#### Scenario: Dépendant d'un change en revue
- **WHEN** le changement B dépend du changement A qui est en revue
- **THEN** le dispatcher ne distribue pas B tant que A n'a pas été approuvé et fusionné

### Requirement: Résultat d'agent en erreur
Lorsqu'un tour d'agent se termine par un résultat qui signale une erreur (échec d'exécution, nombre maximal de tours atteint, quota ou authentification), le worker SHALL NOT le compter comme un tour réussi : il SHALL passer à `paused` avec une raison de blocage reprenant le motif d'erreur signalé par l'agent, sans lancer la validation ni finaliser.

#### Scenario: Agent qui termine en erreur
- **WHEN** l'agent répond à l'invocation d'application par un résultat marqué en erreur
- **THEN** le worker passe à `paused`, sa raison de blocage reprend le motif d'erreur, et aucune validation n'est exécutée

#### Scenario: Agent qui termine normalement
- **WHEN** l'agent répond par un résultat non marqué en erreur
- **THEN** le worker poursuit avec la validation comme avant

### Requirement: Délais maximums des tours d'agent et de la validation
Le worker SHALL borner la durée d'un tour d'agent par un délai d'inactivité (aucune sortie de l'agent pendant la durée définie) et la durée d'exécution de la commande de validation par un délai maximal. Lorsqu'un délai est dépassé, le worker SHALL terminer le processus concerné et tous ses processus enfants, passer à `paused` avec une raison de blocage indiquant quel délai a été dépassé, et libérer son emplacement dans le pool. Un tour d'agent qui produit régulièrement des sorties SHALL NOT être interrompu par le délai d'inactivité, quelle que soit sa durée totale.

#### Scenario: Agent bloqué sans sortie
- **WHEN** l'agent n'émet aucune sortie pendant plus que le délai d'inactivité
- **THEN** le processus de l'agent et ses enfants sont terminés, le worker passe à `paused` avec une raison indiquant l'inactivité de l'agent, et l'emplacement est libéré

#### Scenario: Agent long mais actif
- **WHEN** l'agent travaille plus longtemps que le délai d'inactivité tout en émettant régulièrement des sorties
- **THEN** le tour n'est pas interrompu

#### Scenario: Validation qui ne termine pas
- **WHEN** la commande de validation dépasse son délai maximal
- **THEN** la commande et ses processus enfants sont terminés et le worker passe à `paused` avec une raison indiquant le dépassement du délai de validation

### Requirement: Terminaison des groupes de processus
Les processus d'agent et de validation lancés par un worker SHALL s'exécuter dans leur propre groupe de processus, et l'annulation d'un worker (arrêt du pool, désarmement forcé du changement, dépassement de délai) ainsi que l'arrêt du serveur SHALL terminer le groupe entier, processus enfants compris. Lors de l'arrêt du serveur, le backend SHALL arrêter tous les pools actifs et attendre la terminaison de leurs workers, dans la limite du délai d'arrêt du serveur, avant de quitter.

#### Scenario: Arrêt du pool avec un agent qui a lancé des commandes
- **WHEN** l'utilisateur arrête le pool alors qu'un agent a démarré un processus enfant (test, serveur de développement)
- **THEN** l'agent et ce processus enfant sont terminés

#### Scenario: Arrêt du serveur avec des pools actifs
- **WHEN** le serveur reçoit un signal d'arrêt alors que des pools ont des workers actifs
- **THEN** tous les pools sont arrêtés, les groupes de processus des agents sont terminés, puis le serveur se ferme

### Requirement: Branche de base du changement
Lors de la première prise en charge d'un changement, c'est-à-dire lorsque la branche `feature/<change>` est créée, le backend SHALL enregistrer le nom de la branche du dépôt dont elle est issue (sa branche de base). Cet enregistrement SHALL survivre aux pauses et aux redémarrages du serveur, SHALL NOT être réécrit lors d'une reprise de la branche existante, et SHALL disparaître avec la branche. Aucune base n'est enregistrée lorsque le HEAD du dépôt est détaché à la création de la branche.

En mode `full-autonomy`, avant de fusionner, le worker SHALL comparer la branche actuellement extraite du dépôt à la branche de base enregistrée. Si elles diffèrent, ou si le HEAD du dépôt est détaché alors qu'une base est enregistrée, le worker SHALL NOT fusionner, SHALL conserver la branche et le worktree avec le travail committé, et SHALL passer à `paused` avec une raison de blocage nommant la branche de base et la branche courante. Une branche `feature/<change>` sans base enregistrée (créée avant l'introduction de cet enregistrement, ou depuis un HEAD détaché) n'est soumise à aucune contrainte de base.

#### Scenario: Base enregistrée à la création de la branche
- **WHEN** un worker prend en charge `add-user-auth` pour la première fois alors que le dépôt est sur `main`
- **THEN** `main` est enregistrée comme branche de base de `feature/add-user-auth`

#### Scenario: Reprise sans réécriture de la base
- **WHEN** un worker reprend `add-user-auth` après une pause alors que le dépôt est sur une autre branche que `main`
- **THEN** la base enregistrée reste `main`

#### Scenario: Fusion dans la branche de base
- **WHEN** le worker de `add-user-auth` (base `main`) atteint la fusion alors que le dépôt est toujours sur `main`
- **THEN** la fusion suit son cours normal

#### Scenario: Branche courante différente de la base
- **WHEN** le worker de `add-user-auth` (base `main`) atteint la fusion alors que l'utilisateur a extrait `feature/other` dans le dépôt
- **THEN** aucune fusion n'a lieu, la branche et le worktree sont conservés, et le worker passe à `paused` avec une raison nommant `main` et `feature/other`

#### Scenario: Reprise après retour sur la branche de base
- **WHEN** l'utilisateur ré-extrait `main` dans le dépôt puis reprend le worker en pause
- **THEN** le worker poursuit et peut fusionner dans `main`

#### Scenario: HEAD détaché
- **WHEN** le worker de `add-user-auth` (base `main`) atteint la fusion alors que le HEAD du dépôt est détaché
- **THEN** aucune fusion n'a lieu et le worker passe à `paused` avec une raison indiquant que le dépôt n'est sur aucune branche

#### Scenario: Branche sans base enregistrée
- **WHEN** `feature/legacy-change` existait avant l'introduction de l'enregistrement et que son worker atteint la fusion
- **THEN** le contrôle de branche de base n'est pas appliqué et la fusion dans la branche courante suit son cours normal

### Requirement: Intégration et revalidation avant fusion
En mode `full-autonomy`, avant d'acquérir le verrou de fusion du workspace, le worker SHALL vérifier que la branche cible (la branche actuellement extraite du dépôt) ne contient aucun commit absent de `feature/<change>`. Si elle en contient, le worker SHALL les intégrer dans le worktree du changement, puis SHALL rejouer la validation sur le résultat intégré, avec la même boucle d'auto-guérison et le même budget de tentatives `max_attempts` que la validation initiale, et SHALL committer dans `feature/<change>` les éventuelles corrections apportées par l'agent. Si l'intégration provoque un conflit, le worker SHALL annuler l'intégration, conserver la branche et le worktree tels qu'avant l'intégration, et passer à `paused` avec une raison de blocage indiquant le conflit. Si la validation du résultat intégré échoue après épuisement des tentatives, le worker SHALL passer à `paused` sans fusionner.

Le worker SHALL NOT fusionner dans la branche cible tant que celle-ci contient un commit absent de `feature/<change>` au moment où le verrou de fusion est détenu : dans ce cas il SHALL libérer le verrou sans fusionner et recommencer l'intégration et la revalidation, sans jamais revalider tant qu'il détient le verrou. Après 3 intégrations successives sans pouvoir fusionner parce que la branche cible continue d'avancer, le worker SHALL conserver la branche et le worktree et passer à `paused` avec une raison l'indiquant. Lorsque la branche cible ne contient aucun commit absent de `feature/<change>`, le worker SHALL NOT rejouer la validation. Ces règles ne s'appliquent pas au worker en mode `hitl-review`, qui ne fusionne jamais : l'intégration de la branche cible, la revalidation et la fusion y sont effectuées à l'approbation explicite de l'utilisateur (voir `change-review-actions`).

#### Scenario: Branche cible inchangée
- **WHEN** le worker de `add-user-auth` finit sa validation et que `main` n'a reçu aucun commit depuis la création de la branche
- **THEN** aucune intégration ni revalidation supplémentaire n'a lieu et la fusion suit son cours normal

#### Scenario: Branche cible avancée sans conflit
- **WHEN** l'utilisateur a committé sur `main` pendant le run et que ces commits s'intègrent sans conflit dans `feature/add-user-auth`
- **THEN** le worker intègre `main` dans le worktree, rejoue la validation sur le résultat, et ne fusionne qu'une fois cette validation réussie

#### Scenario: Validation en échec après intégration
- **WHEN** le résultat intégré fait échouer un test qui passait avant l'intégration
- **THEN** le worker lance un tour de guérison dans la même session d'agent, rejoue la validation, committe la correction, et fusionne si elle réussit ; si les tentatives sont épuisées, il passe à `paused` sans fusionner

#### Scenario: Conflit d'intégration
- **WHEN** les commits de `main` entrent en conflit avec le travail de `feature/add-user-auth`
- **THEN** l'intégration est annulée, la branche et le worktree retrouvent leur état d'avant l'intégration, aucune fusion n'a lieu et le worker passe à `paused` avec une raison indiquant le conflit

#### Scenario: Deuxième worker après la fusion du premier
- **WHEN** deux workers du même workspace finissent leur validation (y compris simultanément), que le premier fusionne dans `main`, puis que le second prépare sa fusion
- **THEN** le second intègre le résultat du premier dans son worktree, rejoue sa validation sur ce résultat, puis fusionne, sans pause ni échec dû au premier

#### Scenario: Branche cible avancée pendant la finalisation
- **WHEN** `main` reçoit un commit entre la fin de l'intégration et l'obtention du verrou de fusion
- **THEN** le worker libère le verrou sans fusionner, intègre ce commit, rejoue la validation, puis fusionne, sans pause

#### Scenario: Branche cible qui n'arrête pas d'avancer
- **WHEN** `main` reçoit un nouveau commit après chacune des 3 intégrations successives du worker
- **THEN** le worker ne fusionne pas, libère le verrou, conserve la branche et le worktree, et passe à `paused` avec une raison indiquant que la branche cible continue d'avancer

#### Scenario: Mode hitl-review
- **WHEN** un worker en `hitl-review` termine avec succès alors que `main` a avancé
- **THEN** aucune intégration ni fusion n'est effectuée par le worker, et le changement attend la revue ; l'intégration éventuelle a lieu à l'approbation de l'utilisateur

### Requirement: Reprise en finalisant sans tour d'agent
Lorsque la reprise d'un worker en pause est demandée avec `{"finalize_only": true}`, le backend SHALL vérifier d'abord que le `tasks.md` du worktree du worker existe, contient au moins une tâche et n'en contient plus aucune non cochée ; sinon il SHALL retourner `409` avec un message indiquant le nombre de tâches restantes (ou l'absence de liste de tâches), laisser le worker en pause et ne rien modifier. Si la vérification passe, la reprise SHALL lever la pause comme une reprise ordinaire, et le worker qui reprend le changement SHALL NOT démarrer de subprocess d'agent ni envoyer de tour `/opsx:apply`. Il SHALL rejouer la validation du worktree, puis enchaîner la vérification de complétude, le commit du travail et la finalisation propres au mode de délégation (fusion en `full-autonomy`, passage en To Review en `hitl-review`). Comme aucun agent n'est disponible, un échec de validation SHALL mettre le worker en `paused` avec la raison de l'échec, sans tour de guérison ni tentative consommée. L'intention de finaliser SHALL ne valoir que pour cette reprise : une pause ultérieure suivie d'une reprise sans `finalize_only` relance l'agent, et l'intention SHALL être oubliée si le pool est arrêté ou si la pause du change est levée avant que le dispatcher ne l'ait consommée.

#### Scenario: Finalisation après tâche manuelle en hitl-review
- **WHEN** le worker du change `add-user-auth` est en pause avec 9 tâches cochées sur 10, que l'utilisateur coche la dernière dans le worktree, puis demande la reprise avec `finalize_only` en mode `hitl-review`
- **THEN** aucun subprocess d'agent n'est démarré, la validation est rejouée, le travail est committé dans `feature/add-user-auth` et le change passe en To Review

#### Scenario: Finalisation en full-autonomy
- **WHEN** la même reprise en finalisant est demandée en mode `full-autonomy` et que la validation réussit
- **THEN** aucun agent n'est lancé, le travail est committé, la branche est fusionnée selon les règles de fusion sûre, puis le worktree et la branche sont supprimés

#### Scenario: Tâches restantes
- **WHEN** la reprise avec `finalize_only` est demandée alors que le `tasks.md` du worktree contient encore 2 tâches non cochées
- **THEN** le backend retourne `409` avec un message indiquant qu'il reste 2 tâches, le worker reste en pause et sa raison de blocage est inchangée

#### Scenario: Validation en échec sans agent
- **WHEN** la reprise en finalisant est acceptée mais que la validation échoue
- **THEN** le worker passe à `paused` avec la raison de l'échec, sans tour de guérison ni tentative consommée, et le travail du worktree est conservé

#### Scenario: Intention non persistée
- **WHEN** l'utilisateur arrête le pool après avoir demandé une reprise en finalisant que le dispatcher n'a pas encore consommée, puis redémarre le pool et reprend le change sans `finalize_only`
- **THEN** le worker repris relance l'agent normalement
