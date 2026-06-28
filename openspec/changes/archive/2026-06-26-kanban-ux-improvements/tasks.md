## 1. Backend — Endpoint détail du change

- [x] 1.1 Ajouter `ChangeDetail` struct dans `internal/openspec/change.go` (champs : `tasks []Task`, `artifacts Artifacts`)
- [x] 1.2 Ajouter `Task` struct `{ Text string, Done bool }` et `Artifacts` struct `{ Proposal string, Design string }`
- [x] 1.3 Implémenter `GetChangeDetail(workspacePath, name string)` qui lit tasks.md (liste détaillée) et le contenu de proposal.md / design.md
- [x] 1.4 Ajouter handler `GetChange` dans `internal/api/handlers/kanban.go`
- [x] 1.5 Enregistrer la route `GET /api/workspaces/{id}/changes/{name}` dans `internal/api/router.go`

## 2. Frontend — Hook et types

- [x] 2.1 Créer `frontend/src/hooks/useChangeDetail.ts` avec type `ChangeDetail` (tâches + artifacts) et appel `GET /api/workspaces/{id}/changes/{name}`

## 3. Frontend — Composant DetailPanel

- [x] 3.1 Créer `frontend/src/components/DetailPanel.tsx` — structure panneau fixe droite (même pattern qu'ExplorePanel)
- [x] 3.2 Afficher l'en-tête : nom du change, statut courant, bouton fermeture
- [x] 3.3 Afficher la liste des tâches avec état coché/décoché (lecture seule)
- [x] 3.4 Afficher les sections Proposal et Design (contenu markdown brut dans un `<pre>` ou texte simple)
- [x] 3.5 Ajouter les boutons de transition de statut (selon `kanban_status` courant) avec appel `onUpdateStatus`
- [x] 3.6 Ajouter le bouton Archiver (uniquement si statut `done`) avec gestion erreur

## 4. Frontend — Carte cliquable et épurée

- [x] 4.1 Dans `ChangeCard.tsx`, supprimer les boutons inline (`→ To Do`, `Explorer`, `Archiver`)
- [x] 4.2 Ajouter `onClick` sur le div principal de la carte, `cursor: 'pointer'`
- [x] 4.3 Mettre à jour l'interface `Props` de `ChangeCard` : remplacer `onExplore` par `onOpen: (name: string) => void`
- [x] 4.4 Mettre à jour `KanbanColumn.tsx` : passer `onOpen` à toutes les cartes (toutes colonnes)

## 5. Frontend — Gestion du panneau actif dans KanbanPage

- [x] 5.1 Remplacer `exploreChange: string | null` par `activePanel: { type: 'explore' | 'detail'; name: string } | null` dans `KanbanPage.tsx`
- [x] 5.2 Implémenter `handleOpen(name, type)` qui met à jour `activePanel`
- [x] 5.3 Conditionner le rendu : `activePanel.type === 'explore'` → `<ExplorePanel>`, `activePanel.type === 'detail'` → `<DetailPanel>`
- [x] 5.4 Passer `handleOpen` à `KanbanColumn` avec le type correct selon la colonne (`explore` pour `to-explore`, `detail` pour les autres)

## 6. Frontend — Corrections layout

- [x] 6.1 Dans `KanbanPage.tsx`, changer `alignItems: 'flex-start'` en `alignItems: 'stretch'` sur le div flex-row des colonnes
- [x] 6.2 Ajouter `width: '100%'` sur le div flex-row des colonnes
- [x] 6.3 Dans `KanbanColumn.tsx`, remplacer `minHeight: 200` par rien (la hauteur est gérée par le parent stretch)

## 7. Corrections vérification

- [x] 7.1 Dans `GetChangeDetail`, ajouter `os.Stat` sur le répertoire du change avant d'appeler `loadChange` — retourner une erreur `os.ErrNotExist` si absent (fix 404)
- [x] 7.2 Dans `useChanges.ts`, invalider aussi `['change-detail', workspaceId, name]` dans `onSuccess` de `useUpdateStatus` (fix stale status dans DetailPanel)
- [x] 7.3 Dans `readFileContent`, retourner `""` quand le fichier est absent — comportement identique, déjà correct côté UI (la spec attendait `null` mais l'UI traite `""` comme falsy ; documenter le choix)
