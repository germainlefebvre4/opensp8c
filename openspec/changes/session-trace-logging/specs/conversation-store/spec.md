## ADDED Requirements

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
