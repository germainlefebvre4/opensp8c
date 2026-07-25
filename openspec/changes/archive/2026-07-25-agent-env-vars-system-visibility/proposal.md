## Why

Actuellement, l'interface de configuration des variables d'environnement de l'agent affiche des champs vides même si les variables recommandées (comme `GOOGLE_CLOUD_PROJECT`) sont déjà définies au niveau du système hôte (shell parent du serveur). Cela s'avère extrêmement confusant pour l'utilisateur, qui n'a pas conscience que l'agent utilisera en réalité ces variables déjà configurées sous le capot lors du lancement de son processus.

## What Changes

- Enrichissement de la réponse de l'endpoint API `GET /api/preferences` pour y inclure un dictionnaire sécurisé/whitelisté `systemEnv` contenant les valeurs des variables d'environnement définies sur la machine hôte.
- Mise à jour de la boîte de dialogue de configuration de l'agent (`AgentSettingsModal`) pour intégrer visuellement ces valeurs système de manière adaptative :
  - Si une variable système est définie mais non surchargée par l'utilisateur, l'entrée montre la valeur système en placeholder et affiche un message indiquant que la variable sera héritée du système.
  - Si l'utilisateur saisit sa propre valeur, l'interface affiche explicitement qu'elle surcharge la valeur système actuelle.

## Capabilities

### New Capabilities

<!-- Aucun -->

### Modified Capabilities

- `agent-selection`: Mettre à jour l'exigence d'affichage de la configuration pour intégrer la visibilité de l'environnement système hôte et distinguer visuellement l'héritage système de la surcharge personnalisée de l'utilisateur.

## Impact

- `backend/internal/api/handlers/preferences.go` : Exposition dynamique d'un ensemble restreint et sécurisé de variables d'environnement via `systemEnv`.
- `frontend/src/lib/api.ts` : Extension de l'interface `Preferences` pour typer `systemEnv`.
- `frontend/src/components/AgentSettingsModal.tsx` : Intégration visuelle (Option A : placeholders et statuts adaptatifs) et détection des états de surcharge.
- `frontend/src/locales/fr/dialogs.json` & `frontend/src/locales/en/dialogs.json` : Ajout de traductions explicatives sur l'état d'héritage ou de surcharge système.
