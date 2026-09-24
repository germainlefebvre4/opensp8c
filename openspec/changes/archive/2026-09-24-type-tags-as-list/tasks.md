# Tasks

## 1. Script de migration des `.openspec.yaml` existants

- [x] 1.1 Créer `backend/cmd/migratetypetags/main.go` : parcourt `openspec/changes/*/.openspec.yaml` et `openspec/changes/archive/*/.openspec.yaml`, lit `tags.type` avec une struct locale (`Type string`, indépendante de la struct `Tags` de production), applique la table de conversion de `design.md` (D3 : `frontend`→`[frontend]`, `backend`→`[backend]`, `batch`→`[batch]`, `fullstack`→`[frontend, backend]`, `""`→`[]`), et réécrit le fichier avec `Type []string`. Vérifier avec `go build ./...` que le script compile.
- [x] 1.2 Rendre le script idempotent : un fichier dont `tags.type` est déjà une séquence YAML (et non un scalaire) n'est pas réécrit. Vérifier en exécutant le script deux fois de suite sur un même fichier déjà migré et en confirmant (`git diff`) qu'aucune modification n'est produite au second passage.
- [x] 1.3 Exécuter `go run ./cmd/migratetypetags` depuis `backend/` sur le repo, puis vérifier qu'aucun `.openspec.yaml` ne contient plus de `type:` scalaire avec `grep -rEn '^\s*type:\s*("?[a-z]|"")\s*$' openspec/changes --include=.openspec.yaml` (doit ne rien retourner).
- [x] 1.4 Vérifier par échantillonnage que la conversion est correcte : inspecter `git diff` sur au moins un fichier de chaque valeur legacy (`frontend`, `backend`, `batch`, `fullstack`, `""`) et confirmer que la liste produite correspond à la table de `design.md`, et que les autres champs (`schema`, `created`, `tags.complexity`, `tags.components`, `tags._auto`, `tags._tagged_at`, `dependencies`) sont inchangés.

## 2. Modèle de données backend

- [x] 2.1 Dans `backend/internal/openspec/change.go`, changer `Tags.Type` de `string` en `[]string` (tags `json:"type"` et `yaml:"type"` inchangés). Vérifier avec `go build ./...` (le build échoue tant que `tagger.go` n'est pas mis à jour à l'étape suivante - normal, corrigé en 2.2).
- [x] 2.2 Dans `backend/internal/openspec/tagger.go`, réécrire `DeriveType` pour qu'elle renvoie `[]string` : initialise `[]string{}`, ajoute `"frontend"` si `frontend/` est présent, `"backend"` si `backend/` est présent, `"batch"` si `scripts/`/`batch/`/`cmd/` est présent (aucune valeur combinée `fullstack`). Adapter `TagChange` qui assigne `tags.Type = tagType` en conséquence. Vérifier avec `go build ./...` que le package compile.
- [x] 2.3 Écrire/adapter les tests unitaires de `DeriveType` (nouveau fichier `backend/internal/openspec/tagger_test.go` s'il n'existe pas) couvrant : aucun chemin reconnu → `[]`, frontend seul → `[frontend]`, frontend+backend → `[frontend, backend]`, les trois catégories → `[frontend, backend, batch]`. Vérifier avec `go test ./internal/openspec/...`.
- [x] 2.4 Vérifier que `loadChange` (`change.go`) et les handlers API (`backend/internal/api/handlers`) qui sérialisent `Change`/`ChangeDetail` ne font aucune hypothèse résiduelle sur `Tags.Type` en tant que chaîne (recherche `\.Type\b` dans `backend/internal/openspec` et `backend/internal/api`). Vérifier avec `go build ./...` puis `go test ./...` côté backend.

## 3. Nettoyage du script de migration

- [x] 3.1 Supprimer `backend/cmd/migratetypetags` une fois la migration exécutée et vérifiée (tâches 1.3-1.4). Vérifier avec `go build ./...` que le retrait ne casse rien.

## 4. Frontend : modèle et rendu

- [x] 4.1 Dans `frontend/src/hooks/useChanges.ts`, changer l'interface `Tags.type` de `string` en `string[]`. Vérifier avec `npx tsc --noEmit` que les erreurs de type attendues apparaissent dans les fichiers listés aux tâches suivantes.
- [x] 4.2 Dans `frontend/src/components/ChangeCard.tsx`, remplacer le badge unique (ternaire sur `change.tags.type`) par une boucle rendant un badge par valeur de `change.tags.type` (icône par catégorie : 🖥 frontend, ⚙ backend, ⚡ batch). Vérifier visuellement dans le navigateur qu'un change à plusieurs types affiche plusieurs badges.
- [x] 4.3 Dans `frontend/src/components/TimelineChangeCard.tsx`, retirer l'entrée `fullstack` de `TYPE_ICONS`, boucler sur `c.tags.type` pour rendre un bouton par valeur (chaque bouton appelle toujours `onFilterClick` avec sa propre valeur). Vérifier visuellement qu'un clic sur un badge type ajoute bien ce type aux filtres actifs de la Timeline.
- [x] 4.4 Dans `frontend/src/components/DetailPanel.tsx`, remplacer le badge type unique de l'onglet Tags par une boucle sur `data.tags.type` rendant un badge par valeur. Vérifier visuellement dans le DetailPanel d'un change à plusieurs types.
- [x] 4.5 Vérifier avec `npx tsc --noEmit` (depuis `frontend/`) qu'il ne reste aucune erreur de type après les tâches 4.1-4.4.

## 5. Frontend : recherche et filtrage

- [x] 5.1 Dans `frontend/src/pages/KanbanPage.tsx`, remplacer `c.tags?.type?.toLowerCase().includes(lower)` par un test sur toutes les valeurs du tableau (`c.tags?.type?.some(t => t.toLowerCase().includes(lower))`). Vérifier manuellement que saisir "backend" dans la recherche Kanban affiche un change dont `tags.type` contient `backend` parmi plusieurs valeurs.
- [x] 5.2 Dans `frontend/src/pages/TimelinePage.tsx`, remplacer `c.tags?.type === f` par `c.tags?.type?.includes(f)` dans le filtre `filtered`. Vérifier manuellement qu'un clic sur un badge type dans la Timeline filtre correctement les entrées à plusieurs types.

## 6. Vérification finale

- [x] 6.1 Exécuter la suite de tests backend (`go test ./...` depuis `backend/`) et frontend (tests existants pertinents, ex. `npm test` si applicable) et confirmer qu'ils passent.
- [x] 6.2 Démarrer l'application (backend + frontend), ouvrir le Kanban et la Timeline, et confirmer visuellement que les changes migrés affichent les badges type attendus (y compris les anciens changes `fullstack` affichant désormais deux badges frontend+backend) sans erreur console.
