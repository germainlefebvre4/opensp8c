# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

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

Le worker SHALL NOT fusionner dans la branche cible tant que celle-ci contient un commit absent de `feature/<change>` au moment où le verrou de fusion est détenu : dans ce cas il SHALL libérer le verrou sans fusionner et recommencer l'intégration et la revalidation, sans jamais revalider tant qu'il détient le verrou. Après 3 intégrations successives sans pouvoir fusionner parce que la branche cible continue d'avancer, le worker SHALL conserver la branche et le worktree et passer à `paused` avec une raison l'indiquant. Lorsque la branche cible ne contient aucun commit absent de `feature/<change>`, le worker SHALL NOT rejouer la validation. Ces règles ne s'appliquent pas en mode `hitl-review`, où le backend ne fusionne pas.

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
- **THEN** aucune intégration ni fusion n'est effectuée par le backend, et le changement attend la revue comme avant
