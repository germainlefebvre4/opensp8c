## Context

Le `WorkspaceSidebar` reçoit actuellement `Workspace[]` (id, name, path) — sans aucune donnée de changes. Les compteurs Kanban sont disponibles via `GET /api/workspaces/:id/changes`, mais uniquement pour le workspace actif, pollé toutes les 5s par `useChanges`. Il n'existe pas de vue agrégée des counts sur tous les workspaces.

## Goals / Non-Goals

**Goals:**
- Afficher des badges de compteurs Kanban par statut dans le sidebar, pour tous les workspaces.
- Maintenir les badges à jour sans polling agressif.
- Conserver l'affordance existante du sidebar (survol, suppression, sélection).

**Non-Goals:**
- Afficher le statut `done` dans les badges (indicatif, pas actionnable dans la sidebar).
- Modifier le polling du kanban board lui-même (`useChanges`).
- Ajouter une barre de progression ou une vue détaillée au sidebar.

## Decisions

### 1. Agrégation backend vs frontend

**Décision** : Étendre `GET /api/workspaces` pour inclure `task_counts` par workspace.

**Alternatives considérées** :
- *N+1 frontend* : `useChanges` par workspace dans le sidebar → N requêtes simultanées avec polling x5s sur chacune. Trop lourd pour 10+ projets.
- *Endpoint dédié `/api/workspaces/stats`* : requête séparée, maintient la séparation des concerns, mais ajoute un round-trip supplémentaire sans gain réel ici.

**Rationale** : L'agrégation est légère (scan filesystem local, pas de DB), et l'endpoint workspace est déjà le point d'entrée de l'UI. Un champ additif ne casse pas les clients existants.

### 2. Stratégie de refresh

**Décision** : `refetchInterval: 15000` sur `useWorkspaces`.

**Alternatives considérées** :
- *5s (sync avec kanban)* : badges parfaitement frais, mais double les requêtes sans valeur ajoutée perceptible pour un indicateur secondaire.
- *Invalidation croisée* : quand `useChanges` reçoit des données → `queryClient.invalidateQueries(['workspaces'])`. Élégant mais couplage fort entre hooks non liés.

**Rationale** : Les badges sidebar sont indicatifs, pas temps-réel critiques. 15s est imperceptible pour un count de navigation.

### 3. Layout des badges

**Décision** : Inline sur une ligne — nom + badges à droite, largeur `w-64` (vs `w-56`).

**Alternative** : Deux lignes (nom / badges) — meilleure lisibilité du nom complet, mais sidebar plus chargée verticalement et items moins denses.

**Rationale** : La sidebar est un menu de navigation. Le nom prime ; les badges sont secondaires. Avec `w-64`, 2-3 badges typiques laissent suffisamment d'espace au nom tronqué. Les 4 statuts actifs simultanément sont rares en pratique.

### 4. Couleurs des badges

Réutilisation des couleurs existantes de `KanbanColumn.tsx` :
- `to-explore` → violet (`bg-violet-400`)
- `todo` → slate (`bg-slate-400`)
- `in-progress` → amber (`bg-amber-400`)
- `done` → non affiché dans la sidebar

## Risks / Trade-offs

- **Scan filesystem sur chaque listing** : `GET /api/workspaces` appelle désormais `openspec.ListChanges` pour chaque workspace. Coût acceptable sur un outil local (fichiers locaux, peu de workspaces), mais pourrait être lent sur un réseau monté (NFS, WSL2). → Mitigation : le polling est à 15s, pas en temps réel.

- **Désynchronisation badges/kanban** : les badges peuvent avoir jusqu'à 15s de retard vs le kanban board (5s). → Acceptable : le kanban reste la source de vérité authoritative ; les badges sont une indication.
