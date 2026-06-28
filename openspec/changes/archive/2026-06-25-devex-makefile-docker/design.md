## Context

Le projet a un backend Go (`cmd/server`, port `8080`) et un frontend React/Vite. Le Makefile actuel expose 4 cibles avec des noms non standards (`backend`, `frontend` au lieu de `dev-backend`, `dev-frontend`). Il n'y a pas de Dockerfile ni de docker-compose. Le router Go n'expose que `/api/*` — il ne sert pas de fichiers statiques.

Les binaires `claude` et `openspec` sont des dépendances hôte appelées en subprocess : ils doivent être disponibles sur la machine qui exécute le backend.

## Goals / Non-Goals

**Goals:**
- Makefile avec 8 cibles standardisées couvrant dev, build, et docker
- Dockerfile multi-stage produisant une image alpine slim auto-suffisante
- Le backend Go sert le frontend buildé (fichiers statiques) via `//go:embed`
- docker-compose pour usage local avec volumes hôte
- Registry configurable via variable Makefile

**Non-Goals:**
- Installation automatique de `claude`/`openspec` dans l'image Docker
- Hot-reload dans le conteneur Docker
- CI/CD pipeline (hors scope)
- Multi-architecture image build (linux/amd64 uniquement pour l'instant)

## Decisions

### 1. Frontend servi par le backend Go via `//go:embed`

**Décision** : Le router Go intègre le frontend buildé via `//go:embed dist` et expose une route catch-all qui sert `index.html` pour toutes les routes non-API.

**Alternatives considérées** :
- *nginx dans le conteneur* : nécessite un docker-compose même pour un usage simple, deux processus à superviser.
- *Deux conteneurs séparés* : complexité opérationnelle, CORS à gérer en prod.

**Rationale** : Conteneur unique, un seul port (8080), binary self-contained. Cohérent avec l'architecture "binaire unique" du design général.

---

### 2. Dockerfile multi-stage : node → go → alpine

**Décision** :
```
Stage 1 (node:22-alpine)  : npm ci + vite build → dist/
Stage 2 (golang:alpine)   : go mod download + go build (embed dist/) → binaire
Stage 3 (alpine:3.20)     : binaire seul + ca-certificates
```

**Rationale** : Image finale ~20-30MB. Node et Go toolchain absents de l'image finale.

---

### 3. `claude` et `openspec` montés depuis l'hôte

**Décision** : Les binaires ne sont pas installés dans l'image. Le docker-compose monte `/usr/local/bin/claude` et `/usr/local/bin/openspec` en lecture seule depuis l'hôte.

**Alternatives considérées** :
- *Installer dans l'image* : versions figées, rebuild nécessaire à chaque mise à jour des CLI.

**Rationale** : L'utilisateur gère ses propres versions de `claude` et `openspec`. Le montage garantit la cohérence hôte/conteneur.

---

### 4. Variables Makefile pour le registry Docker

**Décision** :
```makefile
REGISTRY  ?= docker.io/germainlefebvre4
IMAGE     ?= opensp8c
TAG       ?= latest
```

Surchargeable à la volée : `make docker-build-push TAG=v0.1.0`.

---

### 5. `build-frontend` précède `build-backend` (dépendance embed)

**Décision** : `build` s'exécute séquentiellement : `build-frontend` d'abord, puis `build-backend`. Le Go build nécessite que `frontend/dist/` existe pour le `//go:embed`.

## Risks / Trade-offs

- **`//go:embed` chemin** : Le directive embed dans Go doit être dans le même package que le fichier source. Il faudra un fichier dédié (ex: `backend/internal/api/static.go`) avec `//go:embed ../../../frontend/dist`. Les chemins relatifs doivent être valides au moment du build → `build-frontend` DOIT précéder `build-backend`.
- **docker-compose et chemins absolus** : Les chemins de workspaces dans `config.yaml` sont absolus sur la machine hôte. En Docker, ils doivent être montés identiquement ou le `config.yaml` doit être adapté. → Le docker-compose monte `/home` ou le répertoire de projets de l'utilisateur.
- **Port en dev** : Vite tourne sur `:5173`, le backend sur `:8080`. Les appels API depuis le frontend en dev utilisent une URL absolue ou un proxy Vite. Le CORS `*` déjà en place évite le besoin d'un proxy, mais une config `server.proxy` dans `vite.config.ts` serait plus propre pour les WebSockets.
