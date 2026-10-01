# Proposal

## Why

En mode `full-autonomy`, un worker dont le `ctx` est annulé (Stop du pool, `Unlaunch --force`) peut quand même fusionner son travail dans la branche courante de l'utilisateur : aucune vérification d'annulation n'existe entre la fin de la validation et le `git merge` (étapes 6 à 8 de `runWorker`). La fenêtre couvre le `git commit` (et ses hooks), puis l'attente de `mergeMu` pendant le merge d'un autre worker. Un `Stop` suivi d'un `Start` rapide aggrave le cas : l'ancien worker, encore en attente du verrou, fusionne `feature/X` et supprime le worktree que le nouveau worker de `X` utilise. De plus `Unlaunch --force` rend la main avant la fin réelle du worker, donc le statut renvoyé ne reflète pas l'état du dépôt.

## What Changes

- Le worker vérifie l'annulation de son `ctx` avant le commit du travail de l'agent et, surtout, juste après l'acquisition de `mergeMu` : si annulé, il libère le verrou, ne fusionne pas, conserve branche et worktree, et termine avec l'issue `stopped`. L'acquisition du verrou de fusion et la vérification forment le point de non-retour : un merge démarré va toujours jusqu'au bout.
- `git merge` n'est PAS rendu annulable (pas de `exec.CommandContext`) : tuer git en plein merge laisserait `MERGE_HEAD` ou `index.lock` dans le dépôt de l'utilisateur.
- Chaque worker expose un signal de fin portant son issue (`stopped`, `completed`, `paused`, `awaiting-review`) ; l'annulation d'un worker par changement peut attendre cette fin avec un délai maximum.
- `Unlaunch --force` attend la fin réelle du worker avant d'agir et rapporte l'état réel :
  - issue `stopped` : `launched: false` est écrit, réponse 204 (inchangé) ;
  - issue `completed` (le merge a eu lieu avant que l'annulation soit honorée) : le démote est **refusé**, réponse 409 indiquant que le change est déjà fusionné dans la branche cible, `launched` reste inchangé ;
  - délai dépassé (worker encore en cours de fusion) : pas de démote, erreur explicite invitant à réessayer.
- Le frontend affiche un message distinct pour le refus « déjà fusionné » au lieu du toast générique d'échec.

Hors périmètre : validation après merge, résolution automatique des conflits, état « mergé mais non nettoyé », timeout de `git merge`, branche de départ vs branche cible, usage de `Discard` (points de vigilance identifiés mais traités séparément).

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `agent-pool-orchestrator`: l'exigence « Fusion sûre en mode full-autonomy » ajoute le contrat d'annulation (aucune fusion après annulation constatée avant le point de non-retour, y compris après Stop puis Start rapide) et l'issue `stopped` correspondante.
- `kanban-ready-column`: l'exigence « Rétrogradation manuelle To Do → Ready » précise que `force=true` attend la fin réelle du worker, et définit les réponses selon l'issue (204, 409 « déjà fusionné », délai dépassé).

## Impact

- Backend : `backend/internal/pool/worker.go` (vérifications `ctx`, issue du run), `backend/internal/pool/manager.go` et `types.go` (signal de fin par worker, attente avec délai dans `CancelWorkerForChange` ou méthode dédiée), `backend/internal/api/handlers/kanban.go` (handler `Unlaunch`).
- Frontend : `frontend/src/pages/KanbanPage.tsx` (`handleConfirmUnlaunchWorker`, message du 409 « déjà fusionné ») et les fichiers i18n correspondants.
- Tests : `backend/internal/pool/finalize_test.go` / `worker_test.go` (test déterministe qui tient `m.mergeMu`), tests du handler kanban.
- Aucune nouvelle dépendance ; pas de changement de format de données persistées.
