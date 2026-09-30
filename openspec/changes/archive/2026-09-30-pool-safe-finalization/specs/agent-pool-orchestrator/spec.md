# Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

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
