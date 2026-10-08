# Spec Delta

## Purpose

Définit l'étape `ui` de la file de vérification d'un change : une vérification dans l'application lancée depuis le worktree, exécutée par un agent sous un verrou exclusif, qui produit un verdict, des preuves et la coche des tâches de validation humaine qu'elle a vérifiées.

## ADDED Requirements

### Requirement: Étape `ui` de la file de vérification
L'étape `ui` SHALL s'ajouter aux étapes de la vérification d'un change (voir `verification-stage`), après l'étape de conformité. Elle SHALL s'exécuter lorsque `ui` est résolue à activée (voir `verification-settings`) et que toutes les étapes précédentes activées ont réussi ; elle ne SHALL PAS s'exécuter après l'échec d'une étape précédente. Lorsque la commande de lancement (`uiStartCommand`) ou l'URL de base (`uiBaseUrl`) résolue est absente, l'étape SHALL échouer aussitôt, sans lancer ni l'application ni l'agent, avec une raison désignant le paramètre manquant. Le rôle de l'agent SHALL être `verifier` et l'étape SHALL se nommer `ui` dans `verification_step`, dans le journal et dans le rapport.

#### Scenario: Conformité puis UI
- **WHEN** `conformity` et `ui` sont activées et que la conformité d'un change réussit
- **THEN** l'étape `ui` démarre pour ce change

#### Scenario: Échec de la conformité
- **WHEN** la conformité d'un change échoue alors que `ui` est activée
- **THEN** l'étape `ui` ne démarre pas et le marqueur vaut `failed`

#### Scenario: UI seule
- **WHEN** seule `ui` est activée pour le change
- **THEN** l'étape `ui` s'exécute sans étape de conformité préalable

#### Scenario: Commande de lancement manquante
- **WHEN** `ui` est activée mais qu'aucune `uiStartCommand` n'est résolue
- **THEN** l'étape échoue avec la raison « uiStartCommand non configurée » et aucun processus n'est lancé

#### Scenario: URL de base manquante
- **WHEN** `ui` est activée mais qu'aucune `uiBaseUrl` n'est résolue
- **THEN** l'étape échoue avec la raison « uiBaseUrl non configurée » et aucun processus n'est lancé

### Requirement: Verrou exclusif de la vérification UI
Une seule étape `ui` SHALL s'exécuter à la fois pour l'ensemble du backend, tous workspaces confondus. Une étape `ui` prête à démarrer alors que le verrou est pris SHALL attendre dans l'ordre d'arrivée, avec l'état `waiting` exposé dans `verification_state` (et `verification_step` à `ui`). Une vérification dans l'état `waiting` SHALL NE PAS compter dans la limite de vérifications simultanées, de sorte qu'elle ne retarde pas les vérifications de conformité d'autres changes. Le verrou SHALL être libéré à la fin de l'étape, quelle qu'en soit l'issue, y compris par annulation. L'arrêt du pool ou l'annulation pendant l'attente SHALL retirer la vérification de la file d'attente et laisser son marqueur `pending`.

#### Scenario: Deux vérifications UI simultanées
- **WHEN** deux changes sont prêts pour l'étape `ui`
- **THEN** l'un exécute l'étape et l'autre est dans l'état `waiting` jusqu'à la fin du premier

#### Scenario: Ordre d'arrivée
- **WHEN** trois changes attendent le verrou
- **THEN** ils l'obtiennent dans l'ordre où ils l'ont demandé

#### Scenario: L'attente ne bloque pas la conformité
- **WHEN** la taille du pool est 1, qu'un change exécute l'étape `ui`, qu'un second attend le verrou et qu'un troisième est prêt pour la conformité
- **THEN** la conformité du troisième démarre sans attendre la fin de l'étape `ui`

#### Scenario: Libération sur échec
- **WHEN** l'étape `ui` qui détient le verrou échoue
- **THEN** le verrou est libéré et le change suivant en attente démarre

#### Scenario: Arrêt du pool pendant l'attente
- **WHEN** le pool est arrêté alors qu'un change est dans l'état `waiting`
- **THEN** la vérification est retirée de la file et son marqueur reste `pending`

### Requirement: Cycle de vie de l'application vérifiée
Pour l'étape `ui`, la plateforme SHALL choisir un port TCP libre sur la machine, substituer le jeton `{port}` dans `uiStartCommand` et dans `uiBaseUrl`, puis lancer la commande par `sh -c` dans le worktree du change, dans son propre groupe de processus, avec les variables d'environnement `PORT` et `OPENSP8C_UI_PORT` (le port), `OPENSP8C_UI_URL` (l'URL de base substituée) et `OPENSP8C_WORKSPACE_PATH` (le chemin du dépôt principal du workspace). L'application SHALL être considérée prête lorsque l'URL de base répond en HTTP avec un statut inférieur à 500, sondée périodiquement ; si elle n'est pas prête dans le délai de démarrage, ou si le processus se termine avant, l'étape SHALL échouer avec la raison et les dernières lignes de sortie de l'application. La sortie de l'application SHALL être journalisée dans le run. À la fin de l'étape, quelle qu'en soit l'issue (réussite, échec, annulation), le groupe de processus entier SHALL être arrêté : d'abord par un signal de terminaison, puis par destruction après un délai de grâce.

#### Scenario: Port substitué
- **WHEN** `uiStartCommand` vaut `npm run dev -- --port {port}` et `uiBaseUrl` vaut `http://localhost:{port}`
- **THEN** l'application est lancée avec le même port libre dans la commande et l'URL, et `PORT` porte ce port

#### Scenario: Application prête
- **WHEN** l'URL de base répond avec un statut 200
- **THEN** l'agent est lancé

#### Scenario: Redirection ou erreur client
- **WHEN** l'URL de base répond avec un statut 302 ou 404
- **THEN** l'application est considérée prête

#### Scenario: Erreur serveur persistante
- **WHEN** l'URL de base répond en 500 pendant tout le délai de démarrage
- **THEN** l'étape échoue avec la raison « l'application n'est pas prête dans le délai »

#### Scenario: Processus terminé prématurément
- **WHEN** la commande de lancement se termine avec un code d'erreur avant que l'application réponde
- **THEN** l'étape échoue avec ce code et les dernières lignes de sortie

#### Scenario: Arrêt garanti
- **WHEN** l'étape `ui` se termine, qu'elle ait réussi, échoué ou été annulée
- **THEN** aucun processus du groupe lancé par `uiStartCommand` ne reste en vie

#### Scenario: Chemin du dépôt principal exposé
- **WHEN** l'application est lancée
- **THEN** `OPENSP8C_WORKSPACE_PATH` vaut le chemin du dépôt principal du workspace, et non celui du worktree

### Requirement: Scénarios et tâches soumis à l'agent
Une fois l'application prête, la plateforme SHALL envoyer à l'agent un tour contenant l'URL de base, le dossier de preuves (variable `OPENSP8C_VERIFY_ARTIFACTS`) et la liste de ce qu'il doit vérifier : les scénarios (`#### Scenario:` avec leurs lignes WHEN et THEN) des fichiers `openspec/changes/<change>/specs/**/spec.md` de la branche, et le texte, marqueur retiré, de chaque tâche non cochée portant le marqueur `<!-- human review required -->` du `tasks.md` de la branche. La consigne système SHALL imposer la lecture seule des fichiers du dépôt, laisser l'agent choisir l'outil de navigation parmi ceux dont il dispose, et lui demander de n'exécuter que ce qui est observable dans l'interface, de déposer ses captures dans le dossier de preuves et de conclure selon le contrat de verdict.

#### Scenario: Contenu du tour
- **WHEN** un change a deux scénarios de spec et une tâche marquée non cochée « 4.2 Parcours manuel de la navigation »
- **THEN** le tour envoyé à l'agent contient les deux scénarios, le texte « 4.2 Parcours manuel de la navigation » sans le marqueur, l'URL de base et le chemin du dossier de preuves

#### Scenario: Tâches déjà cochées
- **WHEN** une tâche marquée est déjà cochée
- **THEN** elle n'est pas soumise à l'agent

#### Scenario: Change sans spec delta
- **WHEN** le change n'a aucun fichier de spec delta et aucune tâche marquée
- **THEN** le tour est envoyé sans liste de scénarios et l'agent est invité à répondre `VERDICT: SKIP` s'il ne peut rien observer

### Requirement: Contrat de verdict de la vérification UI
L'agent SHALL conclure par une ligne `VERDICT: PASS`, `VERDICT: FAIL` ou `VERDICT: SKIP`, la dernière ligne de ce type faisant foi, et SHALL déclarer chaque tâche marquée qu'il a effectivement vérifiée par une ligne `TASK-VERIFIED: <texte de la tâche>`. `PASS` SHALL signifier que tous les scénarios et toutes les tâches exécutés ont réussi ; `FAIL` qu'au moins l'un a échoué ; `SKIP` qu'aucun comportement du change n'est observable dans l'interface. L'étape SHALL réussir sur `PASS` et sur `SKIP`, et échouer dans tous les autres cas : verdict absent ou illisible, agent qui ne démarre pas, tour en erreur, inactivité au-delà du délai d'inactivité, dépassement du délai maximal de l'étape, fichier suivi par git modifié pendant la vérification, ou `SKIP` alors que des tâches marquées non cochées subsistent dans la branche. Les fichiers non suivis créés pendant l'étape, notamment par l'application elle-même, SHALL NE PAS être considérés comme une modification.

#### Scenario: Tous les scénarios réussissent
- **WHEN** l'agent conclut par `VERDICT: PASS`
- **THEN** l'étape réussit

#### Scenario: Un scénario échoue
- **WHEN** l'agent conclut par `VERDICT: FAIL`
- **THEN** l'étape échoue et le texte de sa réponse devient le rapport

#### Scenario: Rien d'observable
- **WHEN** l'agent conclut par `VERDICT: SKIP` et qu'aucune tâche marquée non cochée ne subsiste
- **THEN** l'étape réussit sans cocher de tâche

#### Scenario: SKIP refusé
- **WHEN** l'agent conclut par `VERDICT: SKIP` alors qu'une tâche marquée non cochée subsiste
- **THEN** l'étape échoue avec la raison « tâches de validation humaine non vérifiées »

#### Scenario: Verdict absent
- **WHEN** la réponse de l'agent ne contient aucune ligne `VERDICT:`
- **THEN** l'étape échoue avec la raison « verdict absent »

#### Scenario: Fichier suivi modifié
- **WHEN** un fichier suivi du worktree est modifié pendant la vérification et que l'agent conclut par `PASS`
- **THEN** l'étape échoue avec la raison « le vérificateur a modifié le worktree » et les fichiers sont laissés en place

#### Scenario: Fichier non suivi créé par l'application
- **WHEN** l'application crée un fichier non suivi (journal, base de données locale) et que l'agent conclut par `PASS`
- **THEN** l'étape réussit

#### Scenario: Délai maximal
- **WHEN** l'étape dépasse le délai maximal d'exécution
- **THEN** l'agent et l'application sont arrêtés et l'étape échoue avec la raison de dépassement

### Requirement: Coche des tâches de validation humaine vérifiées
Après un verdict `PASS`, et seulement alors, la plateforme SHALL cocher dans le `tasks.md` de la branche `feature/<change>` chaque tâche non cochée portant le marqueur de validation humaine dont le texte (marqueur retiré, espaces de bordure retirés) est identique à celui d'une ligne `TASK-VERIFIED:`. Chaque coche SHALL produire un commit propre dans la branche, selon la convention des coches de la branche, et le marqueur SHALL être conservé sur la ligne. L'agent ne SHALL jamais écrire dans `tasks.md`. Une ligne `TASK-VERIFIED:` qui ne correspond à aucune tâche marquée non cochée (texte inconnu, tâche non marquée, tâche déjà cochée) SHALL être ignorée et mentionnée dans le rapport. Les tâches non déclarées vérifiées SHALL rester décochées. Sur `FAIL`, `SKIP` ou en cas d'échec de l'étape, aucune tâche SHALL être cochée.

#### Scenario: Tâche vérifiée cochée
- **WHEN** l'agent conclut par `VERDICT: PASS` avec `TASK-VERIFIED: 4.2 Parcours manuel de la navigation`
- **THEN** la tâche 4.2 est cochée dans la branche par un commit dédié, avec son marqueur conservé

#### Scenario: Plusieurs tâches
- **WHEN** l'agent déclare vérifiées deux des trois tâches marquées
- **THEN** deux commits de coche sont créés et la troisième reste décochée

#### Scenario: Aucune coche sur échec
- **WHEN** l'agent déclare une tâche vérifiée mais conclut par `VERDICT: FAIL`
- **THEN** aucune tâche n'est cochée

#### Scenario: Ligne sans correspondance
- **WHEN** l'agent déclare `TASK-VERIFIED: tâche inventée`
- **THEN** aucune tâche n'est cochée et le rapport mentionne que la ligne a été ignorée

#### Scenario: Tâche non marquée
- **WHEN** l'agent déclare vérifiée une tâche d'implémentation décochée sans marqueur
- **THEN** elle n'est pas cochée et la ligne est mentionnée dans le rapport comme ignorée

#### Scenario: Échec d'un commit de coche
- **WHEN** le commit de la coche d'une tâche échoue
- **THEN** le fichier est restauré, l'étape échoue avec la raison de l'échec du commit et les coches déjà commitées restent

### Requirement: Preuves de la vérification UI
Le dossier de preuves d'un run `verify` SHALL se trouver à côté de son journal, sous `conversations/<workspace>/<change>/verify/<horodatage>/` ; il SHALL être créé avant le lancement de l'agent et NE SHALL PAS apparaître comme un run dans la liste des runs du change. Le rapport (`GET …/verification/report`) SHALL lister les preuves du run (`artifacts`, avec `name` et `size`), limitées aux images `png`, `jpg`, `jpeg` et `webp` d'au plus 5 Mo, au nombre de 50 au maximum. Le backend SHALL servir une preuve par `GET /workspaces/{id}/changes/{name}/verification/artifacts/{run}/{name}` avec le type de contenu de l'image, `404` si elle n'existe pas, et `400` pour un nom ou un identifiant de run contenant un séparateur de chemin, `..` ou qui n'est pas un simple nom de fichier. Les preuves SHALL être supprimées avec les journaux du change par la rétention existante.

#### Scenario: Captures listées
- **WHEN** l'agent dépose deux captures `nav-1.png` et `nav-2.png` dans le dossier de preuves
- **THEN** le rapport du run les liste avec leur taille

#### Scenario: Capture servie
- **WHEN** l'utilisateur demande `…/verification/artifacts/<run>/nav-1.png`
- **THEN** la réponse contient l'image avec le type de contenu `image/png`

#### Scenario: Chemin hors du dossier
- **WHEN** la requête contient `../` dans le nom ou dans le run
- **THEN** elle est refusée en `400` et aucun fichier hors du dossier n'est lu

#### Scenario: Fichier d'un autre type
- **WHEN** l'agent dépose un fichier `notes.txt` ou une image de plus de 5 Mo
- **THEN** il n'est ni listé ni servi

#### Scenario: Dossier invisible dans les runs
- **WHEN** un run `verify` possède un dossier de preuves
- **THEN** l'onglet Log du change liste un seul run, sans entrée pour le dossier

#### Scenario: Suppression avec le change
- **WHEN** les journaux d'un change archivé sont purgés par la rétention
- **THEN** ses dossiers de preuves sont supprimés avec eux

### Requirement: Rapport par étape
Chaque étape de la vérification SHALL être journalisée comme son propre run `verify`, dont les marqueurs de début et de fin portent le nom de l'étape (`conformity` ou `ui`). `GET …/verification/report` SHALL retourner par défaut le run le plus récent, qui est celui qui a décidé de l'issue, et SHALL accepter un paramètre `step` retournant le run le plus récent de cette étape (`404` s'il n'existe pas). Le rapport d'une étape `ui` SHALL contenir, outre le texte de l'agent, la liste des lignes `TASK-VERIFIED:` retenues et ignorées.

#### Scenario: Rapport par défaut après un échec UI
- **WHEN** la conformité a réussi puis l'étape `ui` a échoué
- **THEN** `GET …/verification/report` retourne le run de l'étape `ui`

#### Scenario: Rapport d'une étape précise
- **WHEN** l'utilisateur demande `…/report?step=conformity`
- **THEN** la réponse contient le dernier run de conformité

#### Scenario: Étape sans run
- **WHEN** l'utilisateur demande `…/report?step=ui` pour un change dont l'étape `ui` n'a jamais tourné
- **THEN** la réponse est `404`

### Requirement: Affichage de l'état `waiting`
La carte d'un change dont `verification_state` vaut `waiting` SHALL afficher un badge « en attente de l'interface » dans la colonne Verifying, distinct du badge « en file » ; le bandeau de vérification du DetailPanel SHALL afficher le même état. Un état `running` à l'étape `ui` SHALL être libellé « vérification UI en cours ».

#### Scenario: Badge d'attente
- **WHEN** une vérification attend le verrou UI
- **THEN** sa carte affiche le badge « en attente de l'interface »

#### Scenario: Badge d'exécution UI
- **WHEN** l'étape `ui` s'exécute
- **THEN** sa carte affiche « vérification UI en cours »
