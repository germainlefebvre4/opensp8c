## Why

L'expérience développeur est fragmentée : le Makefile actuel n'expose que 4 cibles non conventionnelles, il n'y a ni Dockerfile ni docker-compose, rendant le build reproductible et la distribution impossibles.

## What Changes

- Remplacement du Makefile par une version complète avec 8 cibles standardisées (`dev`, `dev-backend`, `dev-frontend`, `build`, `build-backend`, `build-frontend`, `docker-build`, `docker-build-push`)
- Ajout d'un `Dockerfile` multi-stage (node → go build avec embed frontend → alpine slim)
- Ajout d'un handler de fichiers statiques dans le router Go pour servir le frontend buildé
- Ajout d'un `docker-compose.yml` pour l'usage local avec volumes hôte
- Variable `REGISTRY ?= docker.io/germainlefebvre4` pour le push d'image

## Capabilities

### New Capabilities

- `developer-tooling`: Makefile complet, Dockerfile multi-stage, docker-compose, et serving statique Go pour l'expérience développeur et la distribution

### Modified Capabilities

## Impact

- `Makefile` : réécriture complète
- `backend/internal/api/router.go` : ajout du handler de fichiers statiques (`//go:embed` ou `fs.FS`)
- Nouveau fichier `Dockerfile` à la racine
- Nouveau fichier `docker-compose.yml` à la racine
- Nouveau répertoire `bin/` pour les binaires buildés (gitignore)
