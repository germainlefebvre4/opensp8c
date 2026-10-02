# Proposal

## Why

Relire le travail d'un change en revue suppose de voir ce qu'il apporte. Aujourd'hui le « panneau de review » décrit dans la spec `agent-pool-ui` (liste des fichiers, diff) n'existe pas, et l'utilisateur doit ouvrir le worktree à la main avec git. Sans lecture du diff dans l'application, les actions de revue (`review-actions`) obligent à approuver à l'aveugle.

## What Changes

- Deux endpoints en lecture seule exposent la revue d'un change : la **liste des fichiers modifiés** (chemin, statut, lignes ajoutées/supprimées, branche de base) et le **patch d'un fichier** à la demande. Le diff est calculé contre le point de divergence avec la branche de base (`base...feature/<change>`) : c'est ce que la fusion apporterait réellement.
- Un troisième endpoint en lecture seule renvoie le **contenu final d'un fichier** (version de `feature/<change>`), pour afficher les fichiers Markdown rendus.
- Un **onglet « Revue »** est ajouté au `DetailPanel`, visible pour un change en To Review, dans les usages existants de la plateforme : fichiers groupés en **OpenSpec** (tout chemin sous `openspec/`) et **code**, diff par fichier dépliable (lib `diff` déjà présente côté frontend), toggle diff/rendu pour les fichiers Markdown (composant `Markdown`), lien vers le dernier run du worker (`AgentRunPanel`), textes i18n fr/en, rafraîchissement par les événements SSE existants.
- Le requirement « Panneau interactif de Review HITL » de `agent-pool-ui` est **retiré** : il mêlait affichage, bouton d'approbation et champ de feedback dans un panneau propre ouvert « au clic sur la carte ». L'affichage devient la capability `change-review-panel` (onglet du `DetailPanel`) ; l'approbation et la correction sont dans `change-review-actions` (change `review-actions`). Pour éviter une spec contradictoire, archiver `review-panel` avant ou avec `review-actions`.

Hors périmètre : les actions de revue (change `review-actions`), les commentaires par ligne, l'édition du code depuis l'interface, la revue des changes sans marqueur de revue.

## Capabilities

### New Capabilities
- `change-review-panel`: lecture de la revue d'un change (fichiers et diff via API) et onglet « Revue » du `DetailPanel`.

### Modified Capabilities
- `agent-pool-ui`: retrait du requirement « Panneau interactif de Review HITL (Human-In-The-Loop) » (affichage repris par `change-review-panel`, approbation et correction par `change-review-actions`).

## Impact

- Backend : nouveaux endpoints `GET /api/workspaces/{id}/changes/{name}/review`, `.../review/diff?path=` et `.../review/file?path=` (handler dédié, `router.go`) ; méthodes git en lecture seule dans `pool/worktree.go` (liste des fichiers, patch d'un fichier, contenu final) ; validation stricte du paramètre `path`.
- Frontend : `DetailPanel.tsx` (nouvel onglet), nouveau composant de revue, hooks de données, clés i18n fr/en, réutilisation de `Markdown`, `AgentRunPanel` et de la lib `diff`.
- Dépend du change `review-state` (statut `to-review`) ; indépendant de `review-actions` (livrable avant ou après).
