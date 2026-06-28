## 1. Layout KanbanPage

- [x] 1.1 Refactoriser `KanbanPage.tsx` : passer à un layout flex horizontal `flex-row` englobant colonnes + slot panel
- [x] 1.2 Zone colonnes : `flex-1 overflow-x-auto min-h-0 flex gap-3`
- [x] 1.3 Slot panel conditionnel : `w-[420px] shrink-0 border-l border-slate-200 flex flex-col overflow-hidden` — visible uniquement si `activePanel !== null`

## 2. DetailPanel — suppression overlay

- [x] 2.1 Supprimer `position: fixed`, `right`, `top`, `bottom`, `zIndex`, `boxShadow` du container principal de `DetailPanel.tsx`
- [x] 2.2 Passer le container à `h-full flex flex-col bg-white` pour remplir son slot parent
- [x] 2.3 Remplacer les inline styles restants du header par des classes Tailwind

## 3. ExplorePanel — suppression overlay

- [x] 3.1 Supprimer `position: fixed`, `right`, `top`, `bottom`, `zIndex`, `boxShadow` du container principal de `ExplorePanel.tsx`
- [x] 3.2 Passer le container à `h-full flex flex-col bg-white` pour remplir son slot parent
- [x] 3.3 Remplacer les inline styles restants du header par des classes Tailwind

## 4. DetailPanel — toggle Raw/Rendu

- [x] 4.1 Ajouter l'état `viewMode: 'raw' | 'rendered'` dans `DetailPanel.tsx` (défaut : `'rendered'`)
- [x] 4.2 Afficher le toggle (icônes `Code` / `Eye` de lucide-react) dans la barre de tabs uniquement quand `activeTab === 'proposal' || activeTab === 'design'`
- [x] 4.3 Onglet Proposal : afficher `<ReactMarkdown className="prose prose-slate prose-sm">` si `viewMode === 'rendered'`, sinon `<pre>` brut
- [x] 4.4 Onglet Design : même logique que Proposal
- [x] 4.5 Styliser le toggle : deux boutons compacts avec state actif mis en évidence (background + couleur), classes Tailwind

## 5. Vérification

- [x] 5.1 Vérifier que les colonnes Kanban restent visibles et interactibles quand DetailPanel ou ExplorePanel est ouvert
- [x] 5.2 Vérifier le scroll horizontal des colonnes quand l'espace est insuffisant (fenêtre étroite ou mode page)
- [x] 5.3 Vérifier le toggle Raw/Rendu sur Proposal et Design : changement d'onglet conserve le mode
- [x] 5.4 Vérifier que le toggle est absent sur l'onglet Tâches
- [x] 5.5 Vérifier `tsc --noEmit` : zéro erreur TypeScript
