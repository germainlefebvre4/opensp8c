# Proposal

## Why

L'audit des changes `review-state`, `review-panel` et `review-actions` a relevé deux écarts restants.

1. **Nettoyage en échec après une fusion réussie.** Dans `ApproveReview`, si le merge aboutit mais que la suppression du worktree ou de la branche échoue (par exemple un fichier non suivi dans le worktree fait refuser `git worktree remove`), `integrateAndMerge` renvoie `merged=true` avec une `CleanupError`, que le handler traduit en `500 approve_failed`. Le marqueur de revue n'est pas levé et la branche existe toujours : `to-review` continue de primer, la carte reste dans To Review alors que `main` contient déjà le travail, et l'interface affiche une erreur pour une fusion pourtant réussie. La spec dit « marqueur conservé en cas d'échec » sans traiter ce cas.
2. **Documentation périmée.** `README.md` affirme qu'en `hitl-review` le change n'est pas redistribué « until the pool is restarted » : faux depuis le marqueur persistant. Les pages générées sous `docs/opensp8c/` décrivent encore l'ancien flux (clic sur la carte qui ouvre un panneau, feedback injecté dans l'invite système, statut To Review déduit des seules tâches cochées).

## What Changes

- **Approbation tolérante au nettoyage** : quand le merge a eu lieu, `ApproveReview` lève le marqueur de revue (le change passe en Done d'après le `tasks.md` fusionné), publie `change_updated`, et répond `200` avec la branche cible et un **avertissement `cleanup_incomplete`** portant la liste `remaining` de ce qui reste à nettoyer à la main (`worktree`, `branch` et/ou `marker`). Ce n'est plus présenté comme un échec. Si le marqueur lui-même ne peut pas être levé, une nouvelle approbation reconnaît la branche comme déjà fusionnée (sans nouvelle intégration, validation ni fusion) et termine le nettoyage.
- **Interface** : un succès d'approbation avec avertissement ferme le dialogue, la carte passe en Done, et un toast d'avertissement indique ce qu'il reste à nettoyer ; aucune erreur n'est affichée.
- **README** : correction de la phrase sur `hitl-review` et description succincte du cycle de revue (To Review persistant, onglet Revue, Approuver & Fusionner, Demander correction).
- **Documentation générée** : régénération de `docs/opensp8c/` à partir des specs à jour, plutôt qu'une édition manuelle.

Hors périmètre : la finalisation du worker `full-autonomy` (qui met déjà le worker en pause sur échec de nettoyage), l'instabilité du test `TestManager_PauseThenResumeReusesWorktree` sous charge, et l'absence de `Content-Type` JSON sur les endpoints de lecture de la revue.

## Capabilities

### New Capabilities

### Modified Capabilities
- `change-review-actions`: l'approbation distingue l'échec avant fusion (état conservé) du nettoyage incomplet après fusion (marqueur levé, succès avec avertissement) ; l'interface confirme le succès et affiche l'avertissement.

## Impact

- Backend : `backend/internal/pool/review_actions.go` (traitement de `CleanupError`, levée du marqueur, résultat enrichi), `backend/internal/api/handlers/review_actions.go` (réponse `200` avec `warning`), tests associés.
- Frontend : `lib/api.ts` (type `ApproveResult`), `hooks/useReviewActions.ts` (toast d'avertissement), `components/ui/Toast.tsx` (variante `warning` et durée configurable), locales fr/en (`dialogs.json`), tests associés.
- Documentation : `README.md` ; régénération de `docs/opensp8c/*.md` via l'endpoint de génération existant.
- Aucun changement du worker, du statut `to-review` ni des autres endpoints.
