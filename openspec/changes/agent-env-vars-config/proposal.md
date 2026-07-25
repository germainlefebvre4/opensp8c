## Why

Actuellement, les agents CLI (particulièrement le CLI Gemini via Gemini Enterprise) dépendent de variables d'environnement telles que `GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, ou `GEMINI_SANDBOX` pour s'authentifier et s'exécuter correctement. Permettre à l'utilisateur de configurer dynamiquement ces variables à chaud depuis l'interface Web évite de devoir relancer tout le serveur backend avec de nouvelles variables, améliorant ainsi considérablement l'expérience utilisateur et la flexibilité d'utilisation.

## What Changes

- Ajout d'un menu de configuration des variables d'environnement de l'agent à côté du sélecteur d'agent dans la barre latérale.
- Persistance et chargement de ces variables personnalisées dans `preferences.json` sous un nouveau champ `env`.
- Injection dynamique ("à chaud") de ces variables d'environnement dans les processus fils de chaque agent CLI lancés par le backend (explore session et fast-forward runs), tout en préservant les variables d'environnement existantes du système hôte.

## Capabilities

### New Capabilities

<!-- Aucun -->

### Modified Capabilities

- `agent-selection`: Mettre à jour l'exigence de persistance des préférences pour inclure la mémorisation et la récupération d'un dictionnaire de variables d'environnement personnalisées, et modifier le comportement de démarrage du subprocess pour combiner et injecter ces variables d'environnement au démarrage des agents CLI.

## Impact

- `backend/internal/preferences/preferences.go` : Ajout du champ `Env` à la structure `Preferences` et création d'une méthode thread-safe pour le mettre à jour.
- `backend/internal/session/subprocess.go` : Mise à jour de `StartSubprocess` pour accepter et injecter les variables d'environnement combinées.
- `backend/internal/api/handlers/preferences.go` : Exposition du dictionnaire `env` dans les endpoints `GET /api/preferences` et `PATCH /api/preferences`.
- `frontend/src/components/AgentSelector.tsx` : Ajout d'un bouton d'engrenage ouvrant une modale de configuration.
- `frontend/src/components/AgentSettingsModal.tsx` : Création du composant modal pour éditer les variables d'environnement (recommandées et personnalisées).
