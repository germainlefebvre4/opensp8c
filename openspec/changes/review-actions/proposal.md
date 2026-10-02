# Proposal

## Why

Une fois la colonne To Review alimentée (change `review-state`), l'utilisateur ne peut rien en faire : les boutons « Approuver & Fusionner » et « Demander correction » du `DetailPanel` n'ont pas de `onClick`, le routeur n'expose aucun endpoint de revue, et le worker en `hitl-review` ne fusionne jamais. Un change en revue est donc un cul-de-sac. De plus, aucune transition de drag-and-drop ne permet d'en sortir : `fix-kanban-valid-drops` ramène la table des transitions à la spec (sans aucune entrée depuis To Review) et ce change l'étend.

## What Changes

- **Approuver & Fusionner** : un endpoint backend fusionne `feature/<change>` dans la branche courante, après intégration de la branche cible et revalidation si elle a avancé, en réutilisant le verrou de fusion et les garde-fous du mode `full-autonomy`. En cas de succès le change passe en Done (le `tasks.md` de main est coché par la fusion) ; en cas d'échec il reste en To Review avec une erreur explicite. L'endpoint ne suppose pas que le pool tourne et ne lance pas de heal d'agent.
- **Demander correction** : l'utilisateur saisit un retour (obligatoire). Le backend l'ajoute comme **tâche décochée dans une section `## Corrections` du `tasks.md` du worktree**, la commite dans `feature/<change>`, lève le marqueur de revue ; le change redevient éligible et la boucle existante du worker (`/opsx:apply`, contrôle de complétion) le reprend sans logique nouvelle.
- **Dialogues UI** : dialogue de saisie du retour de correction (partagé par le bouton et le drag) et confirmation d'approbation, avec états de chargement et erreurs lisibles.
- **Drag-and-drop** : prolonge la table de `fix-kanban-valid-drops` (`lib/kanbanDrops.ts`) : `to-review → done` approuve, `to-review → in-progress` ouvre le dialogue de correction ; To Review reste refusée comme cible ; les cartes To Review deviennent draggables.
- **Suppression d'un change en revue** : elle nettoie aussi la branche, le worktree et le marqueur (aujourd'hui seul le dossier du change est supprimé, ce qui laisserait des restes orphelins).
- La spec de l'intégration avant fusion précise que le worker `hitl-review` ne fusionne pas mais que l'approbation utilisateur déclenche intégration, revalidation et fusion.

Hors périmètre : le panneau fichiers/diff (change `review-panel`), le heal automatique à l'approbation et une session interactive de correction. Après une correction, le change repasse en In Progress dès que son worker coche des tâches grâce à `pool-worktree-progress` (livré).

## Capabilities

### New Capabilities
- `change-review-actions`: approbation (fusion), demande de correction par tâches `## Corrections`, dialogues UI associés et suppression propre d'un change en revue.

### Modified Capabilities
- `kanban-drag-drop`: nouvelles transitions `to-review → done` et `to-review → in-progress` ; la règle « In Progress, To Review et Done ne sont jamais une cible » devient « To Review n'est jamais une cible ; In Progress et Done ne le sont que depuis To Review ».
- `kanban-board`: les cartes To Review sont draggables.
- `agent-pool-orchestrator`: la règle d'intégration/revalidation précise que le worker `hitl-review` ne fusionne pas et que l'approbation utilisateur s'en charge.

## Impact

- Backend : nouveaux endpoints sous `/api/workspaces/{id}/changes/{name}/review/` (approve, request-correction) dans `router.go` et un handler dédié ; extraction de la séquence intégration/validation/fusion de `pool/worker.go` en une fonction partagée avec l'approbation ; `runValidation` rendu indépendant de `Worker` ; écriture de `tasks.md` du worktree et commit ; `DeleteChange` (`kanban.go`) nettoie branche/worktree d'un change en revue.
- Frontend : `DetailPanel.tsx` (onglet Actions), `KanbanPage.tsx` (transitions et handlers de drop), nouveaux dialogues, hooks et clés i18n fr/en.
- Dépend du change `review-state` (marqueur et statut `to-review`) et de `fix-kanban-valid-drops`, à archiver avant : le delta `kanban-drag-drop` de ce change est écrit sur le bloc modifié par ce dernier ; à régénérer si son texte change avant son archivage.
- Remplace les scénarios « Approbation » et « Demande de corrections » du requirement « Panneau interactif de Review HITL » de la spec `agent-pool-ui` (ce requirement est retiré par le change `review-panel`, qui en reprend l'affichage) : le comportement de correction n'est plus « feedback injecté dans l'invite système » mais « tâche ajoutée à `tasks.md` ». Archiver `review-panel` avant ou avec ce change pour ne pas laisser deux specs contradictoires.
