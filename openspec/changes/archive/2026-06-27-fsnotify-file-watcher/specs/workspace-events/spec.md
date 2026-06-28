## ADDED Requirements

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
