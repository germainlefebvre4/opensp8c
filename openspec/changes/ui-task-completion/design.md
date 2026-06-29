## Context

Le backend Go expose une API REST lue par un frontend React. Les fichiers `tasks.md` des changes sont parsés à chaque GET (`parseTaskList()` dans `change.go`). Aucune écriture de fichier n'existe aujourd'hui côté backend, sauf l'archive qui délègue au CLI `openspec`. Le frontend lit les données via React Query et n'a aucune logique de write directe.

## Goals / Non-Goals

**Goals:**
- Permettre le toggle `[ ]` ↔ `[x]` d'une tâche via l'UI, sans quitter l'app
- Resynchroniser l'affichage depuis le fichier après chaque toggle (source de vérité = `tasks.md`)
- Toggle pessimiste : l'UI n'évolue qu'après confirmation du serveur

**Non-Goals:**
- Réordonnancement des tâches depuis l'UI
- Ajout / suppression de tâches depuis l'UI
- Gestion de conflits si le fichier est modifié simultanément par un éditeur externe

## Decisions

### 1. Endpoint `PATCH /api/workspaces/{id}/changes/{name}/tasks/{index}`

L'index est 0-based et correspond à la position dans la liste retournée par `parseTaskList()`. Pas de body requis — le toggle est implicite (si `[ ]`, passe à `[x]`, et inversement).

**Alternatives considérées :**
- `PUT` avec body `{done: bool}` : plus explicite mais inutile pour un toggle ; ajoute du payload sans valeur
- Identifier la tâche par son texte : fragile (doublons possibles, sensible à la casse)

### 2. Implémentation backend : réécriture ligne par ligne

La fonction `toggleTask(path string, index int)` :
1. Lit `tasks.md` ligne par ligne
2. Compte les lignes correspondant à des checkboxes (`- [ ]` ou `- [x]`)
3. À l'index cible, flip l'état
4. Réécrit le fichier en place

Pas de parsing markdown avancé — le format `- [ ] texte` est stable et défini par OpenSpec.

**Alternatives considérées :**
- Réécrire via le CLI `openspec` comme pour l'archive : ajoute une dépendance CLI, surcharge pour une opération simple
- Patch en mémoire avec regex : équivalent, moins lisible

### 3. Toggle pessimiste côté frontend

Le checkbox est disabled pendant la requête PATCH. On invalide `['changeDetail', workspaceId, changeName]` après succès pour forcer un re-fetch. Pas d'optimistic update.

**Rationale :** latence locale < 10ms, rollback inutile, implémentation 3x plus simple.

### 4. Pas de gestion de conflit

Si l'utilisateur édite `tasks.md` dans son éditeur pendant que l'app est ouverte, le re-fetch SSE (workspace events) rafraîchira l'UI. Le toggle PATCH écrasera l'état au moment de l'écriture — comportement last-write-wins acceptable pour un usage solo.

## Risks / Trade-offs

- **Index drift** : si `tasks.md` est modifié entre le GET et le PATCH, l'index peut pointer sur la mauvaise tâche → Mitigation : le re-fetch SSE réduit la fenêtre ; acceptable pour usage solo
- **Fichier malformé** : si une ligne checkbox est mal formatée, `toggleTask` peut ignorer silencieusement la ligne → Mitigation : le backend retourne une erreur 404 si l'index n'est pas trouvé
