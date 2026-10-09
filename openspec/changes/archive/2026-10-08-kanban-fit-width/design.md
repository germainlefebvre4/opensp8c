# Design

## Context

Aujourd'hui (`KanbanPage.tsx`, `KanbanColumn.tsx`) : six slots `flex-1 min-w-[220px]`, `gap-3`, conteneur `p-4`, `overflow-x-auto` + `min-w-max`, DetailPanel fixe `w-[420px]` poussant les colonnes. Le plancher de largeur est de ~1412px sans panel, ~1832px avec. Voir `proposal.md` pour la motivation.

Contraintes à respecter :
- `createClampToRectModifier` et `dragContainerRectRef` s'appuient sur `columnsContainerRef` ; ce conteneur doit rester celui qui englobe les colonnes.
- `kanbanCollision.ts` ne reconnaît que des droppables dont l'id est dans `DEFAULT_KANBAN_COLUMN_IDS` (`done` inclus).
- Le slot Done/Archived empile deux `KanbanColumn` ; `archived` a déjà son propre chevron (`collapsible`), indépendant.

## Goals / Non-Goals

**Goals:**
- Zéro scrollbar horizontale tant que cinq slots dépliés + rail tiennent (~1046px).
- Comportement prévisible : chaque palier de l'échelle est déterministe selon la largeur du conteneur.
- Logique de décision isolée dans une fonction pure testable sans layout (jsdom n'en fournit pas).

**Non-Goals:**
- Pas de redimensionnement manuel du DetailPanel (poignée), pas de persistance du repli entre sessions.
- Pas de mode compact des cartes (container queries) ni de repli d'autres colonnes que Done/Archived.
- Pas de changement backend.

## Decisions

### 1. Mode de layout calculé en JS à partir de la largeur mesurée
Une fonction pure `computeKanbanLayout({ width, panelOpen, manualFolded })` (dans `frontend/src/lib/kanbanLayout.ts`) renvoie `{ doneFolded, panelMode: 'none' | 'inline' | 'overlay', panelWidth, scroll }`. `KanbanPage` mesure la largeur du conteneur parent (colonnes + panel) via `ResizeObserver` et passe le résultat au rendu.

Besoins en largeur (constantes exportées) : colonne 190, gap 8, padding 16 (2×8), rail 40, panel 320–420.
`need(expanded) = 6×190 + 5×8 + 16 = 1196`, `need(folded) = 5×190 + 40 + 5×8 + 16 = 1046`.

Échelle pour une largeur `W` avec panel ouvert et sans surcharge :
1. `W − 420 ≥ 1196` → déplié, panel inline 420.
2. `W − 320 ≥ 1196` → déplié, panel inline `W − 1196` (entre 320 et 420).
3. `W − 320 ≥ 1046` → rail, panel inline `min(420, W − 1046)`.
4. `W ≥ 1046` → rail, panel en overlay.
5. sinon → rail, panel en overlay, scroll horizontal.

Panel fermé : déplié si `W ≥ 1196`, rail sinon ; scroll si `W < 1046`.
Le palier overlay conserve le rail (échelle cumulative) pour éviter un dépli/repli en yo-yo autour des seuils.

*Alternative écartée : container queries CSS pures.* Elles expriment bien les seuils, mais la surcharge manuelle, l'effacement de l'overlay pendant un drag et la bascule de rendu du slot Done/Archived imposent de toute façon un état JS ; une fonction pure est en plus testable à l'identique de `kanbanCollision`.

### 2. Surcharge manuelle : état tri-valué en mémoire
`manualFolded: boolean | null` dans `KanbanPage` (non persisté, perdu au rechargement). `null` = automatique. Un clic sur le chevron bascule vers l'état opposé de l'état courant ; si cet état égale celui que le calcul automatique choisirait, `manualFolded` repasse à `null`. Ainsi aucun contrôle supplémentaire n'est nécessaire pour « revenir en auto ». Avec surcharge « déplié » forcée et espace insuffisant, l'échelle continue avec `need(expanded)` : overlay puis scroll. Avec surcharge « replié », `need(folded)` s'applique toujours.

*Alternative écartée : persister en `localStorage`.* Un état oublié masquerait Done des semaines plus tard sans explication ; le repli auto couvre le cas courant.

### 3. Rail = composant dédié, droppable `done` conservé
Quand `doneFolded`, le slot rend un `DoneRail` (≈40px) à la place des deux `KanbanColumn` : il enregistre `useDroppable({ id: 'done' })`, affiche le compteur Done, un chevron et, vertical, le libellé. Les ids de colonnes de `kanbanCollision` sont inchangés. Pendant un drag dont la source figure dans `validDropSources` de Done, le rail reprend le style violet des colonnes valides et affiche le libellé Done ; sa largeur ne change pas.

*Alternative écartée : dépli au survol pendant le drag.* Déplier le slot en plein drag ferait varier la largeur des autres colonnes (voire apparaître le scroll) et déplacerait les droppables sous le pointeur ; la surbrillance donne la même information sans reflow. C'est un écart assumé par rapport à l'idée initiale « s'ouvre au survol ».

### 4. DetailPanel : largeur pilotée par le layout, overlay absolu
Le panel reçoit sa largeur de `computeKanbanLayout` (style inline, remplaçant `w-[420px]`). En mode `overlay`, il est positionné en `absolute right-0 inset-y-0` dans le conteneur de la ligne (colonnes + panel), avec ombre portée et `z-index` sous les dialogs. Le conteneur des colonnes prend alors toute la largeur. Pendant un drag (`activeDragId !== null`), le panel overlay est masqué (`opacity-0 pointer-events-none`, transition courte) pour libérer les colonnes de droite, puis réapparaît. Pas de backdrop : les colonnes restent interactives hors de la zone du panel, comme en mode inline.

### 5. Tailles resserrées
`min-w-[220px]` → `min-w-[190px]` dans `KanbanColumn` et dans le slot Done/Archived ; `gap-3` → `gap-2` ; conteneur `p-4` → `p-2`. Les constantes de `kanbanLayout.ts` sont la source unique de ces valeurs (les classes Tailwind restent littérales, un test garde la cohérence avec les constantes).

## Risks / Trade-offs

- [Cartes plus serrées à 190px : badge worker, bouton d'arrêt, tags peuvent déborder] → vérifier visuellement `ChangeCard` (états worker, ghost, ff) à 190px ; ajuster `flex-wrap`/`min-w-0` sans créer de variante compacte.
- [Boucle de mesure : un changement de mode modifie la largeur du conteneur mesuré] → mesurer le conteneur de la ligne (colonnes + panel), dont la largeur ne dépend pas du mode choisi, pas celui des colonnes.
- [Le repli automatique du slot en plein usage peut surprendre] → compteur Done visible dans le rail, chevron pour surcharger, retour auto à la fermeture du panel.
- [Panel overlay qui s'efface pendant un drag peut sembler clignotant] → transition courte ; il ne s'efface que pendant le drag.
- [`ResizeObserver` absent en jsdom] → fonction pure testée directement ; les tests de page mockent `ResizeObserver` avec une largeur injectée.
