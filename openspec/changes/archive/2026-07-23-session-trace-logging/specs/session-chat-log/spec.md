## ADDED Requirements

### Requirement: Journalisation stdin/stdout/stderr d'une session de chat

Pour toute session de chat interactive (nommée via `session.Manager.Start` ou anonyme via `session.Manager.StartAnonymous`), le backend SHALL ouvrir un fichier JSONL dédié à la création de la session et y écrire chaque message stdin envoyé, chaque message stdout reçu et chaque ligne stderr émise par le subprocess, chacune horodatée et taguée par sa direction (`in`, `out`, `err`).

#### Scenario: Ouverture du fichier de log au démarrage de la session
- **WHEN** une session de chat démarre (nommée ou anonyme)
- **THEN** un fichier `<ts>.jsonl` est créé pour cette session, où `<ts>` est l'horodatage de démarrage

#### Scenario: Message utilisateur journalisé
- **WHEN** un message utilisateur est écrit sur le stdin du subprocess
- **THEN** une ligne `{"ts":"...","dir":"in","data":<message brut>}` est ajoutée au fichier de log de la session

#### Scenario: Message assistant journalisé
- **WHEN** le subprocess produit une ligne sur stdout
- **THEN** une ligne `{"ts":"...","dir":"out","data":<ligne brute>}` est ajoutée au fichier de log de la session

#### Scenario: Sortie stderr journalisée
- **WHEN** le subprocess écrit une ligne sur stderr
- **THEN** une ligne `{"ts":"...","dir":"err","data":"<texte>"}` est ajoutée au fichier de log de la session, en plus du `log.Printf` existant côté serveur

#### Scenario: Écritures concurrentes sérialisées
- **WHEN** stdin, stdout et stderr produisent des lignes à journaliser au même moment
- **THEN** les écritures dans le fichier de log sont sérialisées de sorte qu'aucune ligne n'est tronquée ou entrelacée

### Requirement: Un fichier de log par cycle de vie de session

Le fichier de log SHALL correspondre exactement au cycle de vie d'un subprocess (une `Session`), indépendamment du nombre de reconnexions WebSocket qui s'y attachent pendant sa durée de vie.

#### Scenario: Reconnexion WebSocket sur une session déjà active
- **WHEN** un client WebSocket se reconnecte à une session déjà en cours (subprocess non arrêté)
- **THEN** les nouveaux messages continuent d'être écrits dans le même fichier de log, sans en créer un nouveau

#### Scenario: Fermeture du fichier à l'arrêt de la session
- **WHEN** la session est arrêtée (`Session.Stop`, expiration, ou fin normale du subprocess)
- **THEN** le fichier de log est fermé proprement (flush garanti)

### Requirement: Emplacement de stockage distinct pour les sessions nommées et anonymes

Les logs de chat des sessions nommées SHALL être stockés sous `conversations/<workspaceId>/<changeName>/chat/<ts>.jsonl`. Les logs de chat des sessions d'exploration anonymes (sans `changeName` avant promotion) SHALL être stockés sous `conversations/<workspaceId>/_explore/<ghostSessionId>/chat/<ts>.jsonl`.

#### Scenario: Session nommée
- **WHEN** une session de chat démarre pour un change existant `<name>`
- **THEN** son log est écrit sous `conversations/<workspaceId>/<name>/chat/<ts>.jsonl`

#### Scenario: Session anonyme
- **WHEN** une session d'exploration anonyme démarre avec l'identifiant `<ghostSessionId>`
- **THEN** son log est écrit sous `conversations/<workspaceId>/_explore/<ghostSessionId>/chat/<ts>.jsonl`
