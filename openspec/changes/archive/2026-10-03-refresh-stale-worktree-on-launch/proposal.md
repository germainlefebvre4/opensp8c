## Why

Le worker crée la branche et le worktree d'un change **avant** de vérifier que `tasks.md` y est présent. Si le change n'était pas encore committé, la vérification échoue et le worker passe en pause, mais la branche `feature/<change>` et son worktree restent, figés sur le commit antérieur. L'utilisateur committe alors le change comme le message le demande, et la reprise échoue de nouveau avec le même message : `Provision` réutilise la branche existante « telle quelle » et ne la met jamais à jour. Le seul remède est de supprimer à la main la branche et le worktree.

## What Changes

- Avant de provisionner une branche, le worker vérifie que `tasks.md` du change est committé dans la branche courante du dépôt ; sinon il passe en pause **sans créer ni branche ni worktree**.
- Lorsqu'une branche `feature/<change>` existe déjà mais que son worktree ne contient pas `tasks.md` alors que le change est committé dans la branche courante, et que cette branche n'a **aucun travail** (worktree propre, aucun commit que la branche courante n'a pas), le backend la recrée à partir de la branche courante, sans perte possible, puis poursuit.
- Si cette branche porte du travail (modifications non committées ou commits propres), rien n'est supprimé ni réécrit : le worker passe en pause avec une raison distincte qui explique que la branche est en retard sur le change et ce que l'utilisateur peut faire.
- Le message de pause « doit être committé » ne reste utilisé que lorsque le change n'est effectivement pas committé.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `agent-pool-orchestrator` : exigences « Présence du change dans le worktree » (vérification avant provisionnement, raison de pause distincte) et « Reprise de la branche et du worktree existants » (recréation d'une branche périmée sans travail).

## Impact

- Backend : `internal/pool/worker.go` (ordre de la garde, branche « worktree périmé »), `internal/pool/worktree.go` (vérification qu'un fichier est committé dans la branche courante, recréation d'une branche sans travail, ré-enregistrement de la base).
- Tests : `worker_test.go`, `worktree_test.go` (change non committé sans orphelin ; branche périmée sans travail recréée ; branche périmée avec travail conservée).
- Aucun changement d'API ni d'interface ; seul le texte des raisons de pause évolue.
- Complète `release-paused-worker-on-demote`, qui permet de sortir d'une pause par la rétrogradation du change.
