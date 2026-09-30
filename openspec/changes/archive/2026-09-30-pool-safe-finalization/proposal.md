# Proposal

## Why

En mode `full-autonomy`, un worker peut déclarer un change « terminé » alors que le travail de l'agent a été perdu ou n'a jamais eu lieu. Le pool ne committe jamais rien : `MergeAndCleanup` supprime le worktree avec `git worktree remove --force` **avant** de fusionner `feature/<change>`, donc toute modification non committée disparaît et le merge d'une branche vide réussit. Par ailleurs, la vérification de complétion passe à vide quand `tasks.md` est absent (0/0), et un merge en échec ou un mode HITL relancent le même change toutes les ~5 s sans aucune raison affichée. Des worktrees réels de l'utilisateur montrent déjà ce schéma (branches sans commit, worktree avec 54 fichiers non committés). La finalisation doit devenir sûre avant de pouvoir faire confiance à l'enchaînement explore → dev.

## What Changes

- Le worker **committe** le travail de l'agent dans la branche `feature/<change>` avant toute finalisation (modes `full-autonomy` et `hitl-review`) ; un run qui n'a produit aucun changement est mis en pause au lieu d'être déclaré terminé.
- La fusion en `full-autonomy` devient **non destructive** : le worktree et la branche ne sont supprimés qu'après une fusion réussie, sans `--force` ; en cas d'échec, le merge est annulé, la branche et le worktree sont conservés et le worker passe à `paused` avec une raison lisible. Les fusions d'un même dépôt sont sérialisées et n'annulent jamais un merge démarré par l'utilisateur.
- La **vérification de complétion devient stricte** : `tasks.md` doit exister dans le worktree et contenir au moins une tâche, toutes cochées. Un pré-contrôle au provisionnement refuse un change absent du commit courant (non committé) avec une raison explicite.
- Un tour d'agent dont l'événement `result` signale une erreur (`is_error`, max_turns, quota, authentification) met le worker en pause au lieu d'être compté comme réussi.
- Un change **en attente de revue** (`hitl-review`) n'est plus redistribué par le dispatcher.
- **Délais maximums** : inactivité d'un tour d'agent et durée de la commande de validation ; dépassement = pause avec raison lisible.
- **Arrêt propre des processus** : les subprocess (agent, validation) tournent dans leur propre groupe de processus et le groupe entier est tué à l'annulation, à l'arrêt du pool et à l'arrêt du serveur.
- L'arrêt ou l'annulation d'un worker ne laisse plus de **pause fantôme** ; `Start` repart d'un état de pause vide ; un ancien worker ne supprime plus un worker plus récent qui a réutilisé son identifiant.
- Les worktrees sont **isolés par workspace** (`<worktrees>/<workspaceID>/wt-<change>`, avec reprise des worktrees existants à l'ancien emplacement) ; `Remove` ne supprime plus de répertoire par `RemoveAll` en secours. Le répertoire racine des worktrees devient injectable pour que les tests n'écrivent plus dans le vrai `~/.opensp8c`.

**BREAKING** : l'emplacement des worktrees change (`~/.opensp8c/worktrees/wt-<change>` → `~/.opensp8c/worktrees/<workspaceID>/wt-<change>`). Les worktrees existants sont repris tels quels depuis l'ancien emplacement, sans migration manuelle.

Hors périmètre (à traiter par d'autres changes) : le flux HITL d'approbation/correction côté backend (les boutons « Approuver & Fusionner » / « Demander correction » et le statut `to-review` ne sont pas alimentés par le backend), la persistance de l'état « en attente de revue » après redémarrage du serveur, le durcissement de l'API (CORS, écoute réseau, validation des noms) et la fiabilité du chat d'exploration.

## Capabilities

### New Capabilities
<!-- Aucune : le comportement est porté par la capacité existante agent-pool-orchestrator. -->

### Modified Capabilities
- `agent-pool-orchestrator`: isolation des worktrees par workspace ; vérification de complétion stricte ; changes en pause exclues du dispatcher sans pause fantôme à l'arrêt ; reprise de l'ancien emplacement des worktrees ; nouvelles exigences sur la présence du change dans le worktree, le commit du travail, la fusion sûre, l'exclusion des changes en attente de revue, le résultat d'agent en erreur, les délais maximums et l'arrêt des groupes de processus.

## Impact

- **Code** : `backend/internal/pool/` (`worktree.go`, `worker.go`, `manager.go`, `validation.go`, `registry.go`), `backend/internal/session/subprocess.go` (groupe de processus, partagé avec l'exploration), `backend/cmd/server/main.go` (arrêt du pool à l'arrêt du serveur), tests associés.
- **API** : aucune route ajoutée ni modifiée ; les raisons de blocage exposées par le statut de pool changent de contenu (nouveaux cas).
- **Données** : nouvel emplacement des worktrees ; les branches `feature/<change>` portent désormais des commits produits par la plateforme.
- **Dépendances** : aucune nouvelle dépendance ; usage de `syscall` (groupes de processus) avec variante de repli hors Unix.
- **Utilisateur** : en `full-autonomy`, un change doit être committé dans le dépôt avant d'être lancé ; sinon le worker se met en pause avec une raison explicite.
