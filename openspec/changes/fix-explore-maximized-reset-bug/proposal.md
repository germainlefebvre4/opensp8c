## Why

Lorsqu'un utilisateur se trouve dans le volet d'exploration anonyme en mode maximisé (`panelMaximized === true`) et clique sur "Créer le change" ou "Abandonner l'exploration", le volet se ferme correctement (`anonymousExploreOpen` passe à `false`), mais l'état `panelMaximized` reste à `true`. Comme l'affichage du tableau Kanban est masqué lorsque `panelMaximized` est actif, l'intégralité du contenu central du Kanban reste invisible (page blanche sous la barre de navigation supérieure). 

## What Changes

- **Réinitialisation de l'état maximisé lors de la promotion** : La fonction `handlePromoteConfirm` dans `KanbanPage.tsx` réinitialisera explicitement `panelMaximized` à `false` lors de la transition réussie.
- **Réinitialisation de l'état maximisé lors de l'abandon** : La fonction `handleDeleteGhostById` dans `KanbanPage.tsx` réinitialisera également `panelMaximized` à `false` lors de la suppression d'une exploration fantôme.
- **Alignement des spécifications** : Mise à jour des scénarios de la spécification de promotion pour refléter ce comportement d'UI propre.

## Capabilities

### New Capabilities
- Aucun nouveau comportement fonctionnel n'est introduit.

### Modified Capabilities
- `exploration-promote-to-change` : Précise que lors de la validation ou de la suppression d'une exploration, l'état d'UI maximisé du volet est réinitialisé et le tableau Kanban redevient visible.

## Impact

- **Frontend** : Modification de `frontend/src/pages/KanbanPage.tsx`.
- **Spécifications** : Ajout d'une delta spec pour `exploration-promote-to-change`.
