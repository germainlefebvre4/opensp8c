## Why

L'étape `ui` de la vérification (voir `ui-verification-step`) laisse l'agent « choisir l'outil de navigation parmi ceux dont il dispose » : la plateforme ne passe à `claude` ni configuration MCP, ni liste d'outils autorisés, ni demande de permission réglée. Le résultat dépend donc de la config Claude de la machine hôte, n'est pas reproductible d'un poste à l'autre, et l'utilisateur ne peut ni choisir le mode de vérification autonome qu'il accepte ni guider l'agent sur le mode opératoire. Un agent lancé en `--print` ne peut de plus pas répondre à une demande de permission : il doit être déterministe plutôt que bloqué.

## What Changes

- Ajouter un réglage `uiDriver` (le **pilote** de la vérification UI) à la Configuration et au workspace, avec quatre valeurs : `auto` (défaut, la plateforme ne passe rien et l'agent utilise la config de l'hôte), `playwright` (navigateur Playwright headless et isolé fourni par la plateforme), `chrome` (intégration claude-in-chrome, qui pilote le vrai navigateur de l'utilisateur) et `custom` (configuration MCP et outils autorisés fournis par l'utilisateur).
- `chrome` n'est définissable **qu'au niveau workspace** : la Configuration le refuse, et un `chrome` présent au niveau Configuration dans un fichier édité à la main est ignoré. Un réglage de plateforme ne peut ainsi jamais l'activer d'un coup pour tous les workspaces.
- Ajouter pour `custom` les réglages `uiMcpConfig` (chemin d'un fichier de configuration MCP) et `uiAllowedTools` (liste d'outils autorisés), à la Configuration et au workspace.
- Ajouter un guidage texte libre `uiGuidance` à la Configuration et au workspace, cumulés dans cet ordre (jamais surchargés), ajouté au tour de l'agent sans pouvoir relâcher la lecture seule ni le contrat de verdict. L'écran de réglages avertit de ne pas y mettre de secret.
- Construire les paramètres de lancement de l'agent selon le pilote (`--mcp-config`, `--strict-mcp-config`, `--allowedTools`, `--chrome`, `--permission-prompts none`) : la plateforme les porte dans un nouveau champ `ExtraArgs` de `AgentConfig`, sans changer la signature de `session.StartSubprocess`. Seul l'agent Claude supporte un pilote autre que `auto`.
- Vérifier avant le tour de l'agent que le pilote est réellement disponible (événement `init` du flux : serveur MCP `connected`, outils du pilote listés) et refuser un `PASS` sans aucun appel d'outil du pilote.
- Adapter la consigne système selon le pilote (l'agent n'utilise que les outils du pilote, sans script jetable) et indiquer le pilote utilisé dans le rapport et dans le bandeau de vérification du DetailPanel.
- Ajouter une tâche d'abord : constater par un test réel le comportement de `claude --print` face à un outil non autorisé, avec et sans `--permission-prompts none`, ainsi que la visibilité des serveurs MCP et d'`--chrome` dans l'événement `init`.

Hors de ce change : un mode `auto` plus intelligent (détection automatique d'un pilote), l'application de `--permission-prompts none` à l'étape de conformité de `verification-stage`, le niveau change pour le pilote ou le guidage, et un fichier de guidage versionné dans le dépôt.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `verification-settings`: nouveaux réglages `uiDriver`, `uiMcpConfig`, `uiAllowedTools` et `uiGuidance` (niveaux, cascade, validation, `chrome` réservé au workspace, persistance rétrocompatible) et leurs champs dans les écrans Configuration et Settings.
- `ui-verification-step`: sélection du pilote et paramètres de lancement de l'agent, vérification de disponibilité avant le tour, refus d'un `PASS` sans usage d'outil du pilote, guidage utilisateur dans le tour, consigne système dépendante du pilote (remplace la clause « l'agent choisit l'outil »), pilote indiqué dans le rapport.
- `kanban-change-detail`: le bandeau de vérification affiche le pilote utilisé par l'étape `ui`.

## Impact

- Backend : `internal/verification` (types `Level`, `Resolved`, patchs, validation du pilote), `internal/preferences/verification.go` et `preferences.go` (champs persistés, patch, `chrome` refusé à la Configuration), `internal/agents/agents.go` (champ `ExtraArgs`, `SupportsDrivers`), `internal/pool/verify.go` et `verify_ui.go` (résolution du pilote, génération de la configuration MCP temporaire, observation du flux de l'agent, contrôle du `PASS`), `verify_ui_prompt.go` (consigne et tour), rapport de vérification (`driver`), `api/handlers/workspace_settings.go` et handler de préférences.
- Frontend : `lib/api.ts`, `lib/verification.ts`, `VerificationSettingsForm.tsx` (sélecteur de pilote, avertissement `chrome`, champs `custom`, guidage), `VerificationBanner.tsx`, clés fr et en.
- Exécution : `playwright` suppose `npx` et un Chromium Playwright sur l'hôte ; `chrome` suppose Chrome ouvert avec l'extension. L'image Docker finale (Alpine sans Node, navigateur ni CLI `claude`) n'est pas concernée.
- Aucune migration : un fichier existant sans ces champs équivaut à `auto` sans guidage.
