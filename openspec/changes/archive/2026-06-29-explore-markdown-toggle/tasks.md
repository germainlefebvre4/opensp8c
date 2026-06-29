## 1. Hook localStorage

- [x] 1.1 Créer `frontend/src/hooks/useExploreViewMode.ts` avec lecture/écriture localStorage (clé `explore-view-mode`, défaut `raw`, try/catch sur accès storage)

## 2. ExploreAnonymousPanel

- [x] 2.1 Importer `useExploreViewMode`, `ReactMarkdown`, et les icônes `Code` / `Eye` depuis lucide-react
- [x] 2.2 Ajouter le toggle groupé dans le header (entre statut connexion et bouton X)
- [x] 2.3 Rendre les messages assistant conditionnellement (raw: `whitespace-pre-wrap` / rendered: `ReactMarkdown` + classes prose)
- [x] 2.4 Laisser les messages utilisateur toujours en raw

## 3. ExplorePanel

- [x] 3.1 Importer `useExploreViewMode`, `ReactMarkdown`, et les icônes `Code` / `Eye`
- [x] 3.2 Ajouter le toggle dans le header (même UI que ExploreAnonymousPanel)
- [x] 3.3 Appliquer le rendu conditionnel identique sur les messages assistant

## 4. Vérification

- [x] 4.1 Vérifier que le toggle persiste après rechargement de page
- [x] 4.2 Vérifier que les messages utilisateur restent en raw en mode rendered
- [x] 4.3 Vérifier que les deux panels partagent bien la même préférence localStorage
