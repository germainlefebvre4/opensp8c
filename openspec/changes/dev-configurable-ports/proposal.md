## Why

Permettre aux développeurs de personnaliser les ports d'écoute du backend Go et du frontend Vite en mode développement, afin d'éviter tout conflit de port avec d'autres applications s'exécutant simultanément sur la même machine.

## What Changes

- Introduction de variables de ports personnalisables au niveau du Makefile (`BACKEND_PORT` et `FRONTEND_PORT`) avec des valeurs par défaut pour préserver la rétrocompatibilité.
- Configuration dynamique du backend Go avec le port défini par `BACKEND_PORT`.
- Configuration dynamique de l'application frontend Vite avec le port défini par `FRONTEND_PORT`.
- Liaison automatique du frontend vers l'API du backend via l'injection de la variable d'environnement `VITE_API_URL` correspondant à l'adresse de `BACKEND_PORT`.

## Capabilities

### New Capabilities
<!-- None -->

### Modified Capabilities
- `developer-tooling`: Ajout du support de configuration des ports d'écoute backend et frontend via les variables d'environnement `BACKEND_PORT` et `FRONTEND_PORT` dans les cibles du Makefile de développement, avec transmission automatique de `VITE_API_URL`.

## Impact

- **Root Makefile** : Mise à jour des cibles de développement (`dev`, `dev-backend`, `dev-frontend`).
- Aucune rupture de compatibilité (breaking change) : les ports par défaut restent `8080` (backend) et `5173` (frontend).
