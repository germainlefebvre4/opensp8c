## Context

Actuellement, les applications frontend et backend démarrent sur des ports d'écoute fixes en développement (`8080` pour le backend Go, `5173` pour le frontend Vite via React). Si un développeur a déjà un autre processus s'exécutant sur l'un de ces ports (par exemple, un autre projet d'assistant ou de backend), le démarrage échoue ou entre en conflit.
Le backend Go supporte déjà une variable d'environnement `PORT` ou l'option `--port`. Le frontend de son côté utilise `import.meta.env.VITE_API_URL` pour cibler l'API et est démarré par Vite.

## Goals / Non-Goals

**Goals:**
- Rendre les ports du frontend et du backend configurables via des variables d'environnement (`BACKEND_PORT` et `FRONTEND_PORT`) au niveau du `Makefile`.
- Permettre au frontend de cibler automatiquement le bon port du backend sans action manuelle supplémentaire du développeur.
- Préserver le comportement par défaut (ports `8080` et `5173`) lorsqu'aucune variable d'environnement n'est spécifiée.

**Non-Goals:**
- Modifier la configuration de production (Docker, docker-compose).
- Introduire des dépendances de chargement de fichier `.env` côté Go.

## Decisions

### Décision 1 : Utilisation des variables par défaut dans le Makefile (`?=`)
Nous définissons `BACKEND_PORT ?= 8080` et `FRONTEND_PORT ?= 5173` dans le Makefile racine.
- **Raisonnement** : C'est le standard pour définir des valeurs de repli (fallback) dans un Makefile tout en permettant aux variables d'environnement d'écraser ces valeurs de manière transparente.
- **Alternative considérée** : Parser un fichier `.env` au niveau du Makefile. Rejeté car cela ajoute de la complexité inutile alors que les variables d'environnement du shell et du Makefile s'en sortent nativement.

### Décision 2 : Transmission du port via la commande Vite CLI
Le script `dev-frontend` appelle Vite avec `--port $(FRONTEND_PORT)`.
- **Raisonnement** : C'est la façon la plus simple et propre de surcharger le port de démarrage de Vite sans modifier le fichier `vite.config.ts`.
- **Alternative considérée** : Modifier `vite.config.ts` pour lire le port depuis les variables d'environnement. Rejeté car la ligne de commande est plus directe et évite du code supplémentaire dans la configuration Vite.

### Décision 3 : Liaison automatique de l'URL API via `VITE_API_URL`
Le script `dev-frontend` injecte la variable d'environnement `VITE_API_URL=http://localhost:$(BACKEND_PORT)`.
- **Raisonnement** : Vite expose automatiquement au client toutes les variables d'environnement préfixées par `VITE_` qui lui sont fournies lors du démarrage. Le code frontend actuel consomme déjà `import.meta.env.VITE_API_URL` s'il est présent. En l'injectant au démarrage du frontend, la connexion se fait automatiquement sur le bon port sans intervention manuelle du développeur.

## Risks / Trade-offs

- **[Risk]** Incompatibilité avec d'anciennes versions de make.
  - **Mitigation** : La syntaxe `?=` est supportée par GNU Make depuis des décennies et est standard sur Linux et macOS.
- **[Risk]** Confusion des variables d'environnement (ex: configurer `PORT` vs `BACKEND_PORT`).
  - **Mitigation** : Dans `dev-backend`, nous faisons correspondre `PORT=$(BACKEND_PORT)`. Ainsi, même si le développeur utilise `PORT`, l'intégration reste cohérente, mais `BACKEND_PORT` est privilégié dans la documentation/Makefile.
