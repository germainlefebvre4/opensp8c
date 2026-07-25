## Context

La page des spécifications (`SpecsPage.tsx`) gère son état sélectionné localement sans tenir compte de l'URL (`searchParams`). Cela empêche l'initialisation automatique ou la mise à jour de la spécification affichée lors d'une redirection ou d'un partage de lien avec le paramètre de requête `selected`.

## Goals / Non-Goals

**Goals:**
- Prendre en compte le paramètre de requête `selected` dans l'URL pour initialiser la spécification sélectionnée sur `SpecsPage`.
- Mettre à jour l'URL lorsque l'utilisateur change de spécification manuellement.
- Permettre à l'historique de navigation de fonctionner correctement (boutons Suivant/Précédent du navigateur).

**Non-Goals:**
- Modifier le format des URLs de redirection générées par d'autres composants.
- Modifier l'API de récupération des spécifications.

## Decisions

### Utiliser `useEffect` pour synchroniser le query parameter avec l'état local

- **Option A (State de vérité dans l'URL) :** Supprimer l'état local `selectedSpec` et lire directement de l'URL à chaque render via `searchParams.get('selected')`.
  * *Avantage :* Source de vérité unique.
  * *Inconvénient :* Peut causer des sauts de mise en page légers ou des délais le temps que `setSearchParams` se propage et déclenche un ré-affichage, et interfère potentiellement avec d'autres états locaux comme `isEditing`.
- **Option B (State local synchronisé) :** Conserver `selectedSpec` comme état local de type `useState`, initialisé par le query param de l'URL, et synchronisé via `useEffect` si le query param change extérieurement (par exemple en cliquant sur "Voir l'historique" puis sur "Retour" dans le navigateur).
  * *Avantage :* Plus fluide, isole les états d'édition locale et s'intègre parfaitement avec les composants enfants existants.
  * *Décision :* **Option B** pour minimiser les effets secondaires et conserver une réactivité maximale de l'interface utilisateur.

## Risks / Trade-offs

- **Risk:** Accumulation de requêtes de recherche dans l'historique de retour si chaque sélection manuelle crée une nouvelle entrée de navigation.
  - *Mitigation:* Utiliser `{ replace: true }` dans l'appel `setSearchParams` lors de la sélection d'une spécification dans la barre latérale.
