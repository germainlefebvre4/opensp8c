# Spec: Server Listen Configuration

## Purpose

Define how the server resolves its listen address and port from CLI flags and environment variables.

## Requirements

### Requirement: Configuration du port via argument CLI
Le serveur SHALL accepter un flag `--port` qui définit le port d'écoute. Ce flag a priorité sur la variable d'environnement `PORT`. Si ni le flag ni la variable ne sont fournis, le port par défaut est `8080`.

#### Scenario: Flag --port fourni
- **WHEN** le serveur est lancé avec `--port 3001`
- **THEN** le serveur écoute sur le port `3001`

#### Scenario: FLAG absent, env var PORT définie
- **WHEN** le serveur est lancé sans `--port` et `PORT=9000` est défini
- **THEN** le serveur écoute sur le port `9000`

#### Scenario: Ni flag ni env var
- **WHEN** le serveur est lancé sans `--port` et sans `PORT`
- **THEN** le serveur écoute sur le port `8080`

#### Scenario: Flag --port a priorité sur env var PORT
- **WHEN** le serveur est lancé avec `--port 3001` et `PORT=9000` est défini
- **THEN** le serveur écoute sur le port `3001`

### Requirement: Configuration de l'adresse d'écoute via argument CLI
Le serveur SHALL accepter un flag `--host` qui définit l'adresse d'écoute (bind address). Ce flag a priorité sur la variable d'environnement `HOST`. Si ni le flag ni la variable ne sont fournis, l'adresse par défaut est `0.0.0.0`.

#### Scenario: Flag --host fourni
- **WHEN** le serveur est lancé avec `--host 127.0.0.1`
- **THEN** le serveur écoute uniquement sur l'interface loopback

#### Scenario: Flag absent, env var HOST définie
- **WHEN** le serveur est lancé sans `--host` et `HOST=127.0.0.1` est défini
- **THEN** le serveur écoute sur `127.0.0.1`

#### Scenario: Ni flag ni env var HOST
- **WHEN** le serveur est lancé sans `--host` et sans `HOST`
- **THEN** le serveur écoute sur toutes les interfaces (`0.0.0.0`)

### Requirement: Log de démarrage complet
Le serveur SHALL afficher l'adresse complète (`host:port`) dans le log de démarrage.

#### Scenario: Log au démarrage
- **WHEN** le serveur démarre avec host `127.0.0.1` et port `3001`
- **THEN** le log affiche `Server listening on 127.0.0.1:3001`
