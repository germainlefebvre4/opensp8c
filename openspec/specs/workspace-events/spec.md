# workspace-events Specification

## Purpose

Diffusion d'événements temps-réel SSE pour les changements de fichiers, l'état du pool et l'activité dans le workspace.

## Requirements

### Requirement: Stream SSE d'événements de changement par workspace
Le backend SHALL exposer un endpoint SSE `/api/workspaces/{id}/events` qui pousse des événements temps-réel quand les fichiers OpenSpec du workspace changent sur le filesystem. Le stream SHALL rester ouvert jusqu'à déconnexion du client. Le backend SHALL surveiller récursivement `openspec/changes/` via fsnotify et envoyer un événement typé pour chaque changement détecté, avec debounce de 150ms par change pour absorber les rafales d'écritures.

#### Scenario: Connexion au stream SSE
- **WHEN** le client envoie GET `/api/workspaces/{id}/events`
- **THEN** le serveur répond avec `Content-Type: text/event-stream`, `Cache-Control: no-cache`, et maintient la connexion ouverte

#### Scenario: Fichier modifié dans un change existant
- **WHEN** `tasks.md` ou `.openspec.yaml` d'un change est modifié sur le filesystem
- **THEN** après 150ms de silence sur ce change, le serveur envoie `event: change_updated\ndata: {"name":"<change-name>"}\n\n`

#### Scenario: Nouveau répertoire de change créé
- **WHEN** un nouveau sous-répertoire est créé dans `openspec/changes/`
- **THEN** le serveur envoie `event: change_created\ndata: {"name":"<change-name>"}\n\n`

#### Scenario: Répertoire de change supprimé ou archivé
- **WHEN** un sous-répertoire de `openspec/changes/` est supprimé ou déplacé dans `archive/`
- **THEN** le serveur envoie `event: change_deleted\ndata: {"name":"<change-name>"}\n\n`

#### Scenario: Keepalive ping
- **WHEN** aucun événement n'a été envoyé depuis 30 secondes
- **THEN** le serveur envoie `event: ping\ndata: {}\n\n` pour maintenir la connexion

#### Scenario: Déconnexion du client
- **WHEN** le client ferme la connexion SSE
- **THEN** le serveur détecte la déconnexion via le contexte HTTP et libère les ressources associées (canal broadcaster retiré)

### Requirement: Watching lazy-recursive du filesystem
Le WatcherService SHALL démarrer en observant uniquement `openspec/` (garanti présent). Il SHALL étendre son périmètre dynamiquement : quand `openspec/changes/` est créé, il l'ajoute au watcher et scan les sous-répertoires existants ; quand un nouveau répertoire de change apparaît, il l'ajoute au watcher. Le watching de `openspec/changes/archive/` SHALL être ajouté quand ce répertoire est créé.

#### Scenario: Workspace sans openspec/changes/ au démarrage
- **WHEN** le watcher démarre pour un workspace dont `openspec/changes/` n'existe pas encore
- **THEN** le watcher observe `openspec/` et attend la création de `changes/` sans erreur

#### Scenario: openspec/changes/ créé après démarrage du watcher
- **WHEN** la CLI crée `openspec/changes/` pour la première fois
- **THEN** le watcher l'ajoute automatiquement à son périmètre et commence à surveiller les changes créés à l'intérieur

#### Scenario: Changes existants au démarrage du watcher
- **WHEN** le watcher démarre et `openspec/changes/` existe déjà avec des sous-répertoires
- **THEN** chaque sous-répertoire de change existant est ajouté au watcher ; aucun événement `change_created` n'est émis pour les changes déjà présents

### Requirement: Événement SSE de mise à jour de l'état du pool d'agents
Le backend SHALL émettre, sur le même stream SSE `/api/workspaces/{id}/events`, un événement `pool_updated` chaque fois que l'état du pool d'agents change : démarrage du pool, arrêt du pool, ou changement de statut d'un worker (`idle`, `working`, `testing`, `healing`, ou `paused`) ou de son change assigné. Cet événement est déclenché par les transitions d'état en mémoire du gestionnaire de pool, indépendamment de toute modification de fichier sur le filesystem, et ne SHALL PAS être soumis au debounce de 150ms utilisé pour les événements liés aux fichiers.

#### Scenario: Démarrage du pool
- **WHEN** le pool d'agents démarre suite à un appel `POST /workspaces/{id}/pool/start`
- **THEN** le serveur envoie `event: pool_updated\ndata: {}\n\n` à tous les clients abonnés au stream de ce workspace

#### Scenario: Arrêt du pool
- **WHEN** le pool d'agents s'arrête suite à un appel `POST /workspaces/{id}/pool/stop`
- **THEN** le serveur envoie `event: pool_updated\ndata: {}\n\n` à tous les clients abonnés au stream de ce workspace

#### Scenario: Changement de statut d'un worker
- **WHEN** un worker du pool change de statut (par exemple de `working` à `testing`, ou se voit assigner un nouveau change)
- **THEN** le serveur envoie `event: pool_updated\ndata: {}\n\n` à tous les clients abonnés au stream de ce workspace

#### Scenario: Réception côté client
- **WHEN** le frontend reçoit un événement `pool_updated` sur le stream SSE
- **THEN** il invalide et recharge l'état du pool (`GET /workspaces/{id}/pool/status`) ainsi que la liste des changes, pour refléter à jour le bouton d'en-tête, le panneau d'état, et les badges `worker_active` sur les cartes

### Requirement: Événement SSE d'ajout d'une entrée d'activité
Le backend SHALL émettre, sur le même stream SSE `/api/workspaces/{id}/events`, un événement `activity_appended` chaque fois qu'une nouvelle entrée est ajoutée à l'`ActivityStore` d'un change (toggle de tâche, déclenchement de run, reset de tâches, transition de statut worker, commit git détecté). Cet événement inclut le nom du change concerné et ne SHALL PAS être soumis au debounce de 150ms utilisé pour les événements liés aux fichiers.

#### Scenario: Ajout d'une entrée d'activité
- **WHEN** une nouvelle entrée est ajoutée à l'`ActivityStore` du change `<changeName>`
- **THEN** le serveur envoie `event: activity_appended\ndata: {"name":"<changeName>"}\n\n` à tous les clients abonnés au stream de ce workspace

#### Scenario: Réception côté client
- **WHEN** le frontend reçoit un événement `activity_appended` pour le change actuellement affiché dans le DetailPanel
- **THEN** il invalide et recharge le flux fusionné (`GET /changes/{name}/activity`) pour refléter la nouvelle entrée dans l'onglet Conversation
