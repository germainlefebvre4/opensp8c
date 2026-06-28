## Context

Le frontend poll actuellement l'API toutes les 5 secondes (`useChanges` → `refetchInterval: 5000`). Le backend lit le filesystem à chaque requête sans cache. Claude Code écrit des fichiers en rafales lors des sessions actives — l'utilisateur perçoit un délai jusqu'à 5s avant de voir le kanban se mettre à jour.

L'infrastructure WebSocket (`nhooyr.io/websocket`) existe déjà pour le panel Explore. Le backend est Go/chi. Le frontend utilise React Query (TanStack Query v5).

## Goals / Non-Goals

**Goals:**
- Réactivité instantanée : le kanban se met à jour dès que Claude Code écrit un fichier
- Événements précis : `change_updated`, `change_created`, `change_deleted` avec le nom du change
- Invalidation chirurgicale React Query (pas de refetch global)
- Aucun polling résiduel pour les changes actifs

**Non-Goals:**
- Remplacer le polling `useWorkspaces` (15s) — les mutations l'invalident déjà, le polling est un filet de sécurité acceptable
- Surveiller les specs ou d'autres répertoires OpenSpec
- Persistance des événements manqués pendant une déconnexion SSE (reconnect = refetch initial)

## Decisions

### D1 — SSE plutôt que WebSocket pour les événements

SSE est unidirectionnel (server → client), HTTP/1.1 natif, reconnect automatique côté browser. La feature "events workspace" n'a pas besoin de bidirectionnel — c'est exactement le cas d'usage de SSE. WebSocket existe déjà dans le projet pour Explore (bidirectionnel nécessaire là-bas).

Alternatif rejeté : WebSocket. Overhead de setup/handshake non justifié pour un canal read-only.

### D2 — Watching lazy-recursive plutôt que statique

`openspec/changes/` peut ne pas exister à l'ajout du workspace. Le watcher doit démarrer sur `openspec/` (toujours présent, vérifié par `workspace.Validate()`), puis étendre son périmètre au fur et à mesure :

```
openspec/           ← watch au démarrage
└── changes/        ← ajouté quand créé ; scan des sous-dirs existants
    ├── <name>/     ← ajouté pour chaque change (actif ou à la découverte)
    └── archive/    ← ajouté quand créé ; pas besoin de watcher les sous-dirs
```

Alternatif rejeté : watch récursif via `fsnotify.WithRecurse`. Moins de contrôle sur les événements à ignorer, et génère du bruit sur les fichiers non-pertinents.

### D3 — Debounce 150ms par change

Claude Code fait des rafales d'écritures (ex. `tasks.md` + `.openspec.yaml` en quelques ms). Un debounce par change (clé = nom du change) évite les rafales d'événements SSE. 150ms = assez court pour paraître instantané, assez long pour absorber une rafale d'écriture.

### D4 — Broadcaster en mémoire (pas de message broker)

Un `Broadcaster` par workspace maintient une liste de canaux SSE (`[]chan Event`). Quand un événement est debounced, il est envoyé à tous les canaux. Approche simple, pas de dépendance externe.

```
WatcherService
└── workspace-abc/
    ├── fsnotify.Watcher
    ├── debouncers: map[changeName]*time.Timer
    └── broadcaster: []chan Event
```

### D5 — Format d'événement SSE

```
event: change_updated
data: {"name":"my-feature"}

event: change_created
data: {"name":"my-feature"}

event: change_deleted
data: {"name":"my-feature"}

event: ping
data: {}
```

Le frontend identifie le type via `event:` (SSE standard) et invalide les queries React Query correspondantes.

### D6 — Ping keepalive 30s

Les proxies HTTP coupent les connexions SSE inactives. Un ping toutes les 30s maintient la connexion. Le client l'ignore (pas de handler `onmessage` pour `ping`).

## Risks / Trade-offs

- **Rafale d'événements au démarrage** → Mitigé par le fait que le watcher est initialisé *après* le scan initial : on ne génère pas d'événements pour les changes déjà présents lors du démarrage.
- **Memory leak si le client ne se déconnecte pas proprement** → Le handler SSE MUST détecter `r.Context().Done()` et retirer le canal du broadcaster.
- **Multiple onglets ouverts sur le même workspace** → Chaque onglet crée sa propre connexion SSE. Le broadcaster fan-out vers tous — comportement correct.
- **fsnotify inotify limit (Linux)** → Par défaut 8192 watches. Chaque workspace actif ajoute ~3-5 watches (openspec/, changes/, archive/, N × change/). À 100 workspaces avec 20 changes chacun ≈ 2300 watches. Confortable. Documenté dans le README si besoin.

## Migration Plan

1. Déployer le backend avec l'endpoint SSE (l'ancien endpoint polling reste intact)
2. Déployer le frontend avec `useWorkspaceEvents` + retrait de `refetchInterval`
3. Pas de rollback complexe : si SSE échoue, React Query fait un fetch initial au montage — données pas totalement obsolètes, juste pas live
4. Aucun changement de schéma de données, aucune migration de base de données

## Open Questions

- Faut-il watcher `openspec/changes/archive/` pour les événements `change_deleted` lors d'un archivage ? L'archivage via le bouton "Sync & Archive" passe par l'API (qui invalide déjà la query). L'archivage CLI serait le seul cas non couvert → **décision : oui, watcher archive/ pour la cohérence.**
