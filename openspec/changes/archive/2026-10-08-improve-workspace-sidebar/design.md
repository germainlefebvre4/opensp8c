# Design

## Context

- `WorkspaceSidebar.tsx` rend une ligne par projet avec des pastilles chiffrées issues de `task_counts`, un `X` de suppression visible au survol et un `onClick` limité au `<span>` du nom. `BADGE_COLORS`/`BADGE_ORDER` y dupliquent les couleurs de `STATUS_STYLES` (`KanbanColumn.tsx`) et ont dérivé : `ready` et `verifying` manquent.
- `GET /api/workspaces` (`handlers/workspace.go`, `List`) appelle `openspec.ListChanges` brut pour compter les statuts. Le Kanban (`KanbanHandler.ListChanges`) applique ensuite une couche « live » : état des workers (`activeWorkerChanges`), progression de la worktree ou de la branche (`ApplyWorktreeProgress`, `ApplyBranchProgress`), état de vérification (`applyVerificationState`). La raison de pause d'un worker (`BlockedReason`) n'est exposée que par le détail d'un change.
- `KanbanPage` pilote le DetailPanel par un état local `detailOpen` ; aucune URL ne désigne un change ouvert.
- `kanbanLayout.ts` mesure la largeur de la ligne de contenu (hors sidebar) : changer la largeur de la sidebar ne demande aucune modification de sa logique. Le palier « tout déplié + panel 420 » exige 1616px de contenu ; avec une sidebar de 288px il tient jusqu'à une fenêtre de ~1904px.
- Un composant `ConfirmDialog` (Radix) et un système de `Toast` existent déjà.

Voir `proposal.md` pour la motivation et `specs/` pour les exigences.

## Goals / Non-Goals

**Goals:**
- Calculer les signaux d'action côté backend dans le passage existant de `GET /api/workspaces`, sans endpoint ni polling supplémentaire.
- Une seule source de couleurs de statut pour sidebar et Kanban.
- Logique de présentation (tri, plafonds, répartition de la barre) isolée dans des fonctions pures testables sans layout, dans l'esprit de `kanbanLayout.ts`.

**Non-Goals:**
- Pas de persistance de l'état déplié ou replié, pas de recherche, de tri ou d'épinglage des projets.
- Pas d'explorateur de fichiers système pour l'ajout (écart de spec existant, hors périmètre).
- Pas de modification de `kanbanLayout.ts` ni du comportement du Kanban hors ouverture d'un change depuis l'URL.
- Pas de temps réel : le rafraîchissement reste le polling de 15s de `useWorkspaces`.

## Decisions

### 1. `attention` calculé dans `workspace.List` à partir d'une liste de changes « live » partagée
Le calcul des signaux a besoin de l'état des workers, de la vérification et du `tasks.md` effectif (branche ou worktree). Plutôt que de le dupliquer, la couche « live » de `KanbanHandler.ListChanges` (workers, progression, vérification) est extraite en une fonction réutilisable, appelée par le Kanban et par `workspace.List`. Conséquence assumée : `task_counts` du sidebar et colonnes du Kanban reposent alors sur le même calcul, ce qui corrige l'écart connu où un change en cours via une worktree restait compté `todo` dans le sidebar.

Nouveau type `workspace.Attention` : `{ change string, signals []Signal }` avec `Signal{ kind string, reason string }` (`kind` ∈ `review`, `paused`, `hitl`, `verify-failed`). Une fonction pure `ComputeAttention(changes, held, tasksFor)` produit la liste et son ordre (voir 2), testée sans git.

*Alternative écartée : un endpoint `GET /workspaces/{id}/attention` interrogé à la demande.* Il évite de charger tous les workspaces mais ajoute un polling par projet visible (le compteur doit être à jour pour tous les projets), alors que `workspace.List` est déjà interrogé toutes les 15s.

*Alternative écartée : réutiliser `GET /changes` côté frontend pour chaque projet.* N requêtes toutes les 15s, et la raison de pause n'y figure pas.

### 2. Priorité, tri et plafond calculés côté backend
`ComputeAttention` renvoie les changes déjà triés (signal le plus bloquant : `paused` > `verify-failed` > `hitl` > `review`, puis nom) et les signaux d'un change dans le même ordre ; le frontend n'a qu'à rendre. Le plafond de trois lignes `hitl` et la ligne « +N autres » sont une règle d'affichage : le backend renvoie toutes les tâches, le frontend plafonne (fonction pure `capSignals`). La barre segmentée est aussi une fonction pure `segments(task_counts)` (ordre Kanban, proportions, statuts à 0 omis).

### 3. Lecture HITL bornée
Le `tasks.md` d'un change est lu seulement pour les changes ni `done` ni `to-explore` (un `tasks.md` y est soit absent, soit sans objet), via la même résolution que le Kanban (worktree du worker, sinon branche `feature/<change>` par `git show`, sinon fichier du dépôt). `ParseTaskListContent` fournit déjà `HumanReview` et le texte sans marqueur ; aucune nouvelle analyse. Les `git show` sont limités aux changes qui ont une branche (`HasBranch`).

### 4. Raison de pause : exposée par `heldWorker`
`heldWorker.BlockedReason` existe déjà ; `ComputeAttention` le consomme directement. Aucune modification du pool.

### 5. Couleurs de statut : module partagé
Un module `frontend/src/lib/statusColors.ts` exporte `STATUS_ORDER` (ordre Kanban des 7 statuts), le `dot` et le `badge` de chaque statut ; `KanbanColumn` et `WorkspaceSidebar` l'importent. `BADGE_COLORS` et `BADGE_ORDER` sont supprimés. Un test vérifie que chaque statut de `STATUS_ORDER` a une couleur et que `KanbanColumn` n'en définit pas de locale.

*Alternative écartée : laisser deux tables et ajouter un test de cohérence.* Garde la dérive possible à la prochaine colonne ; la source unique l'exclut.

### 6. Structure frontend de la sidebar
`WorkspaceSidebar` est découpé : `WorkspaceItem` (ligne 1, barre, chevron, menu `…`), `StatusBar` (barre segmentée avec `title` par segment), `AttentionList` (vue dépliée), `SidebarRail` (état replié). Le menu `…` utilise `@radix-ui/react-dropdown-menu` (nouvelle dépendance, de la même famille que `react-dialog`, `react-tooltip` et `react-scroll-area` déjà utilisées) pour obtenir gratuitement navigation clavier, focus et ARIA ; la confirmation réutilise `ConfirmDialog`. *Alternative écartée : popover maison, qui obligerait à réimplémenter focus trap, flèches et Échap.* La ligne projet est un `div` `role="button"`/`tabIndex=0` (`aria-current` sur l'actif), les contrôles internes (chevron, menu) arrêtent la propagation du clic et ne sont pas imbriqués dans un autre bouton.

L'état déplié est un `Set<string>` d'ids dans le state de `WorkspaceSidebar` (session, non persisté). Un projet dont `attention` devient vide perd son chevron et se rend replié sans modifier le set.

### 7. Ouverture d'un change : paramètre d'URL `change`
Un clic sur un change navigue vers `/?workspace=<id>&change=<name>` (Kanban). `KanbanPage` lit `change` au montage et à chaque modification de ce paramètre : si le change existe dans la liste, il ouvre `detailOpen` ; sinon il ne fait rien et retire le paramètre. Fermer le DetailPanel retire le paramètre. Cela reste compatible avec le paramètre `workspace` géré par `Layout` et la mémorisation de la dernière route des projets.

*Alternative écartée : état de navigation (`location.state`).* Perdu au rechargement et invisible dans l'URL, alors que `workspace-url-persistence` fait déjà de l'URL le support de l'état de navigation.

### 8. Largeur et rail
Sidebar ouverte `w-72` (288px), rail `w-10` (aligné sur `RAIL_WIDTH = 40` du Kanban). Le rail affiche une pastille d'initiales (deux premières lettres significatives du nom) avec `title` = nom et un point d'attention. Contrairement au Kanban, le repli de la sidebar reste manuel (bouton toggle existant) : pas de repli automatique selon la largeur dans ce change.

## Risks / Trade-offs

- [Le calcul de `attention` ajoute des `git show` toutes les 15s par workspace] → limité aux changes avec branche et ni `done` ni `to-explore` (décision 3) ; si la mesure montre un coût notable, mémoïser par (change, hash de la branche).
- [Extraire la couche « live » touche `KanbanHandler.ListChanges`, qui a des modifications locales en cours] → extraire sans changer son comportement, couvert par les tests existants de `kanban`, avant d'y brancher `workspace.List`.
- [Aligner `task_counts` sur le Kanban change des chiffres du sidebar par rapport à aujourd'hui] → c'est la correction voulue ; mentionnée dans le proposal et couverte par un test sur un change `todo` avec worktree active.
- [Sidebar à 288px réduit de 32px la place du Kanban] → `kanbanLayout.ts` mesure déjà le conteneur et dégrade par paliers ; vérifié à 1920px avec panel ouvert.
- [Beaucoup de signaux `hitl` allongent la vue dépliée] → plafond de trois lignes et « +N autres ».
- [Compteur et liste se désynchronisent entre deux polls] → les deux viennent de la même réponse `attention`, donc cohérents par construction.

## Migration Plan

Aucune migration de données : le champ `attention` est additif dans `GET /api/workspaces` et `task_counts` conserve sa forme. Le déploiement est celui du binaire et du frontend ; le retour arrière consiste à revenir au commit précédent.

## Open Questions

- Libellés exacts des types de signaux (« À revoir », « En pause », « Validation humaine », « Vérification échouée ») : à fixer avec les locales en/fr à l'implémentation, sans effet sur les specs.
