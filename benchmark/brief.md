# Brief : statistiques d'un change

Ajoute au backend Go un endpoint HTTP en lecture seule :

    GET /api/workspaces/{id}/changes/{name}/stats

Il renvoie, pour le change `{name}` du workspace `{id}`, un objet JSON qui résume son avancement et son activité d'agent :

```json
{
  "tasks_total": 4,
  "tasks_done": 3,
  "progress_percent": 75,
  "runs_count": 3,
  "total_duration_seconds": 120
}
```

- `tasks_total`, `tasks_done` et `progress_percent` décrivent l'avancement des tâches du change.
- `runs_count` est le nombre de runs du change.
- `total_duration_seconds` est la durée cumulée de ces runs, en secondes.

Si le change n'existe pas dans le workspace, l'endpoint répond 404 (comme `GET /api/workspaces/{id}/changes/{name}`).

Le change s'appelle `change-stats-endpoint`. Le code existant du backend (`backend/internal/api`, `backend/internal/openspec`, `backend/internal/conversation`) fait foi pour les conventions. `go test ./...` doit rester vert dans `backend/`, avec des tests pour ce que tu ajoutes.
