## Context

OpenSp8c affiche les changes OpenSpec via un Kanban. Les changes stockent leur état dans `.openspec.yaml` et `tasks.md`. Aucune métadonnée sémantique n'existe — le seul filtre disponible est le nom du change. L'objectif est d'enrichir chaque change avec des tags auto-dérivés (type applicatif, complexité, composants touchés) sans action utilisateur, permettant filtrage sémantique et vue timeline.

Backend : Go (chi, fsnotify, WebSocket), session subprocess existante dans `internal/session/`. Frontend : React 19 + TypeScript + TanStack Query. La CLI `claude` est déjà détectée via `internal/agents/`.

## Goals / Non-Goals

**Goals:**
- Tags auto-dérivés pour tous les changes (actifs + archivés), sans action utilisateur
- Trois dimensions : `type` (frontend/backend/batch/fullstack), `complexity` (1-5), `components` (liste kebab-case)
- Vocabulaire émergent : extrait dynamiquement de l'ensemble des YAMLs du workspace, passé en contexte au LLM pour normaliser
- Batch rétroactif au démarrage (ordre chronologique pour construire le vocabulaire progressivement)
- Trigger automatique à l'archivage d'un change
- UI : type + complexity sur les cartes Kanban, tags complets dans le DetailPanel, filtrage par tags dans la search bar
- Nouvelle vue `/timeline` chronologique et filtrable par tags

**Non-Goals:**
- Tags manuels via l'UI (le flag `_auto` prépare cette extension, hors scope initial)
- Déduplication interactive du vocabulaire (futur : `openspec tags dedup`)
- Synchronisation des tags entre workspaces

## Decisions

### 1. Heuristique pour `type`, LLM pour `complexity` et `components`

`type` est déductible mécaniquement des chemins de fichiers dans `tasks.md` :
- Lignes contenant `frontend/` → `frontend`
- Lignes contenant `backend/` → `backend`
- Lignes contenant `scripts/`, `batch/`, `cmd/` → `batch`
- Présence des deux premiers → `fullstack`
- Aucun chemin → fallback sur le préfixe du nom du change (`fix-`, `feat-`, etc.) ou champ vide

LLM pour `complexity` et `components` : le langage naturel de `proposal.md` + `design.md` porte l'information que les chemins de fichiers seuls ne contiennent pas. Un score de complexité ou un composant comme "explore-panel" ne peut pas être déduit mécaniquement.

Alternatives rejetées : LLM pour tout (latence + coût sans gain sur `type`) ; heuristique pour tout (insuffisant pour les composants en langage naturel).

### 2. Invocation LLM via `claude --print`

La CLI Claude supporte le mode non-interactif : `claude --print "<prompt>"`. Cela réutilise la détection d'agent existante (`internal/agents/`) et évite une nouvelle dépendance (SDK Anthropic, gestion de clé API). Le mode `--print` est synchrone et retourne la réponse texte ou JSON.

Prompt structuré : le tagger envoie `proposal.md` + `design.md` + le vocabulaire courant des composants, et demande un JSON `{ complexity: int, components: string[] }`.

Alternative rejetée : appel direct à l'API Anthropic (nouvelle dépendance, distribution d'une clé API requise).

### 3. Vocabulaire extrait dynamiquement, jamais stocké séparément

À chaque déclenchement du tagger pour un change donné, le service scanne tous les `.openspec.yaml` du workspace (actifs + archivés) et collecte l'union de tous les champs `tags.components`. Cette liste est passée en contexte au LLM : "utilise ces termes existants en priorité, crée un nouveau slug kebab-case uniquement si aucun ne correspond". Aucun fichier de vocabulaire supplémentaire — toujours cohérent avec l'état réel des YAMLs.

### 4. Déclenchement : batch au démarrage + trigger à l'archivage + endpoint manuel

- **Démarrage** : goroutine background, tague tous les changes sans section `tags`, dans l'ordre chronologique (pour que le vocabulaire s'enrichisse progressivement). Silencieux, non-bloquant.
- **Archivage** : trigger automatique post-archivage si le change n'a pas de tags.
- **Manuel** : `POST /api/workspaces/{id}/changes/{name}/retag` — force le re-tagging même si `_auto: false`.

### 5. Immutabilité des tags manuels

Le flag `_auto: true` dans le YAML indique une dérivation automatique. Si un utilisateur édite manuellement le YAML (`_auto: false`), le tagger ne re-écrase pas lors du batch de démarrage. L'endpoint `/retag` ignore ce flag (re-tagging explicite).

## Risks / Trade-offs

- **Latence du batch de démarrage** (26+ changes × appel LLM) → background goroutine non-bloquante. L'UI affiche les tags si présents, les cards restent fonctionnelles sans tags.
- **`claude --print` indisponible** (CLI non installée) → le tagger skip silencieusement, les tags restent absents. Dégradation gracieuse, aucun impact fonctionnel sur le reste de l'app.
- **Drift du vocabulaire** sur très longue période → contrôlé par le contexte LLM. Un dédup manuel ponctuel suffira si nécessaire.
- **Coût LLM** → une seule passe par change, pas de re-tagging automatique périodique. Le batch de 26 changes représente ~26 appels, acceptable au démarrage.
