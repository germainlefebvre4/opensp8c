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
Après chaque invocation de l'agent, le worker SHALL exécuter la commande de validation du workspace dans le worktree de son changement. Cette commande SHALL être résolue ainsi : le réglage `validationCommand` du workspace s'il est défini, sinon le défaut global de Configuration, sinon le résultat de l'auto-détection. L'auto-détection SHALL examiner la racine du worktree puis chacun de ses sous-répertoires directs (hors répertoires cachés et `node_modules`) et SHALL retenir, pour chaque répertoire concerné, `go test ./...` lorsqu'il contient un `go.mod` et `npm test` lorsque son `package.json` déclare un script `test` et que ses dépendances sont installées (répertoire `node_modules` présent). Chaque commande détectée SHALL être exécutée dans le répertoire où elle a été détectée, et la validation SHALL réussir uniquement si toutes les commandes réussissent. Une commande explicitement configurée SHALL être exécutée à la racine du worktree et SHALL l'emporter sur toute détection.

#### Scenario: Module Go dans un sous-répertoire
- **WHEN** le worktree du changement `add-user-auth` contient `backend/go.mod` et `frontend/package.json` avec un script `test` et un répertoire `frontend/node_modules`, et qu'aucune commande de validation n'est configurée
- **THEN** le worker exécute `go test ./...` dans `backend/` et `npm test` dans `frontend/`, et la validation réussit uniquement si les deux réussissent

#### Scenario: Projet Node sans dépendances installées
- **WHEN** `frontend/package.json` déclare un script `test` mais que `frontend/node_modules` n'existe pas dans le worktree, et que `backend/go.mod` existe
- **THEN** l'auto-détection ignore `frontend/` et le worker exécute uniquement `go test ./...` dans `backend/`

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
Lorsque la validation ne peut pas être exécutée pour une raison d'environnement, c'est-à-dire lorsque l'auto-détection ne trouve aucune commande, qu'une commande configurée est vide ou introuvable, ou que son répertoire d'exécution n'existe pas, le worker SHALL passer immédiatement au statut `paused` avec une raison de blocage lisible, sans lancer de tour d'auto-guérison ni consommer de tentative. Un échec de test ou de compilation d'une commande qui a démarré SHALL continuer à suivre la boucle d'auto-guérison.

#### Scenario: Aucune commande de validation détectable
- **WHEN** aucune commande n'est configurée et que l'auto-détection ne trouve ni `go.mod` ni `package.json` avec script `test` dans le worktree
- **THEN** le worker passe à `paused` sans tour de guérison, et sa raison de blocage indique qu'aucune commande de validation n'a été détectée et invite à en configurer une

#### Scenario: Commande configurée introuvable
- **WHEN** la commande de validation configurée référence un exécutable absent de la machine
- **THEN** le worker passe à `paused` sans tour de guérison, et sa raison de blocage nomme la commande introuvable

#### Scenario: Échec de tests toujours guérissable
- **WHEN** une commande de validation démarre puis échoue parce qu'un test est rouge
- **THEN** le worker lance un tour de guérison et n'est pas mis en pause avant l'épuisement de `max_attempts`

### Requirement: Reprise de la branche et du worktree existants
Lorsqu'un worker prend en charge un changement dont la branche `feature/<change>` existe déjà, le backend SHALL réutiliser cette branche et, si le worktree du changement existe déjà, ce worktree tel quel, sans recréer la branche ni écraser les modifications non commitées qu'il contient. Un worktree créé à l'ancien emplacement `.opensp8c/worktrees/wt-<change>` (sans segment de workspace) qui est un worktree enregistré du dépôt du workspace sur cette branche SHALL être réutilisé tel quel. L'existence de la branche SHALL être déterminée par le résultat de git et non par le contenu de sa sortie. Ce comportement s'applique à toute reprise d'un changement : reprise après pause, relance après « Demander des corrections », redémarrage du pool.

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

### Requirement: Échec de provisionnement visible
Lorsque le provisionnement de la branche ou du worktree d'un changement échoue, le worker SHALL passer au statut `paused` avec une raison de blocage lisible reprenant la cause de l'échec, au lieu d'échouer silencieusement.

#### Scenario: Provisionnement en échec
- **WHEN** `git worktree add` échoue pour le changement `add-user-auth`
- **THEN** le worker passe à `paused` et sa raison de blocage, exposée par l'endpoint de statut et par la liste globale des pools, décrit l'échec du provisionnement

### Requirement: Changes en pause exclues du dispatcher
Tant qu'un worker est en pause pour un changement, le dispatcher SHALL NOT réassigner ce changement à un worker, même s'il reste dans la colonne Todo et sans dépendance en attente. Le changement SHALL redevenir éligible uniquement lorsque sa pause est levée par une reprise explicite ou par l'arrêt du pool. Les autres changements éligibles SHALL continuer à être distribués normalement. Un worker dont l'exécution est interrompue par l'arrêt du pool ou par l'annulation explicite de son changement SHALL NOT apparaître comme en pause, et un pool redémarré SHALL partir sans aucune pause héritée d'un run précédent. L'identifiant d'un worker terminé SHALL NOT pouvoir supprimer l'état d'un worker plus récent qui a réutilisé cet identifiant.

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

### Requirement: Reprise explicite d'un worker en pause
Le backend SHALL exposer une action de reprise d'un worker en pause, par workspace et par identifiant de worker (`POST /api/workspaces/{id}/pool/workers/{workerId}/resume`). La reprise SHALL retirer le worker de la liste des workers en pause, ce qui rend son changement de nouveau éligible ; le dispatcher SHALL alors le reprendre selon ses règles habituelles en réutilisant la branche et le worktree existants. La reprise SHALL NOT interrompre les autres workers du pool. L'action SHALL retourner une erreur `404` si le workspace est inconnu ou si aucun worker en pause n'a cet identifiant, et `409` si aucun pool n'est actif pour ce workspace.

#### Scenario: Reprise d'un worker en pause
- **WHEN** le client demande la reprise du worker 1, en pause sur le changement `add-user-auth`, dans un pool actif
- **THEN** le worker 1 disparaît des workers en pause, le changement redevient éligible, un worker le reprend dans son worktree existant, et les autres workers du pool ne sont pas interrompus

#### Scenario: Worker non en pause
- **WHEN** le client demande la reprise d'un worker en cours d'exécution ou d'un identifiant inconnu
- **THEN** le backend retourne `404` sans modifier l'état du pool

#### Scenario: Pool arrêté
- **WHEN** le client demande la reprise d'un worker alors qu'aucun pool n'est actif pour ce workspace
- **THEN** le backend retourne `409`
### Requirement: Présence du change dans le worktree
Avant de lancer l'agent, le worker SHALL vérifier que le changement à implémenter existe dans son worktree, c'est-à-dire que le fichier `tasks.md` du changement y est présent. Un worktree étant créé à partir du dernier commit de la branche courante du dépôt, un changement non committé est absent du worktree : le worker SHALL alors ne lancer aucun agent et passer à `paused` avec une raison de blocage lisible indiquant que le changement doit être committé dans le dépôt avant d'être lancé.

#### Scenario: Change committé
- **WHEN** un worker prend en charge le changement `add-user-auth` dont le dossier est committé dans le dépôt
- **THEN** le worktree contient `openspec/changes/add-user-auth/tasks.md` et le worker lance l'agent normalement

#### Scenario: Change non committé
- **WHEN** un worker prend en charge le changement `add-user-auth` dont les fichiers n'ont jamais été committés
- **THEN** aucun agent n'est lancé, le worker passe à `paused` et sa raison de blocage indique que le changement doit être committé avant d'être lancé

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

### Requirement: Changes en attente de revue exclues du dispatcher
En mode `hitl-review`, lorsqu'un worker transmet un changement en revue, le travail étant committé dans la branche `feature/<change>`, le dispatcher SHALL NOT réassigner ce changement à un worker tant que le pool reste actif, même s'il reste dans la colonne Todo. Les autres changements éligibles SHALL continuer à être distribués normalement. L'arrêt puis le redémarrage du pool SHALL rendre le changement de nouveau éligible.

#### Scenario: Change transmis en revue non relancé
- **WHEN** le worker du changement `add-user-auth` termine en mode `hitl-review` avec l'issue `awaiting-review`
- **THEN** le dispatcher ne lui assigne aucun nouveau worker aux ticks suivants et le travail reste committé dans `feature/add-user-auth`

#### Scenario: Autres changements non bloqués par une revue
- **WHEN** le changement A est en attente de revue et que le changement B est éligible avec un worker libre
- **THEN** le dispatcher distribue B normalement

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
