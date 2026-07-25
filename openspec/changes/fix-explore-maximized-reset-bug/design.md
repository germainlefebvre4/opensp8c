## Context

Lorsqu'un utilisateur maximise le volet d'exploration anonyme (`panelMaximized === true`) et clique sur "Créer le change" ou "Abandonner l'exploration", l'application ferme le volet (`anonymousExploreOpen = false`). Cependant, l'état `panelMaximized` reste bloqué à `true`. Comme l'affichage des colonnes Kanban de la page principale est masqué lorsque `panelMaximized` est actif, le tableau Kanban reste invisible, laissant l'utilisateur face à une page blanche.

## Goals / Non-Goals

**Goals:**
- Réinitialiser l'état d'UI `panelMaximized` à `false` lors de la promotion réussie d'une exploration fantôme en un change réel.
- Réinitialiser l'état d'UI `panelMaximized` à `false` lors de la suppression/abandon réussi d'une exploration fantôme.
- Garantir que le tableau Kanban réapparaît immédiatement et de manière fluide.

**Non-Goals:**
- Modifier le comportement de l'API backend ou du protocole de communication (SSE/WS).
- Modifier la gestion d'état de maximisation du volet de détails de change standard.

## Decisions

### Décision 1 : Réinitialiser `panelMaximized` dans `handlePromoteConfirm`
Dans `KanbanPage.tsx`, la fonction `handlePromoteConfirm` est appelée après confirmation de la promotion. Nous allons y insérer l'appel `setPanelMaximized(false)` juste à côté de `setAnonymousExploreOpen(false)`.

*Alternative considérée :* Réinitialiser l'état dans un effet de bord réactif (`useEffect`) basé sur `anonymousExploreOpen`. *Raison du rejet :* Moins explicite, peut provoquer des scintillements d'UI inutiles si d'autres parties du code ouvrent/ferment le volet sans vouloir altérer l'état de maximisation.

### Décision 2 : Réinitialiser `panelMaximized` dans `handleDeleteGhostById`
Dans `KanbanPage.tsx`, la fonction `handleDeleteGhostById` est appelée après confirmation de la suppression/abandon. Nous allons y insérer l'appel `setPanelMaximized(false)` également.

## Risks / Trade-offs

- Aucun risque identifié. Cette modification de l'état d'UI est locale, asynchrone et sans impact sur l'état persistant ou sur le serveur.
