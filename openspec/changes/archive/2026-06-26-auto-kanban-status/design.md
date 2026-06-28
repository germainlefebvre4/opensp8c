## Context

Le Kanban board de l'app stocke actuellement la colonne d'un changement dans le champ `kanban_status` de `.openspec.yaml`. Ce champ est mis à jour via un endpoint `PUT` déclenché par le drag & drop ou un bouton dans la carte. Mais l'app est en réalité read-only : les vrais changements d'état (exploration, implémentation, completion) se font en dehors via le terminal et le CLI OpenSpec. Le `kanban_status` ne se synchronise donc jamais et tous les changements restent en "To Explore" indéfiniment.

## Goals / Non-Goals

**Goals:**
- Dériver la colonne Kanban automatiquement depuis la progression en tasks
- Supprimer toute interaction de mutation de statut dans l'app (drag & drop, bouton)
- Rendre le board cohérent avec la réalité sans action manuelle

**Non-Goals:**
- Modifier le format de `tasks.md` ou les règles de parsing des tasks
- Toucher à la colonne "To Explore" dans l'ExplorePanel (chat)
- Gérer un état intermédiaire custom via `.openspec.yaml`

## Decisions

### Dériver le statut depuis tasks_done / tasks_total

La fonction `deriveStatus` remplace la lecture de `kanban_status` dans `loadChange` :

```
tasks_total == 0              → "to-explore"
tasks_done == 0, total > 0    → "todo"
0 < tasks_done < tasks_total  → "in-progress"
tasks_done == tasks_total > 0 → "done"
```

**Pourquoi pas le champ `status` du CLI openspec ?** Le CLI est un outil utilisateur, pas une librairie. Le backend possède déjà `parseTaskProgress` qui lit `tasks.md` directement en Go — zéro dépendance externe, synchrone, déterministe.

### Suppression de l'endpoint PUT /status

L'endpoint n'a plus de sémantique utile puisque le statut est calculé. Le supprimer évite une surface d'API trompeuse. Le champ `kanban_status` dans `.openspec.yaml` est ignoré à la lecture mais peut rester dans les fichiers existants sans effet.

### Suppression du drag & drop frontend

Le drag & drop appelait `PUT /status`. Sans endpoint, il devient inopérant. On le supprime plutôt que de le laisser silencieux — un geste UI sans effet visible est une source de confusion.

## Risks / Trade-offs

- **Un change avec 0 tasks oscille en "To Explore"** pendant la phase d'écriture des artifacts via `opsx:new` + `opsx:continue`. C'est le comportement voulu : tant que les tasks ne sont pas définies, le change est encore en exploration.
- **Perte du contrôle manuel** → pas de moyen de forcer un changement de colonne depuis l'app. Acceptable car c'est précisément le principe : l'app reflète, ne pilote pas.
- **`kanban_status` dans les `.openspec.yaml` existants** → ignoré silencieusement. Aucune migration nécessaire.
