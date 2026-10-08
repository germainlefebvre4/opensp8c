# Tasks

## 1. Backend : couche « live » partagée

- [x] 1.1 Extraire de `KanbanHandler.ListChanges` (`backend/internal/api/handlers/kanban.go`) la couche d'enrichissement (workers, progression worktree/branche, état de vérification) en une fonction réutilisable, sans changer le comportement ; vérifier que les tests existants de `kanban_test.go` passent inchangés (`go test ./internal/api/handlers/...`)
- [x] 1.2 Faire utiliser cette fonction par `WorkspaceHandler.List` pour calculer `task_counts` ; ajouter un test dans `workspace_test.go` : un change `todo` tenu par un worker avec worktree avancée est compté `in-progress` dans `task_counts`, comme dans le Kanban

## 2. Backend : champ `attention`

- [x] 2.1 Ajouter les types `Attention` et `Signal` (`kind`, `reason`) et le champ `attention` (toujours une liste, jamais `null`) à `workspace.Workspace` ; vérifier par un test que la réponse JSON d'un workspace vide contient `"attention": []`
- [x] 2.2 Implémenter la fonction pure `ComputeAttention` : signaux `review`, `paused` (raison = `BlockedReason`), `hitl` (un par tâche non cochée marquée, texte sans marqueur), `verify-failed` ; aucun signal pour `queued`/`running`/`waiting` ; tests table-driven couvrant chaque scénario de `workspace-attention-signals` (cumul de signaux, tâche humaine cochée, vérification en cours)
- [x] 2.3 Implémenter le tri (priorité `paused` > `verify-failed` > `hitl` > `review`, puis nom, signaux du change dans le même ordre) ; vérifier avec un test d'ordre sur un jeu de changes mélangés
- [x] 2.4 Résoudre le `tasks.md` effectif pour les changes ni `done` ni `to-explore` (worktree, sinon branche `feature/<change>`, sinon dépôt) et brancher `ComputeAttention` dans `WorkspaceHandler.List` ; vérifier par un test d'intégration du handler (workspace temporaire avec `tasks.md` contenant une tâche `<!-- human review required -->`) que `attention` la contient avec le texte sans marqueur
- [x] 2.5 Vérifier que `GET /api/workspaces` reste tolérant : registre de pool absent ou `openspec/changes/` inexistant donnent `attention: []` sans erreur (tests unitaires du handler)

## 3. Frontend : fondations partagées

- [x] 3.1 Créer `frontend/src/lib/statusColors.ts` (`STATUS_ORDER` des 7 statuts, `dot` et `badge` par statut, couleurs actuelles de `KanbanColumn`) et le faire utiliser par `KanbanColumn.tsx` ; test unitaire : chaque statut a une couleur, `ready` est indigo et `verifying` teal ; `KanbanColumn.test.tsx` passe inchangé
- [x] 3.2 Étendre le type `Workspace` de `useWorkspaces.ts` (`attention`) et créer les fonctions pures `segments(task_counts)` (ordre Kanban, proportions, statuts à 0 omis) et `capSignals` (plafond de 3 `hitl` + « +N autres ») avec leurs tests unitaires
- [x] 3.3 Ajouter les clés i18n en/fr du namespace `workspace` (types de signaux, libellés des statuts de la barre, menu d'actions, confirmation de suppression, « +N autres », infobulles) et vérifier que le test i18n existant des locales passe (`npm test` sur les tests `*.i18n.test.ts`)

## 4. Frontend : ligne projet à deux niveaux

- [x] 4.1 Créer le composant `StatusBar` (barre segmentée, `title` « libellé : n » par segment, total à l'extrémité, couleurs de `statusColors.ts`) et vérifier par test de rendu : sept statuts → sept segments dans l'ordre, workspace vide → barre vide avec total 0
- [x] 4.2 Refondre la ligne de `WorkspaceSidebar` en `WorkspaceItem` : ligne 1 (chevron conditionnel, nom tronqué, compteur de changes en attente, masqué à 0) et ligne 2 (`StatusBar`) ; ligne entière cliquable avec `role="button"`, `tabIndex`, Entrée/Espace et `aria-current` sur le workspace actif ; tests : clic hors contrôles sélectionne, clavier sélectionne, `aria-current` présent
- [x] 4.3 Supprimer `BADGE_COLORS` et `BADGE_ORDER` de `WorkspaceSidebar.tsx` et mettre à jour `WorkspaceSidebar.test.tsx` (badges devenus segments) ; `npm test` vert

## 5. Frontend : vue dépliée et ouverture d'un change

- [x] 5.1 Créer `AttentionList` : par change une seule ligne de titre puis une ligne par signal (type + raison tronquée avec `title` complet), via `capSignals` ; tests : change à trois signaux affiche un seul nom et trois lignes, cinq `hitl` donnent trois lignes + « +2 autres »
- [x] 5.2 Gérer l'état déplié (`Set` d'ids en mémoire de session, repliés par défaut, indépendants, non persisté) ; un projet sans signal n'a pas de chevron ; tests : dépliage de A n'affecte pas B, absence de chevron sans signal
- [x] 5.3 Ouvrir un change depuis la sidebar : navigation vers `/?workspace=<id>&change=<name>` ; `KanbanPage` ouvre le DetailPanel si le change existe, retire le paramètre sinon, et le retire à la fermeture du panel ; tests de page (change existant, change absent sans erreur, fermeture)

## 6. Frontend : actions sûres, largeur et rail

- [x] 6.1 Ajouter la dépendance `@radix-ui/react-dropdown-menu` (`npm install`, lockfile mis à jour) et créer le menu `…` de chaque ligne (toujours visible, atteignable au clavier, entrée « Retirer du suivi ») ; test : le menu s'ouvre au clavier
- [x] 6.2 Brancher la suppression sur `ConfirmDialog` (titre/corps nommant le projet et précisant que le dossier n'est pas supprimé) : confirmer appelle `useRemoveWorkspace`, annuler ne fait rien ; tests des deux chemins
- [x] 6.3 Afficher l'erreur d'ajout (message du backend : sans `openspec/`, doublon, chemin invalide) dans le formulaire, qui reste ouvert avec la valeur saisie, et l'effacer à la modification du champ ou à l'annulation ; tests pour les trois cas
- [x] 6.4 Passer la sidebar ouverte à `w-72` et l'état replié à `w-10` avec `SidebarRail` (pastille d'initiales, `title` = nom, point d'attention, clic = sélection) ; tests de rendu des deux états et du point d'attention
- [x] 6.5 Mettre à jour les specs principales dérivées par le change si nécessaire et lancer `openspec validate improve-workspace-sidebar --strict` ; vérifier que la commande réussit

## 7. Intégration

- [x] 7.1 Lancer `go test ./...` côté backend et `npm test` + `npm run build` côté frontend ; vérifier que tout est vert
- [x] 7.2 Parcours manuel de la sidebar avec un workspace de test contenant un change `to-review`, un worker en pause avec raison, une tâche HITL et une vérification échouée : compteur, barre à 7 segments, vue dépliée, clic vers le DetailPanel, menu `…` + confirmation, rail replié <!-- human review required -->
- [x] 7.3 Vérifier visuellement à 1920px avec le DetailPanel ouvert que le Kanban reste tout déplié (sans repli de Done ni scrollbar horizontale) avec la sidebar à `w-72` <!-- human review required -->
