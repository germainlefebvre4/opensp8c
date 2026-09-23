# Spec Delta

## MODIFIED Requirements

### Requirement: Afficher les tags dans le DetailPanel
Le `DetailPanel` SHALL afficher une section **Tags** lorsqu'un change possède une section `tags` dans son `.openspec.yaml`. La section affiche un badge par valeur du tableau `tags.type` (ou aucun badge de type si la liste est vide), le niveau de complexité, et la liste des composants touchés. Si le change n'a pas encore de tags, la section est absente sans erreur.

#### Scenario: Change avec tags complets et un seul type
- **WHEN** le DetailPanel s'ouvre pour un change possédant `tags.type: [frontend]`, `tags.complexity` et `tags.components`
- **THEN** la section Tags est affichée avec un badge type "frontend", l'indicateur de complexité (points ou étoiles sur 5), et les chips de composants

#### Scenario: Change avec plusieurs valeurs de type
- **WHEN** le DetailPanel s'ouvre pour un change possédant `tags.type: [frontend, backend]`
- **THEN** la section Tags affiche deux badges type distincts, un pour "frontend" et un pour "backend"

#### Scenario: Change sans tags
- **WHEN** le DetailPanel s'ouvre pour un change sans section `tags`
- **THEN** la section Tags est absente du panneau, sans message d'erreur

#### Scenario: Bouton de retag dans le DetailPanel
- **WHEN** l'utilisateur clique sur l'icône de rafraîchissement des tags dans le DetailPanel
- **THEN** une requête `POST /api/workspaces/{id}/changes/{name}/retag` est déclenchée et les tags se mettent à jour une fois la dérivation terminée

### Requirement: Endpoint de détail d'un change
Le backend SHALL exposer un endpoint `GET /api/workspaces/{id}/changes/{name}` retournant le détail complet d'un change : métadonnées, liste des tâches avec texte et état, contenu des artifacts `proposal.md` et `design.md`, et tags sémantiques si présents.

#### Scenario: Change existant
- **WHEN** une requête `GET /api/workspaces/{id}/changes/{name}` est effectuée pour un change existant
- **THEN** la réponse contient `name`, `kanban_status`, `tasks_done`, `tasks_total`, `tasks` (tableau d'objets `{ text, done }`), `artifacts` (`{ proposal, design }` avec chaîne vide si absent), et `tags` (objet optionnel `{ type: string[], complexity, components[] }` ou `null` si absent)

#### Scenario: Change inexistant
- **WHEN** la requête cible un change qui n'existe pas
- **THEN** le backend retourne HTTP 404
