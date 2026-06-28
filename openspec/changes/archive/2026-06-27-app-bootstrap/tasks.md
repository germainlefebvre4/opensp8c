## 1. Initialisation du projet

- [x] 1.1 Créer la structure de répertoires racine : `backend/`, `frontend/`, `config.yaml` vide
- [x] 1.2 Initialiser le module Go `github.com/glefebvre/opensp8c` dans `backend/`
- [x] 1.3 Initialiser l'application React v19 avec Vite + TypeScript dans `frontend/`
- [x] 1.4 Créer le `Makefile` racine avec targets : `dev` (lance backend + frontend), `build`

## 2. Backend — Foundation

- [x] 2.1 Ajouter les dépendances Go : `go-chi/chi`, `gopkg.in/yaml.v3`, websocket lib (`nhooyr.io/websocket`)
- [x] 2.2 Créer `backend/internal/config/config.go` — struct `Config` avec liste de workspaces, `Load()` depuis `config.yaml`, `Save()` en écriture
- [x] 2.3 Créer `backend/cmd/server/main.go` — démarrage serveur HTTP sur port configurable, graceful shutdown sur SIGINT/SIGTERM
- [x] 2.4 Créer `backend/internal/api/router.go` — Chi router, CORS permissif en dev, middleware logging, groupe de routes `/api`

## 3. Backend — Workspace API

- [x] 3.1 Créer `backend/internal/workspace/workspace.go` — struct `Workspace` (id UUID, name, path), fonction `Validate(path)` qui vérifie la présence de `openspec/`
- [x] 3.2 Créer `backend/internal/api/handlers/workspace.go` — `GET /api/workspaces`, `POST /api/workspaces` (body : path), `DELETE /api/workspaces/:id`
- [x] 3.3 Implémenter la génération d'un ID stable par workspace (hash du path absolu)

## 4. Backend — Kanban API

- [x] 4.1 Créer `backend/internal/openspec/change.go` — lecture des répertoires `openspec/changes/` (hors `archive/`), parsing `.openspec.yaml` (champs `schema`, `created`, `kanban_status`), parsing `tasks.md` pour le compte `[x]` / total
- [x] 4.2 Créer `backend/internal/api/handlers/kanban.go` — `GET /api/workspaces/:id/changes` retourne la liste des changes avec progression et `kanban_status`
- [x] 4.3 Implémenter `PATCH /api/workspaces/:id/changes/:name/status` — met à jour le champ `kanban_status` dans `.openspec.yaml` en préservant les autres champs

## 5. Backend — Specs API

- [x] 5.1 Créer `backend/internal/openspec/spec.go` — listing de `openspec/specs/`, lecture du fichier `spec.md` de chaque spec
- [x] 5.2 Créer `backend/internal/api/handlers/specs.go` — `GET /api/workspaces/:id/specs` (liste), `GET /api/workspaces/:id/specs/:name` (contenu Markdown brut)

## 6. Backend — Session d'exploration (WebSocket + subprocess)

- [x] 6.1 Créer `backend/internal/session/manager.go` — registre en mémoire des sessions actives (clé : `workspaceId+changeName`), timer d'inactivité 30 min, `Start()`, `Stop()`, `Get()`
- [x] 6.2 Créer `backend/internal/session/subprocess.go` — spawn du subprocess `claude` avec les flags requis (`--print --input-format stream-json --output-format stream-json --include-partial-messages --append-system-prompt "..." --cwd <path>`), goroutines de lecture stdout et écriture stdin
- [x] 6.3 Créer `backend/internal/api/handlers/explore.go` — upgrade WebSocket sur `GET /api/workspaces/:id/changes/:name/explore`, proxy bidirectionnel messages frontend ↔ subprocess
- [x] 6.4 Gérer la fermeture propre : réception close WebSocket → fermeture stdin subprocess → attente exit

## 7. Backend — Archive

- [x] 7.1 Créer `backend/internal/api/handlers/archive.go` — `POST /api/workspaces/:id/changes/:name/archive`, exécute `openspec archive <name> --yes` avec `cwd` = workspace root, capture stdout+stderr combinés
- [x] 7.2 Retourner `200` avec le texte de sortie en succès, `422` avec le message d'erreur si la commande échoue (tasks non complètes ou autre)

## 8. Frontend — Foundation

- [x] 8.1 Installer les dépendances frontend : `@tanstack/react-query`, `react-router-dom`, `axios`, `react-markdown`
- [x] 8.2 Créer `frontend/src/lib/api.ts` — instance Axios avec `baseURL` vers le backend Go (configurable via env var `VITE_API_URL`)
- [x] 8.3 Créer `frontend/src/App.tsx` — `QueryClientProvider`, `BrowserRouter`, routes : `/` (Kanban), `/specs` (Specs view)
- [x] 8.4 Créer `frontend/src/components/Layout.tsx` — sidebar workspace + navigation entre Kanban et Specs

## 9. Frontend — Workspace Management

- [x] 9.1 Créer `frontend/src/components/WorkspaceSidebar.tsx` — liste des workspaces, workspace actif surligné, boutons ajouter/supprimer
- [x] 9.2 Créer `frontend/src/pages/WorkspaceSetup.tsx` — écran affiché quand aucun workspace, bouton "Ajouter un projet"
- [x] 9.3 Créer `frontend/src/hooks/useWorkspaces.ts` — `useQuery` liste workspaces, mutations `addWorkspace(path)` et `removeWorkspace(id)`
- [x] 9.4 Implémenter la sélection de répertoire via `<input type="file" webkitdirectory>` et extraction du chemin (ou champ texte de saisie directe)

## 10. Frontend — Kanban Board

- [x] 10.1 Créer `frontend/src/pages/KanbanPage.tsx` — layout 4 colonnes flexbox
- [x] 10.2 Créer `frontend/src/components/KanbanColumn.tsx` — colonne avec titre, compteur de cartes, zone de drop
- [x] 10.3 Créer `frontend/src/components/ChangeCard.tsx` — nom du change, barre de progression tasks (N/M), actions contextuelles selon la colonne
- [x] 10.4 Implémenter le drag & drop entre colonnes via HTML5 DnD API (`draggable`, `onDragOver`, `onDrop`) et déclencher `PATCH .../status` à la dépose
- [x] 10.5 Créer `frontend/src/hooks/useChanges.ts` — `useQuery` avec `refetchInterval: 5000`, mutation `updateStatus(changeName, status)`
- [x] 10.6 Ajouter le bouton "Passer en To Do" dans les cartes de la colonne To Explore

## 11. Frontend — Chat d'exploration

- [x] 11.1 Créer `frontend/src/components/ExplorePanel.tsx` — panneau latéral avec fil de messages (assistant + utilisateur) et champ de saisie
- [x] 11.2 Créer `frontend/src/hooks/useExploreSession.ts` — gestion du WebSocket (`ws://backend/api/.../explore`), état messages, envoi utilisateur, réception chunks streamés
- [x] 11.3 Afficher les chunks partiels en temps réel dans le fil (append au dernier message assistant en cours)
- [x] 11.4 Gérer l'état "session expirée" : afficher "Session expirée" + bouton "Relancer" qui ferme/réouvre la connexion WebSocket

## 12. Frontend — Vue Specs

- [x] 12.1 Créer `frontend/src/pages/SpecsPage.tsx` — liste des specs à gauche, panneau de détail Markdown à droite
- [x] 12.2 Créer `frontend/src/hooks/useSpecs.ts` — `useQuery` liste et contenu d'une spec
- [x] 12.3 Afficher le contenu Markdown avec `react-markdown`

## 13. Frontend — Archive UI

- [x] 13.1 Ajouter le bouton "Archiver" dans `ChangeCard` pour les cartes de la colonne Done (spinner pendant l'opération, bouton désactivé)
- [x] 13.2 Créer `frontend/src/hooks/useArchive.ts` — mutation `POST .../archive`, gestion états loading/success/error
- [x] 13.3 Afficher le message d'erreur retourné par le backend (tasks non finalisées) directement dans la carte en rouge
