## ADDED Requirements

### Requirement: Afficher les tags dans le DetailPanel
Le `DetailPanel` SHALL afficher une section **Tags** lorsqu'un change possède une section `tags` dans son `.openspec.yaml`. La section affiche le type applicatif, le niveau de complexité, et la liste des composants touchés. Si le change n'a pas encore de tags, la section est absente sans erreur.

#### Scenario: Change avec tags complets
- **WHEN** le DetailPanel s'ouvre pour un change possédant `tags.type`, `tags.complexity` et `tags.components`
- **THEN** la section Tags est affichée avec le badge type, l'indicateur de complexité (points ou étoiles sur 5), et les chips de composants

#### Scenario: Change sans tags
- **WHEN** le DetailPanel s'ouvre pour un change sans section `tags`
- **THEN** la section Tags est absente du panneau, sans message d'erreur

#### Scenario: Bouton de retag dans le DetailPanel
- **WHEN** l'utilisateur clique sur l'icône de rafraîchissement des tags dans le DetailPanel
- **THEN** une requête `POST /api/workspaces/{id}/changes/{name}/retag` est déclenchée et les tags se mettent à jour une fois la dérivation terminée

## MODIFIED Requirements

### Requirement: Endpoint de détail d'un change
Le backend SHALL exposer un endpoint `GET /api/workspaces/{id}/changes/{name}` retournant le détail complet d'un change : métadonnées, liste des tâches avec texte et état, contenu des artifacts `proposal.md` et `design.md`, et tags sémantiques si présents.

#### Scenario: Change existant
- **WHEN** une requête `GET /api/workspaces/{id}/changes/{name}` est effectuée pour un change existant
- **THEN** la réponse contient `name`, `kanban_status`, `tasks_done`, `tasks_total`, `tasks` (tableau d'objets `{ text, done }`), `artifacts` (`{ proposal, design }` avec chaîne vide si absent), et `tags` (objet optionnel `{ type, complexity, components[] }` ou `null` si absent)

#### Scenario: Change inexistant
- **WHEN** la requête cible un change qui n'existe pas
- **THEN** le backend retourne HTTP 404
