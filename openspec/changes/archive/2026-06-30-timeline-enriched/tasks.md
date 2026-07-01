## 1. Fusion client-side — données delta specs dans la Timeline

- [x] 1.1 Importer `useSpecsOverview` dans `frontend/src/pages/TimelinePage.tsx`
- [x] 1.2 Calculer `changeToSpecs: Record<string, string[]>` (inverse de l'overview) via `useMemo`
- [x] 1.3 Calculer `knownSpecNames: Set<string>` depuis `overview.specs` pour la séparation spec/tag
- [x] 1.4 Pour chaque change, dériver `specChips` (delta specs) et `extraComps` (tags non couverts par une spec formelle)

## 2. Mode Changes — enrichissement des cards

- [x] 2.1 Créer `frontend/src/components/TimelineChangeCard.tsx` : affiche nom + type badge + complexité + date + statut dot
- [x] 2.2 Ajouter les spec chips cliquables dans `TimelineChangeCard` : clic navigue vers `/specs?workspace=<id>&selected=<spec-name>`
- [x] 2.3 Ajouter les extra component chips (style atténué, filtre seulement) dans `TimelineChangeCard`
- [x] 2.4 Remplacer les cards inline de `TimelinePage` par `TimelineChangeCard`

## 3. Mode Changes — heatmap fusionnée

- [x] 3.1 Remplacer le calcul de la heatmap (actuellement depuis `tags.components`) par un calcul depuis `overview.specs` (fréquence par spec name)
- [x] 3.2 Mettre à jour le label de la section heatmap : "Composants fréquents" → "Specs fréquentes"
- [x] 3.3 Étendre le système de filtres pour accepter les spec names (en plus des tag types et composants)
- [x] 3.4 Mettre à jour la logique de filtrage de la timeline pour inclure les delta specs (`changeToSpecs`)

## 4. Toggle Changes / Matrice

- [x] 4.1 Ajouter l'état `mode: 'changes' | 'matrice'` dans `TimelinePage`
- [x] 4.2 Ajouter le toggle [Changes | Matrice] dans le header de `TimelinePage`
- [x] 4.3 Lire le paramètre `?spec=` depuis `useSearchParams` ; si présent, initialiser `mode = 'matrice'` et `selectedSpec = value`

## 5. Mode Matrice — composant TimelineSpecMatrix

- [x] 5.1 Créer `frontend/src/components/TimelineSpecMatrix.tsx` : props `specs: SpecWithHistory[]`, `onSpecSelect: (name: string) => void`, `selectedSpec?: string | null`
- [x] 5.2 Calculer les colonnes de dates : extraire et trier les dates uniques depuis tous les `ChangeRef` de l'overview
- [x] 5.3 Rendre la grille : ligne par spec, colonne par date, intensité CSS en fonction du count (0 = vide, 1 = léger, 2 = moyen, 3+ = fort)
- [x] 5.4 Mettre en évidence la ligne de la spec sélectionnée
- [x] 5.5 Afficher un indicateur ⚠ sur les lignes de specs sans aucun change lié
- [x] 5.6 Afficher une section "Orphelins" en bas de la grille si `overview.orphans` est non vide

## 6. Mode Matrice — panel droit (spec history + DetailPanel)

- [x] 6.1 Ajouter les états `selectedSpec: string | null` et `selectedChange: string | null` dans `TimelinePage`
- [x] 6.2 En mode Matrice, afficher la grille à gauche et un panel droit si `selectedSpec !== null`
- [x] 6.3 Dans le panel droit, afficher `SpecHistoryView` filtré sur la spec sélectionnée (construire un `SpecOverview` mono-spec) avec `onChangeClick` → `setSelectedChange`
- [x] 6.4 Quand `selectedChange !== null`, remplacer le panel droit par `DetailPanel` (composant existant réutilisé tel quel)
- [x] 6.5 Fermeture du `DetailPanel` → revenir au panel de spec (`selectedChange = null`)
- [x] 6.6 Ajouter le lien "Voir la spec →" dans le header du panel de spec (navigue vers `/specs?workspace=<id>&selected=<name>`)

## 7. SpecsPage — simplification et lien sortant

- [x] 7.1 Retirer les imports `SpecHistoryView`, `useSpecsOverview`, et le type `Mode` de `frontend/src/pages/SpecsPage.tsx`
- [x] 7.2 Retirer les états `mode`, `setMode`, `historyDetailOpen`, `setHistoryDetailOpen`
- [x] 7.3 Retirer le toggle [Contenu | Historique] du JSX
- [x] 7.4 Retirer le bloc conditionnel `mode === 'history'` et ses composants
- [x] 7.5 Ajouter le lien "Voir l'historique →" dans le panneau de détail d'une spec sélectionnée (visible seulement quand `selectedSpec !== null`), naviguant vers `/timeline?workspace=<id>&spec=<name>`
