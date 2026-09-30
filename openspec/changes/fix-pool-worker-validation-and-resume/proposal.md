# Proposal

## Why

Lors de l'exécution de la Task `agent-role-model-settings` par le pool, le worker s'est arrêté sur « Tentatives de réparation épuisées » alors que l'agent avait terminé 31 tâches sur 32 et que `go build` passait dans son worktree. Trois défauts du pool se sont combinés :

- la validation lance `go test ./...` à la racine du worktree, alors que le module Go d'`opensp8c` est dans `backend/` (`directory prefix . does not contain main module`), et elle ne sait valider que du Go ;
- cette erreur d'environnement est renvoyée à l'agent comme une erreur de code, qui consomme ses tentatives de guérison sans pouvoir la corriger ;
- une fois le worker en pause, sa change reste « runnable » : le tick la relance aussitôt, `Provision` échoue (`a branch named 'feature/...' already exists`) parce qu'il teste la sortie standard de `git show-ref --quiet`, qui est toujours vide, et l'échec n'est visible que dans les logs. La branche existante n'est jamais réutilisée, si bien qu'aucune reprise de travail (pause ou relance après corrections) ne peut fonctionner.

De plus, l'utilisateur n'a aucun moyen de reprendre une change en pause : le seul geste disponible est d'arrêter puis relancer tout le pool, ce qui vide aussi la liste des workers en pause.

## What Changes

- La validation d'un worker utilise une commande résolue par workspace : réglage `validationCommand` (défaut global dans Configuration, surcharge par workspace dans Settings), sinon auto-détection (module Go et projet Node avec script `test`, à la racine du worktree et dans ses sous-répertoires directs).
- Une erreur d'environnement de validation (commande introuvable, répertoire absent, aucune commande détectable) met le worker en pause immédiatement avec une raison lisible, sans tour de guérison.
- `Provision` détecte l'existence de la branche par le code retour de git et réutilise la branche et le worktree existants au lieu d'échouer.
- Un échec de `Provision` met le worker en pause avec une raison lisible, au lieu d'être silencieux.
- Une change dont un worker est en pause n'est plus relancée par le dispatcher tant que l'utilisateur ne la reprend pas explicitement.
- Nouvel endpoint de reprise d'un worker en pause et bouton « Reprendre » par change en pause dans le panneau d'état du pool du Kanban.
- Changement de comportement (non cassant pour l'API) : sans réglage, la commande de validation n'est plus `go test ./...` à la racine du worktree mais le résultat de l'auto-détection.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `agent-pool-orchestrator`: commande de validation résolue par workspace avec auto-détection, pause immédiate sur erreur d'environnement, reprise de la branche et du worktree existants, échec de provisionnement visible, changes en pause non relancées, reprise explicite d'un worker en pause.
- `agent-pool-ui`: action « Reprendre » par worker en pause dans le panneau d'état du pool.
- `workspace-settings`: surcharge de la commande de validation dans Settings > Agent Pool.
- `platform-configuration`: défaut global de la commande de validation dans Configuration > Agent Pool.

## Impact

- Backend : `backend/internal/pool/worker.go` (validation, échec de provisionnement), `worktree.go` (`Provision`), `manager.go` (filtre du tick, reprise), nouveau détecteur de commande de validation, `backend/internal/preferences/roles.go` (champ du réglage de pool, résolution, patch), `backend/internal/api/handlers/pool.go` et `router.go` (route de reprise), `handlers/workspace_settings.go` et handler des préférences (exposition du nouveau champ).
- Frontend : `AgentPoolModal.tsx` (bouton Reprendre), `AgentPoolSettingsForm.tsx` (champ de commande de validation), `lib/api.ts` (appel de reprise, type de réglage), locales `fr` et `en`.
- Docs : `docs/opensp8c` (architecture du pool) et spécifications ci-dessus.
- Hors périmètre : l'affichage de la ligne JSON brute comme activité (`extractActivity`) et le traitement des tâches manuelles dans `tasks.md`.
