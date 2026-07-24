## Context

Actuellement, le panneau d'exploration est intégré en bas de la page Kanban et sa hauteur est limitée à 70% de l'écran par glisser-déposer (drag-to-resize). Lorsque l'agent génère des réponses denses ou du code, la conversation devient difficile à lire et l'utilisateur doit scroller continuellement.

## Goals / Non-Goals

**Goals:**
- Offrir une fonctionnalité de maximisation/minimisation rapide en un seul clic via un bouton dans le header.
- Assurer un rendu plein écran propre et fluide qui occupe tout l'espace disponible sous la barre de navigation.
- Augmenter la limite maximale du drag-to-resize manuel de 70% à 90% pour plus de flexibilité.
- Désactiver le redimensionnement par glisser-déposer lorsque le panneau est maximisé.

**Non-Goals:**
- Modifier l'API backend ou le protocole WebSocket d'exploration.
- Permettre le détachement du panneau dans une fenêtre flottante séparée.

## Decisions

### 1. Gestion de l'état maximisé au niveau de `KanbanPage.tsx`
- **Choix** : Ajouter un état `panelMaximized: boolean` dans `KanbanPage.tsx` et le transmettre aux sous-composants.
- **Raison** : `KanbanPage` gère déjà l'affichage conditionnel des deux panneaux (named et anonymous) et de la partie supérieure (colonnes Kanban + barre de recherche). En masquant la partie supérieure (`search bar` et `Kanban columns`) lorsque `panelMaximized` est vrai, et en forçant la hauteur du bottom panel à `100%`, on obtient un comportement plein écran natif et fluide grâce à Flexbox CSS, sans aucune superposition absolue complexe ni hacks de positionnement.

### 2. Boutons de contrôle dans le Header de `ExplorePanel` & `ExploreAnonymousPanel`
- **Choix** : Ajouter un bouton de bascule avec l'icône `Maximize2` / `Minimize2` de `lucide-react` juste avant le bouton de fermeture `X`.
- **Raison** : C'est l'emplacement standard pour les contrôles de fenêtre, ce qui rend la fonctionnalité immédiatement découvrable et intuitive pour l'utilisateur.

### 3. Désactivation du Drag-to-Resize en mode Maximisé
- **Choix** : Lorsque `isMaximized` est vrai, l'événement `onMouseDown` du drag handle est désactivé, sa classe de curseur passe de `cursor-row-resize` à `cursor-default`, et la couleur de survol n'est pas modifiée.
- **Raison** : Éviter les comportements incohérents de redimensionnement manuel alors que le panneau est maximisé.

## Risks / Trade-offs

- **[Risque] Perte de contexte visuel sur les colonnes Kanban en mode maximisé** → *Atténuation* : Le bouton de minimisation permet de restaurer instantanément la vue Kanban d'un simple clic pour suivre le mouvement des cartes.
- **[Risque] Conflit de types TS sur la propriété `height`** → *Atténuation* : Mettre à jour les interfaces de `ExploreBottomPanel` et `ExploreAnonymousBottomPanel` pour accepter `height: number | string`, de sorte que `100%` puisse être passé sans erreur de type.
