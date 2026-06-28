## Context

Le `Layout` React gère la sélection du workspace actif via un `useState<string | null>(null)`. Ce state est éphémère : il disparaît au refresh. L'app fait alors un fallback sur `workspaces[0]`, ce qui perturbe l'utilisateur s'il travaillait sur un autre projet.

React Router est déjà en place (`BrowserRouter`). L'API `useSearchParams` permet de lire et écrire des query params de manière réactive, sans dépendance supplémentaire.

## Goals / Non-Goals

**Goals:**
- Workspace actif toujours reflété dans l'URL (`?workspace=<id>`)
- Refresh = même workspace sélectionné
- Navigation entre routes (Kanban ↔ Specs) préserve le param

**Non-Goals:**
- Persistance cross-browser ou cross-session (pas de `localStorage`)
- Partage d'URL entre machines (les IDs sont locaux)
- Gestion multi-onglets

## Decisions

### Query param plutôt que `localStorage`

L'URL est la source de vérité naturelle pour l'état de navigation. Avantages : bookmarkable, cohérent avec le modèle React Router, historique navigateur fonctionnel, pas de désynchronisation possible entre state et stockage.

Alternatives considérées :
- `localStorage` : plus simple à coder mais état caché, pas bookmarkable
- `sessionStorage` : même problème, isolé par onglet

### Propagation du param via `search: searchParams.toString()`

Les `NavLink` vers `/` et `/specs` utilisent `to={{ pathname, search: searchParams.toString() }}` pour transporter automatiquement tous les query params existants. Simple et évolutif si d'autres params sont ajoutés plus tard.

### `useEffect` pour initialiser l'URL

Au premier chargement sans `?workspace`, un `useEffect` pousse le workspace par défaut dans l'URL avec `replace: true` (pas de nouvelle entrée d'historique). Cela garantit que l'URL est toujours cohérente avec la sélection affichée.

### Fallback silencieux sur workspace supprimé

Si `?workspace=<id>` ne correspond à aucun workspace connu, on ignore silencieusement le param et on sélectionne `workspaces[0]`. On met ensuite à jour l'URL pour refléter ce fallback.

## Risks / Trade-offs

- **Param invalide dans l'URL** → Fallback sur `workspaces[0]` + correction de l'URL. Pas d'erreur visible.
- **URL partagée entre machines** → L'ID est local, le param sera ignoré et l'URL corrigée. Comportement correct.
- **`useEffect` déclenché au mauvais moment** → Utiliser `replace: true` évite de polluer l'historique. Guard sur `!searchParams.get('workspace')` évite les boucles.
