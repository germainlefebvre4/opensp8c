# Spec Delta

## ADDED Requirements

### Requirement: Bandeau de vérification dans le DetailPanel
Lorsque le change ouvert a le statut `verifying`, le DetailPanel SHALL afficher, au-dessus de la liste des tâches, un bandeau de vérification indiquant l'état (`queued`, `running` avec l'étape, `failed`, `passed`) et, pour les états `failed` et `passed`, le rapport de la dernière vérification rendu en Markdown, chargé par `GET …/verification/report`. Pour l'état `failed`, le bandeau SHALL proposer trois boutons : « Relancer la vérification », « Finaliser sans vérification » et « Demander des corrections ». « Finaliser sans vérification » SHALL être désactivé, avec le nombre de tâches restantes, tant que la liste des tâches du change n'est pas entièrement cochée ; la coche de la dernière tâche SHALL l'activer sans rechargement. « Demander des corrections » SHALL ouvrir la saisie de retour utilisée pour la revue. Le message d'erreur du backend SHALL être affiché sans modifier l'état affiché. Le bandeau SHALL être absent pour un change qui n'est pas `verifying`.

#### Scenario: Change en cours de vérification
- **WHEN** l'utilisateur ouvre un change dont la vérification tourne
- **THEN** le bandeau indique « en cours » avec l'étape, sans bouton d'action

#### Scenario: Change en échec avec rapport
- **WHEN** l'utilisateur ouvre un change `failed`
- **THEN** le bandeau affiche le rapport en Markdown et les trois boutons d'action

#### Scenario: Finalisation bloquée
- **WHEN** 2 tâches du change `failed` sont décochées
- **THEN** « Finaliser sans vérification » est désactivé et indique « 2 tâches restantes »

#### Scenario: Finalisation débloquée
- **WHEN** l'utilisateur coche la dernière tâche d'un change `failed`
- **THEN** « Finaliser sans vérification » devient actif sans rechargement

#### Scenario: Demande de correction
- **WHEN** l'utilisateur clique sur « Demander des corrections », saisit un retour et valide
- **THEN** une requête `POST …/verification/request-correction` est envoyée avec le retour, et la carte quitte la colonne Verifying

#### Scenario: Erreur du backend
- **WHEN** une action est refusée en `409`
- **THEN** le message du backend est affiché dans le bandeau et l'état affiché ne change pas

#### Scenario: Change hors vérification
- **WHEN** l'utilisateur ouvre un change qui n'est pas `verifying`
- **THEN** aucun bandeau de vérification n'est affiché

### Requirement: Lecture de l'état de vérification d'un change
Chaque change retourné par `GET /api/workspaces/{id}/changes` et par `GET /api/workspaces/{id}/changes/{name}` SHALL porter, lorsque son statut est `verifying`, `verification_state` (`queued`, `running`, `failed` ou `passed`) et, lorsque l'état est `running`, `verification_step` (`conformity`). Ces champs SHALL être absents pour un change qui n'est pas `verifying`. L'état `queued` SHALL désigner un marqueur `pending` sans exécution en cours, et `running` un marqueur `pending` dont l'exécution est en cours dans le pool.

#### Scenario: Marqueur pending sans exécution
- **WHEN** un change porte le marqueur `pending` et que le pool est arrêté
- **THEN** `verification_state` vaut `queued` et `verification_step` est absent

#### Scenario: Vérification en cours
- **WHEN** la vérification de conformité d'un change tourne
- **THEN** `verification_state` vaut `running` et `verification_step` vaut `conformity`

#### Scenario: Change sans vérification
- **WHEN** un change n'est pas `verifying`
- **THEN** ni `verification_state` ni `verification_step` ne figurent dans la réponse
