## MODIFIED Requirements

### Requirement: Makefile dev targets
Le Makefile SHALL exposer des cibles de développement permettant de lancer backend et frontend indépendamment ou ensemble. Les ports d'écoute SHALL être configurables via les variables `BACKEND_PORT` (par défaut `8080`) et `FRONTEND_PORT` (par défaut `5173`). De plus, le frontend SHALL être automatiquement configuré via `VITE_API_URL` pour cibler l'adresse et le port d'écoute résolus du backend.

#### Scenario: Lancer le dev complet par défaut
- **WHEN** le développeur exécute `make dev` sans configurer les variables de port
- **THEN** le backend Go démarre sur le port `8080` et le frontend Vite sur le port `5173` en parallèle, et le frontend communique avec l'API sur `http://localhost:8080`

#### Scenario: Lancer le dev complet avec des ports personnalisés
- **WHEN** le développeur exécute `BACKEND_PORT=4000 FRONTEND_PORT=3000 make dev`
- **THEN** le backend Go démarre sur le port `4000`, le frontend Vite démarre sur le port `3000`, et le frontend communique avec l'API sur `http://localhost:4000`

#### Scenario: Lancer uniquement le backend avec port par défaut
- **WHEN** le développeur exécute `make dev-backend` sans variable de port
- **THEN** le backend Go démarre sur le port `8080`

#### Scenario: Lancer uniquement le backend avec port personnalisé
- **WHEN** le développeur exécute `BACKEND_PORT=4000 make dev-backend`
- **THEN** le backend Go démarre sur le port `4000`

#### Scenario: Lancer uniquement le frontend avec port par défaut
- **WHEN** le développeur exécute `make dev-frontend` sans variable de port
- **THEN** le serveur Vite démarre sur le port `5173` et pointe sur l'API `http://localhost:8080`

#### Scenario: Lancer uniquement le frontend avec ports personnalisés
- **WHEN** le développeur exécute `BACKEND_PORT=4000 FRONTEND_PORT=3000 make dev-frontend`
- **THEN** le serveur Vite démarre sur le port `3000` et pointe sur l'API `http://localhost:4000`
