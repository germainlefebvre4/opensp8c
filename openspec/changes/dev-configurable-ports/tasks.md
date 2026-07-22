## 1. Mise à jour du Makefile

- [x] 1.1 Définir les variables `BACKEND_PORT` (par défaut `8080`) et `FRONTEND_PORT` (par défaut `5173`) au début du Makefile de la racine
- [x] 1.2 Modifier la cible `dev-backend` pour passer la variable d'environnement `PORT` positionnée à `BACKEND_PORT`
- [x] 1.3 Modifier la cible `dev-frontend` pour passer `VITE_API_URL` pointant sur le port résolu et lancer Vite avec l'option `--port $(FRONTEND_PORT)`

## 2. Validation et Tests

- [x] 2.1 Lancer l'application avec la configuration par défaut (`make dev`) et vérifier que les ports d'écoute résolus sont `8080` (backend) et `5173` (frontend)
- [x] 2.2 Lancer l'application avec des ports personnalisés (`BACKEND_PORT=4001 FRONTEND_PORT=3001 make dev`) et vérifier que les ports d'écoute résolus sont `4001` (backend) et `3001` (frontend)
- [x] 2.3 Vérifier que le frontend communique avec succès avec le backend sur le port personnalisé et qu'aucune erreur de CORS ou de connexion ne survient
