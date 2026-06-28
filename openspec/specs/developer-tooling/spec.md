# Spec: Developer Tooling

## Purpose

Defines the developer tooling requirements for the opensp8c project, covering Makefile targets for development, build and Docker workflows, Dockerfile structure, static file embedding, and local docker-compose setup.

## Requirements

### Requirement: Makefile dev targets
Le Makefile SHALL exposer des cibles de développement permettant de lancer backend et frontend indépendamment ou ensemble.

#### Scenario: Lancer le dev complet
- **WHEN** le développeur exécute `make dev`
- **THEN** le backend Go et le frontend Vite démarrent en parallèle

#### Scenario: Lancer uniquement le backend
- **WHEN** le développeur exécute `make dev-backend`
- **THEN** le backend Go démarre via `go run ./cmd/server` depuis le répertoire `backend/`

#### Scenario: Lancer uniquement le frontend
- **WHEN** le développeur exécute `make dev-frontend`
- **THEN** le serveur Vite démarre via `npm run dev` depuis le répertoire `frontend/`

---

### Requirement: Makefile build targets
Le Makefile SHALL exposer des cibles de build produisant des artefacts prêts pour la production.

#### Scenario: Build complet séquentiel
- **WHEN** le développeur exécute `make build`
- **THEN** le frontend est buildé en premier, puis le backend Go est compilé avec le frontend embarqué

#### Scenario: Build backend seul
- **WHEN** le développeur exécute `make build-backend`
- **THEN** un binaire `bin/opensp8c` est produit à la racine du projet

#### Scenario: Build frontend seul
- **WHEN** le développeur exécute `make build-frontend`
- **THEN** les fichiers statiques sont produits dans `frontend/dist/`

---

### Requirement: Makefile docker targets
Le Makefile SHALL exposer des cibles Docker avec un registry configurable.

#### Scenario: Build de l'image Docker
- **WHEN** le développeur exécute `make docker-build`
- **THEN** une image Docker est construite et taguée `$(REGISTRY)/$(IMAGE):$(TAG)`

#### Scenario: Build et push de l'image Docker
- **WHEN** le développeur exécute `make docker-build-push`
- **THEN** l'image est construite puis poussée vers le registry

#### Scenario: Override du tag à la volée
- **WHEN** le développeur exécute `make docker-build-push TAG=v0.1.0`
- **THEN** l'image est taguée et poussée avec le tag `v0.1.0`

---

### Requirement: Dockerfile multi-stage
Le Dockerfile SHALL produire une image alpine slim via un build multi-stage.

#### Scenario: Build de l'image
- **WHEN** `docker build` est exécuté
- **THEN** l'image finale contient uniquement le binaire Go et les certificats CA, sans Node.js ni Go toolchain

#### Scenario: Frontend embarqué dans le binaire
- **WHEN** l'image est lancée et une requête non-API est reçue
- **THEN** le backend sert les fichiers statiques du frontend buildé

---

### Requirement: Backend sert les fichiers statiques via embed
Le backend Go SHALL servir le frontend buildé via `//go:embed` sur toutes les routes non-API.

#### Scenario: Route SPA catch-all
- **WHEN** une requête arrive sur une route qui n'est pas sous `/api/`
- **THEN** le backend répond avec `index.html` du frontend embarqué

#### Scenario: Fichiers statiques assets
- **WHEN** une requête arrive pour `/assets/main.js` ou `/assets/main.css`
- **THEN** le backend répond avec le fichier statique correspondant depuis le embed

---

### Requirement: docker-compose pour usage local
Le `docker-compose.yml` SHALL permettre de lancer l'application avec les dépendances hôte montées.

#### Scenario: Lancement via docker-compose
- **WHEN** le développeur exécute `docker compose up`
- **THEN** le conteneur démarre avec `config.yaml` et les binaires `claude`/`openspec` montés depuis l'hôte

#### Scenario: Accès aux workspaces locaux
- **WHEN** le conteneur est lancé via docker-compose
- **THEN** les répertoires de projets définis dans `config.yaml` sont accessibles depuis le conteneur via volumes
