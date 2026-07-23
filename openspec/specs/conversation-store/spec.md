## Purpose

Spec du stockage des logs de conversation par run ff : persistance JSONL, endpoints de listing et récupération, et affichage dans le DetailPanel.
## Requirements
### Requirement: Stockage des logs de conversation par run
Le backend SHALL maintenir un `ConversationStore` qui persiste les messages d'agent sous forme de fichiers JSONL horodatés par run, à l'emplacement `<config-dir>/conversations/<workspaceID>/<changeName>/<kind>/<timestamp>.jsonl`. Chaque ligne du fichier SHALL être un message brut tel que produit par le subprocess (même format que les messages en mémoire des sessions explore). Le `<timestamp>` SHALL être au format RFC3339 UTC avec secondes (ex: `2026-06-29T14-32-00Z`), utilisé comme nom de fichier.

#### Scenario: Création d'un nouveau fichier log au démarrage d'un run
- **WHEN** un subprocess ff démarre
- **THEN** un nouveau fichier `<timestamp>.jsonl` est créé dans `conversations/<wsID>/<changeName>/ff/` avec le timestamp de démarrage

#### Scenario: Append de chaque message stdout
- **WHEN** le subprocess ff produit une ligne sur stdout
- **THEN** cette ligne est ajoutée (append) au fichier JSONL du run courant

#### Scenario: Fichier partiel en cas d'arrêt brutal
- **WHEN** le subprocess ff est interrompu (crash, kill)
- **THEN** le fichier JSONL contient les lignes reçues avant l'interruption ; aucune ligne corrompue n'est écrite (append atomique ligne par ligne)

### Requirement: Listage des runs par changement et kind
Le backend SHALL exposer `GET /api/workspaces/{id}/changes/{name}/conversations/{kind}` retournant la liste des runs disponibles pour ce kind, triés par ordre antéchronologique, avec pour chaque run : `ts` (timestamp ISO8601) et `messageCount` (nombre de lignes dans le fichier).

#### Scenario: Plusieurs runs ff disponibles
- **WHEN** le frontend appelle `GET /changes/{name}/conversations/ff`
- **THEN** la réponse liste tous les runs ff pour ce changement, du plus récent au plus ancien

#### Scenario: Aucun run disponible
- **WHEN** aucun run ff n'a encore été effectué pour ce changement
- **THEN** la réponse retourne un tableau vide sans erreur

### Requirement: Récupération d'un run spécifique
Le backend SHALL exposer `GET /api/workspaces/{id}/changes/{name}/conversations/{kind}/{ts}` retournant les messages du run identifié par `{ts}`, sous forme d'un tableau de lignes JSON brutes dans l'ordre d'écriture.

#### Scenario: Run existant
- **WHEN** le frontend appelle `GET /changes/{name}/conversations/ff/2026-06-29T14-32-00Z`
- **THEN** la réponse retourne `{ "ts": "2026-06-29T14-32-00Z", "messages": [...] }` avec toutes les lignes du fichier JSONL

#### Scenario: Run introuvable
- **WHEN** le timestamp ne correspond à aucun fichier existant
- **THEN** le backend retourne 404 Not Found

### Requirement: Affichage du log ff dans le DetailPanel
Le DetailPanel SHALL exposer un onglet **"Log"** listant les runs ff disponibles pour le changement ouvert. Le run le plus récent SHALL être sélectionné par défaut. Les messages SHALL être affichés en lecture seule dans le même format de rendu que les messages explore (réutilisation du renderer existant). Aucune zone de saisie n'est présente dans cet onglet.

#### Scenario: Onglet Log avec runs disponibles
- **WHEN** l'utilisateur ouvre le DetailPanel d'un changement ayant au moins un run ff
- **THEN** l'onglet "Log" est visible et affiche le run le plus récent par défaut

#### Scenario: Sélection d'un run antérieur
- **WHEN** l'utilisateur sélectionne un run plus ancien dans la liste
- **THEN** les messages de ce run s'affichent à la place du run courant

#### Scenario: Onglet Log sans runs
- **WHEN** le changement n'a encore aucun run ff
- **THEN** l'onglet "Log" est visible mais affiche un message vide ("Aucun run ff pour l'instant")

### Requirement: Résolution de chemin pour les sessions d'exploration pré-promotion

Le `ConversationStore` SHALL exposer une résolution de chemin dédiée aux sessions d'exploration anonymes, indexée par `ghostSessionId` plutôt que par `changeName`, sous `conversations/<workspaceId>/_explore/<ghostSessionId>/<kind>/<ts>.jsonl`.

#### Scenario: Ouverture d'un run pour une exploration anonyme
- **WHEN** le backend ouvre un fichier de log pour une session d'exploration anonyme d'id `<ghostSessionId>`
- **THEN** le fichier est créé sous `conversations/<workspaceId>/_explore/<ghostSessionId>/chat/<ts>.jsonl`

### Requirement: Déplacement des logs d'une exploration vers son change promu

Le `ConversationStore` SHALL exposer une opération qui déplace l'intégralité du dossier de logs d'une exploration (`_explore/<ghostSessionId>/`) vers le dossier de logs du change nouvellement créé (`<changeName>/`), fusionnant les fichiers si le dossier cible existe déjà.

#### Scenario: Promotion réussie
- **WHEN** un ghost d'id `<ghostSessionId>` est promu avec succès en change `<changeName>`
- **THEN** le contenu de `conversations/<workspaceId>/_explore/<ghostSessionId>/` est déplacé vers `conversations/<workspaceId>/<changeName>/`, et le dossier `_explore/<ghostSessionId>/` n'existe plus

#### Scenario: Dossier cible déjà existant
- **WHEN** la promotion déplace les logs vers un `changeName` pour lequel un dossier de logs existe déjà
- **THEN** les fichiers de l'exploration sont ajoutés au dossier existant sans écraser les fichiers déjà présents

#### Scenario: Aucun log d'exploration à déplacer
- **WHEN** un ghost promu n'a jamais eu de log de chat écrit (cas limite)
- **THEN** la promotion n'échoue pas et ne crée aucun dossier de logs vide

