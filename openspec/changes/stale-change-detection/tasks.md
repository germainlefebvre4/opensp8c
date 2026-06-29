## 1. Backend — Config & Struct

- [ ] 1.1 Ajouter `stale_threshold_days` dans la struct `openspecConfig` de `change.go` (avec valeur par défaut 7)
- [ ] 1.2 Ajouter une fonction `readOpenspecConfig(workspacePath)` qui lit `openspec/config.yaml` et retourne le threshold
- [ ] 1.3 Ajouter les champs `DaysSinceActivity int` et `IsStale bool` à la struct `Change`

## 2. Backend — Calcul dans loadChange

- [ ] 2.1 Dans `loadChange()`, appeler `os.Stat(tasksPath).ModTime()` pour obtenir la date de dernière modification
- [ ] 2.2 Calculer `DaysSinceActivity` en jours entiers depuis `ModTime` (`-1` si `tasks.md` absent)
- [ ] 2.3 Passer le threshold en paramètre à `loadChange()` et calculer `IsStale` selon le statut et le seuil

## 3. Backend — ListChanges

- [ ] 3.1 Dans `ListChanges()`, appeler `readOpenspecConfig(workspacePath)` pour lire le threshold
- [ ] 3.2 Passer le threshold à chaque appel de `loadChange()`

## 4. Frontend — Type & Hook

- [ ] 4.1 Ajouter `days_since_activity: number` et `is_stale: boolean` à l'interface `Change` dans `useChanges.ts`

## 5. Frontend — ChangeCard Badge

- [ ] 5.1 Dans `ChangeCard.tsx`, afficher `⚠ Nj` à droite du compteur de tâches quand `change.is_stale === true`
- [ ] 5.2 Styler le badge en ambre (ex: `text-amber-500`) discret, inline sur la même ligne que `X/Y tasks`
