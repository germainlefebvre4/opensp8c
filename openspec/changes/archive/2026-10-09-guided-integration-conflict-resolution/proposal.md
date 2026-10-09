# Proposal

## Why

Quand `main` avance pendant qu'un change attend en To Review (cas typique d'une validation humaine qui traîne), « Approuver & Fusionner » échoue avec `integration_conflict`. L'utilisateur reçoit alors un message générique, sans les fichiers en conflit (git les écrit sur stdout, le message ne reprend que le `stderr`, vide), et aucune piste de sortie : il doit résoudre le conflit à la main dans le worktree. Le circuit « Demander correction » sait déjà rendre le change à un worker, mais il n'est ni relié à cette erreur ni préparé pour ce cas.

Ce circuit a de plus un trou connu : une demande de correction laisse cochées les tâches de validation humaine, alors que la correction (ici, la résolution d'un conflit qui touche précisément le code validé) peut invalider ce qui a été vérifié. La coche périmée débloque « Approuver & Fusionner » sans nouvelle validation.

## What Changes

- La réponse `409 integration_conflict` de l'approbation porte la liste des fichiers en conflit (`files`) et la branche cible (`target`), obtenus avant l'abandon de l'intégration. L'intégration reste annulée, branche et worktree inchangés.
- Le dialogue d'approbation affiche ces fichiers sous le message d'erreur et propose « Demander la résolution au worker » ; le même parcours est disponible depuis le glisser-déposer vers Done.
- Ce bouton ouvre le dialogue de correction existant, **prérempli et éditable**, que l'utilisateur relit et confirme avant tout envoi : intégrer la branche cible dans la branche du change, résoudre les conflits listés en conservant les deux évolutions, relancer la validation. Rien n'est envoyé sans confirmation.
- `request-correction` accepte `reopen_human_tasks` (booléen, faux par défaut). Vrai, la correction décoche, dans le même commit que la tâche de correction, les tâches de validation humaine cochées, marqueur conservé. Le change revient en To Review avec ces tâches à re-valider : « Approuver & Fusionner » reste désactivé tant qu'elles ne sont pas recochées (ou revérifiées par la vérification UI).
- Le dialogue de correction affiche la case « Rouvrir les N tâches de validation humaine » dès qu'au moins une est cochée : **cochée par défaut** quand il est ouvert depuis un conflit d'intégration, **décochée par défaut** pour une correction ordinaire (comportement actuel inchangé).
- Hors périmètre, noté comme limite connue : une intégration de `main` **sans conflit** à l'approbation rejoue la commande de validation du workspace mais jamais la validation humaine ; la coche peut donc rester périmée dans ce cas.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `change-review-actions`: `integration_conflict` expose `target` et `files` ; `request-correction` accepte `reopen_human_tasks` et décoche les tâches humaines cochées dans le commit de correction ; le dialogue d'approbation propose la résolution guidée et le dialogue de correction accepte un texte initial et la case de réouverture.
- `human-review-tasks`: définit la réouverture d'une tâche de validation humaine par une correction (ligne repassée en `- [ ]`, marqueur conservé, compteurs cohérents).

## Impact

- Backend : `backend/internal/pool/worktree.go` (`IntegrateTarget` collecte les fichiers non fusionnés avant `merge --abort`), `backend/internal/pool/merge.go` (`IntegrationConflictError.Files`), `backend/internal/pool/review_actions.go` (`RequestCorrection` / `applyCorrection` avec l'option de réouverture), `backend/internal/openspec` (réouverture des tâches marquées d'un `tasks.md`), `backend/internal/api/handlers/review_actions.go` (corps de la réponse `integration_conflict`, champ `reopen_human_tasks`). Aucun changement du pool, du worker ni du circuit de reprise.
- Frontend : `lib/api.ts` (`ApiError.files`, `requestCorrection` avec options), `components/ApproveDialog.tsx`, `components/CorrectionDialog.tsx`, `components/DetailPanel.tsx`, `pages/KanbanPage.tsx`, `hooks/useReviewActions.ts`, locales `fr`/`en` (`dialogs.json`).
- Documentation : `docs/opensp8c/workflows.md` §5 (parcours guidé, réouverture des tâches humaines, limite de l'intégration sans conflit).
- API : ajout rétrocompatible (champs optionnels dans la réponse et dans le corps).
