## Why

Le serveur écoute actuellement sur un port fixé via la variable d'environnement `PORT`, sans possibilité de configurer l'adresse d'écoute. L'ajout d'arguments CLI `--host` et `--port` permet de contrôler dynamiquement le bind du serveur sans modifier l'environnement, ce qui facilite les déploiements multi-instances et le développement local.

## What Changes

- Ajout du flag `--port` : port d'écoute du serveur (priorité sur `PORT` env var, défaut `8080`)
- Ajout du flag `--host` : adresse d'écoute du serveur (priorité sur `HOST` env var, défaut `0.0.0.0`)
- Mise à jour du log de démarrage pour afficher l'adresse complète (`host:port`)

## Capabilities

### New Capabilities

- `server-listen-config`: Configuration de l'adresse et du port d'écoute du serveur via arguments CLI

### Modified Capabilities

<!-- Aucune spec existante impactée -->

## Impact

- `backend/cmd/server/main.go` : ajout du parsing des flags CLI avec le package `flag` de la stdlib Go
- Chaîne de priorité : `--host`/`--port` > `HOST`/`PORT` env vars > défauts `0.0.0.0:8080`
- Pas de dépendances externes ajoutées (stdlib uniquement)
