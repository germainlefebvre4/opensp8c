## 1. Makefile

- [x] 1.1 Réécrire le Makefile avec les 8 cibles (`dev`, `dev-backend`, `dev-frontend`, `build`, `build-backend`, `build-frontend`, `docker-build`, `docker-build-push`)
- [x] 1.2 Ajouter les variables `REGISTRY`, `IMAGE`, `TAG` avec valeurs par défaut
- [x] 1.3 S'assurer que `build` exécute `build-frontend` avant `build-backend` (dépendance embed)
- [x] 1.4 Mettre à jour la directive `.PHONY` pour toutes les nouvelles cibles

## 2. Static file serving (backend Go)

- [x] 2.1 Créer `backend/internal/api/static.go` avec la directive `//go:embed` pointant sur `frontend/dist`
- [x] 2.2 Ajouter dans le router Go un handler catch-all servant `index.html` pour les routes non-API
- [x] 2.3 Ajouter un handler de fichiers statiques pour `/assets/*` et autres ressources frontend
- [x] 2.4 S'assurer que la route catch-all ne masque pas les routes `/api/*`

## 3. Dockerfile

- [x] 3.1 Créer le `Dockerfile` multi-stage à la racine : stage `frontend-builder` (node:22-alpine)
- [x] 3.2 Ajouter le stage `backend-builder` (golang:alpine) qui copie `dist/` depuis le stage précédent
- [x] 3.3 Ajouter le stage final (alpine:3.20) avec uniquement le binaire et `ca-certificates`
- [x] 3.4 Exposer le port 8080 et définir l'entrypoint

## 4. docker-compose

- [x] 4.1 Créer `docker-compose.yml` à la racine avec le service `app`
- [x] 4.2 Monter `./config.yaml:/config.yaml:ro`
- [x] 4.3 Monter `/usr/local/bin/claude:/usr/local/bin/claude:ro` et `openspec` de même
- [x] 4.4 Monter le répertoire home utilisateur (ou répertoire de projets) pour l'accès aux workspaces
- [x] 4.5 Configurer les variables d'environnement `PORT` et `CONFIG_PATH`
