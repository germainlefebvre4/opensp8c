## Context

La sidebar liste les workspaces en itérant `h.cfg.Workspaces` dans l'ordre du slice config.yaml. Il n'existe pas de champ d'ordre séparé. `Config.Save()` existe déjà et sérialise le slice entier en YAML. `@dnd-kit/core` est déjà installé en frontend ; `@dnd-kit/sortable` est absent.

## Goals / Non-Goals

**Goals:**
- Permettre le drag-and-drop des workspaces dans la sidebar
- Persister l'ordre côté serveur dans `config.yaml` (ordre du slice)
- Exposer un endpoint `PATCH /api/workspaces/order`

**Non-Goals:**
- Ordre par utilisateur (scope global, utilisateur unique)
- Annulation / historique des réordonnements
- Tri automatique (alphabétique, par activité…)

## Decisions

### D1 — Stocker l'ordre comme position dans le slice config.yaml

**Choix** : réordonner `Config.Workspaces []WorkspaceConfig` directement ; l'ordre du YAML est l'ordre affiché.

**Alternatives considérées** :
- Champ `order int` sur chaque `WorkspaceConfig` : plus de complexité pour un gain nul avec un seul utilisateur.
- Table/fichier d'ordre séparé : sur-ingénierie pour un tableau de quelques éléments.

**Rationale** : `Config.Save()` existe déjà, pas de migration, minimal.

### D2 — Endpoint `PATCH /api/workspaces/order` avec liste d'IDs

**Payload** : `{ "order": ["id1", "id2", "id3"] }`  
**Comportement** : valide que les IDs correspondent exactement aux workspaces connus, réordonne le slice, sauvegarde.

**Alternative** : envoyer l'objet complet — rejeté car le payload d'IDs est plus léger et moins error-prone.

### D3 — Optimistic update côté frontend

Le state local est mis à jour immédiatement au drop ; la mutation React Query est lancée en arrière-plan. En cas d'erreur serveur, `onError` invalide la query pour forcer un refetch et restaurer l'état réel.

**Alternative** : attendre la réponse serveur avant d'afficher — rejeté car latence perçue trop élevée.

### D4 — `@dnd-kit/sortable` pour la liste

`@dnd-kit/sortable` fournit `SortableContext`, `useSortable`, et `arrayMove` — API de haut niveau sur `@dnd-kit/core` déjà présent. Pas de nouveau paradigme à introduire.

## Risks / Trade-offs

- **Race condition sur refetch automatique (15 s)** → Le refetch React Query peut écraser l'ordre optimiste si la mutation est lente. Mitigation : désactiver `refetchInterval` pendant le drag, ou utiliser `updater` dans `onMutate` pour synchroniser le cache.
- **IDs inconnus dans le payload** → Le handler rejette avec 400 si les IDs ne correspondent pas exactement à la liste connue. Cela protège contre les états désynchronisés.
