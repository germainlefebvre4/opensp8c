## 1. Implementation

- [x] 1.1 Import `useEffect` dans `SpecsPage.tsx` et récupérer le paramètre de requête `selected` depuis `searchParams`
- [x] 1.2 Synchroniser l'état local `selectedSpec` avec ce paramètre de requête en utilisant un hook `useEffect`
- [x] 1.3 Mettre à jour `handleSelectSpec` pour synchroniser le paramètre `selected` de l'URL lors du clic sur une spec dans la barre latérale

## 2. Validation

- [x] 2.1 Vérifier la redirection depuis l'onglet Matrice de la Timeline vers la page des spécifications
- [x] 2.2 Vérifier que la sélection manuelle dans la barre latérale met bien à jour l'URL sans polluer l'historique
- [x] 2.3 Vérifier que les boutons de retour/avance du navigateur fonctionnent comme prévu
