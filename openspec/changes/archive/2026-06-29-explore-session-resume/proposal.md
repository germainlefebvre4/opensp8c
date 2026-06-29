## Why

Les sessions d'exploration disparaissent après 30 minutes d'inactivité ou à la fermeture du navigateur. Quand un utilisateur revient le lendemain ou deux jours plus tard pour continuer sa réflexion, Claude repart de zéro — la continuité de pensée est perdue. Pour un outil de réflexion collaborative, c'est une rupture fondamentale dans le workflow.

## What Changes

- Le subprocess Claude est lancé avec `--session-id <uuid>` à la création d'une named session (UUID généré par le backend)
- L'UUID est persisté dans `preferences.json` sous la clé de la session (`workspaceID/changeName`)
- Lors des lancements suivants du subprocess (après timeout d'inactivité ou reconnexion), le backend passe `--resume <uuid>` pour que Claude reprenne son contexte complet
- Le subprocess peut être tué après 30 minutes d'inactivité comme avant — seul l'UUID survit dans `preferences.json`
- Les sessions anonymes ne sont pas concernées (pas d'UUID persisté, comportement inchangé)

## Capabilities

### New Capabilities

- `explore-session-resume` : Mécanisme de persistance et de reprise d'une session Claude. Génération d'un UUID côté backend à la première ouverture d'une named session, stockage dans preferences.json, et passage de `--resume <uuid>` aux démarrages suivants du subprocess.

### Modified Capabilities

- `explore-session` : Le scénario "Session expirée" change de comportement — au lieu de proposer "relancer une nouvelle session", le système reprend automatiquement la session Claude existante via `--resume`. L'utilisateur continue là où il s'était arrêté.
- `agent-selection` : Le schéma `preferences.json` s'étend. Chaque named session stocke désormais `claudeSessionId` en plus de l'agent : `workspaceID/changeName → { agent, claudeSessionId }`.

## Impact

- **Backend** : `internal/session/subprocess.go` — ajout des args `--session-id` / `--resume` selon contexte ; `internal/session/manager.go` — lookup du `claudeSessionId` depuis preferences au démarrage, stockage à la première création ; `internal/preferences/` — extension du struct `Session` avec `ClaudeSessionId string`
- **Frontend** : aucun changement — la continuité est transparente pour l'UI
- **Anonymous sessions** : comportement inchangé, pas de persistance d'UUID
