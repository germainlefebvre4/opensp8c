## Context

Le board Kanban affiche les changes OpenSpec sans aucune notion de "dernière activité". Un change peut rester `in-progress` ou `done` indéfiniment sans signal visuel. La struct `Change` (backend) contient uniquement `Created` (date de création depuis `.openspec.yaml`) — aucun champ de dernière modification.

L'app n'est pas encore corrélée à git : le signal d'activité doit être purement filesystem.

## Goals / Non-Goals

**Goals:**
- Calculer `days_since_activity` via le `mtime` de `tasks.md` de chaque change
- Exposer `days_since_activity` (int) et `is_stale` (bool) dans l'API `/changes`
- Rendre le seuil configurable dans `openspec/config.yaml` (défaut : 7 jours)
- Afficher un badge `⚠ Nj` inline dans `ChangeCard` pour les changes stale (`in-progress` et `done` non-archivé)

**Non-Goals:**
- Détection basée sur git log
- Événements watcher pro-actifs quand un change devient stale (recalculé à la demande)
- Configuration du seuil par change individuel
- Alertes ou notifications

## Decisions

### Signal : mtime de tasks.md

`os.Stat(tasksPath).ModTime()` dans `loadChange()`. Coût nul (pas de subprocess). Le fichier `tasks.md` est le seul fichier écrit lors d'interactions utilisateur dans le workflow normal (toggle de tâches). C'est un proxy fiable de l'activité dans l'outil.

Alternatif écarté : `git log` — ajoute un subprocess par change à chaque appel `/changes`, fragilise le système si git n'est pas disponible.

### Champs exposés : days_since_activity + is_stale

Deux champs distincts :
- `days_since_activity int` — pour afficher le nombre de jours dans le badge
- `is_stale bool` — calculé côté backend (qui connaît le seuil de config)

Le frontend ne connaît pas le seuil, donc il ne peut pas calculer `is_stale` lui-même. Retourner les deux évite de dupliquer la logique de seuil.

`days_since_activity = -1` si `tasks.md` n'existe pas (status `to-explore`).

### Seuil : openspec/config.yaml

```yaml
stale_threshold_days: 7   # valeur par défaut si absent : 7
```

Lu une fois dans `ListChanges()` avant l'itération sur les changes. Pas de reload à chaud — c'est une config de projet, pas un paramètre runtime.

### Statuts concernés

`is_stale = true` uniquement pour `in-progress` et `done` (non-archivé). Les statuts `to-explore` et `todo` ne déclenchent pas le badge (pas encore démarrés, l'inactivité est normale).

## Risks / Trade-offs

- **mtime pollué par des outils** → Si un script ou un formateur touche `tasks.md`, le compteur se remet à zéro. Risque faible dans ce contexte (fichier markdown, pas de post-processing automatique connu).
- **Calcul synchrone dans ListChanges()** → `os.Stat` par change est négligeable (< 1µs par appel filesystem). Pas de souci de performance même avec 50 changes.
- **Seuil non rechargeable à chaud** → Modifier `config.yaml` nécessite un redémarrage du backend. Acceptable pour une config de projet.
