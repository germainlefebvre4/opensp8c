# Spec Delta

## ADDED Requirements

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
Lorsqu'un worker prend en charge un changement dont la branche `feature/<change>` existe déjà, le backend SHALL réutiliser cette branche et, si le worktree `.opensp8c/worktrees/wt-<change>` existe déjà, ce worktree tel quel, sans recréer la branche ni écraser les modifications non commitées qu'il contient. L'existence de la branche SHALL être déterminée par le résultat de git et non par le contenu de sa sortie. Ce comportement s'applique à toute reprise d'un changement : reprise après pause, relance après « Demander des corrections », redémarrage du pool.

#### Scenario: Reprise après pause
- **WHEN** un worker précédemment mis en pause pour le changement `add-user-auth` est repris alors que `feature/add-user-auth` et son worktree existent
- **THEN** le worker réutilise ce worktree avec ses modifications non commitées et n'échoue pas sur l'existence de la branche

#### Scenario: Branche existante sans worktree
- **WHEN** `feature/add-user-auth` existe mais que son worktree a été supprimé
- **THEN** le backend crée un nouveau worktree sur cette branche existante, sans tenter de recréer la branche

#### Scenario: Première prise en charge
- **WHEN** aucune branche `feature/add-user-auth` n'existe
- **THEN** le backend crée la branche et le worktree comme pour un nouveau changement

### Requirement: Échec de provisionnement visible
Lorsque le provisionnement de la branche ou du worktree d'un changement échoue, le worker SHALL passer au statut `paused` avec une raison de blocage lisible reprenant la cause de l'échec, au lieu d'échouer silencieusement.

#### Scenario: Provisionnement en échec
- **WHEN** `git worktree add` échoue pour le changement `add-user-auth`
- **THEN** le worker passe à `paused` et sa raison de blocage, exposée par l'endpoint de statut et par la liste globale des pools, décrit l'échec du provisionnement

### Requirement: Changes en pause exclues du dispatcher
Tant qu'un worker est en pause pour un changement, le dispatcher SHALL NOT réassigner ce changement à un worker, même s'il reste dans la colonne Todo et sans dépendance en attente. Le changement SHALL redevenir éligible uniquement lorsque sa pause est levée par une reprise explicite ou par l'arrêt du pool. Les autres changements éligibles SHALL continuer à être distribués normalement.

#### Scenario: Change en pause non relancée
- **WHEN** le worker du changement `add-user-auth` passe en pause alors que ce changement reste éligible dans la colonne Todo
- **THEN** le dispatcher ne lui assigne aucun nouveau worker aux ticks suivants et aucune nouvelle tentative de provisionnement n'a lieu

#### Scenario: Autres changements non bloqués
- **WHEN** le changement A est en pause et que le changement B est éligible avec un worker libre
- **THEN** le dispatcher distribue B normalement

#### Scenario: Arrêt du pool
- **WHEN** l'utilisateur arrête le pool alors que le changement A est en pause, puis le redémarre
- **THEN** le changement A redevient éligible à la distribution

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
