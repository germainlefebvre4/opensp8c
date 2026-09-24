# Design

## Context

`tags.type` est aujourd'hui une chaîne unique (`Tags.Type string` dans `backend/internal/openspec/change.go`), dérivée par `DeriveType` (`backend/internal/openspec/tagger.go`) et lue/écrite via `.openspec.yaml` par un `yaml.Unmarshal`/`yaml.Marshal` sur la struct `openspecMeta`/`Tags`. Environ 65 fichiers `.openspec.yaml` existants (changes actifs et archivés) portent déjà une valeur scalaire (`frontend`, `backend`, `batch`, `fullstack` ou `""`). Voir `proposal.md` - Why pour la motivation du passage à une liste.

Le filtrage/recherche par type est entièrement côté client (aucun endpoint ne prend de query param `type`), donc pas de contrat d'API à versionner sur ce point.

## Goals / Non-Goals

**Goals:**
- `Tags.Type` devient `[]string` côté Go, JSON et YAML, sans valeur combinée type `fullstack`.
- Tous les `.openspec.yaml` existants sont migrés une fois vers le nouveau format, sans code de compatibilité double-format permanent.
- Le rendu et le filtrage UI gèrent nativement une liste (pattern déjà utilisé pour `tags.components`).

**Non-Goals:**
- Ne pas introduire de nouvelle catégorie de type au-delà de `frontend`/`backend`/`batch`.
- Ne pas changer la logique de déclenchement du tagging (batch au démarrage, trigger à l'archivage, endpoint `/retag`) : seul le format de `type` change.
- Ne pas construire un mécanisme de lecture tolérante permanent (string OU liste) : le format legacy est éliminé par migration, pas contourné à la lecture.

## Decisions

### D1 : Migration one-shot via un petit programme Go séparé, pas de compatibilité de lecture permanente
Un programme Go autonome (`backend/cmd/migratetypetags/main.go`) parcourt `openspec/changes/*/.openspec.yaml` et `openspec/changes/archive/*/.openspec.yaml`, convertit `tags.type` scalaire en liste, et réécrit chaque fichier. Il est exécuté une fois (`go run ./cmd/migratetypetags`) pendant l'implémentation de ce change, puis supprimé du repo dans la dernière tâche - c'est un outil de transition, pas une fonctionnalité durable.

Alternative rejetée : accepter les deux formats en lecture (`UnmarshalYAML` custom tolérant scalaire ou liste) et laisser la migration se faire progressivement au fil des retags. Rejetée explicitement au profit d'une migration en masse immédiate, pour ne pas garder deux formats vivants indéfiniment dans les fichiers du repo.

### D2 : Le script de migration ne réutilise pas la struct `Tags` de production
Le script embarque sa propre struct de lecture locale (`Type string`, reste identique à `openspecMeta`/`Tags`) car au moment où le script tourne, la struct `Tags` de production aura déjà été mise à jour vers `Type []string` dans le même changement de code - la réutiliser pour lire les anciens fichiers scalaires échouerait silencieusement (voir Risques). Il écrit ensuite avec la struct de production (`Type []string`), ce qui garantit que le fichier migré est strictement conforme au nouveau format lu par le backend.

### D3 : Table de conversion scalaire → liste
| Valeur legacy | Nouvelle valeur |
|---|---|
| `frontend` | `[frontend]` |
| `backend` | `[backend]` |
| `batch` | `[batch]` |
| `fullstack` | `[frontend, backend]` |
| `""` (vide) | `[]` |

Cette table correspond exactement à la sémantique que `DeriveType` aurait produite si elle avait toujours renvoyé une liste (voir Requirement modifié dans `specs/change-tags/spec.md`).

### D4 : `Tags.Type` est toujours une liste initialisée, jamais `nil`
`DeriveType` et le script de migration initialisent `Type` à `[]string{}` (jamais un slice nil) avant d'y ajouter des éléments, pour que la sérialisation JSON produise `[]` et non `null` quand aucune catégorie n'est détectée. Ça évite d'ajouter un garde `?? []` à chaque consommateur frontend.

### D5 : Rendu UI en un badge par valeur, réutilisant le pattern `components`
`ChangeCard.tsx`, `TimelineChangeCard.tsx` et `DetailPanel.tsx` bouclent sur `tags.type` et rendent un badge par valeur (au lieu d'un badge unique avec ternaire). `TimelineChangeCard.TYPE_ICONS` perd son entrée `fullstack` (plus jamais produite) et garde `frontend`/`backend`/`batch`. Le filtrage (`TimelinePage.tsx`, `KanbanPage.tsx`) passe d'une égalité stricte / `includes` sur une string à un test d'appartenance sur le tableau (`tags?.type?.includes(f)`, `tags?.type?.some(t => t.toLowerCase().includes(lower))`).

## Risks / Trade-offs

- **[Risque] Un fichier `.openspec.yaml` non migré (script interrompu, nouveau fichier créé entre-temps) contient encore `type: frontend` en scalaire une fois le code de production passé en `[]string`.** → `yaml.v3` ne fait pas échouer tout le `Unmarshal` sur une erreur de type de champ : il continue de décoder les autres champs (`created`, `schema`, `tags.complexity`, `tags.components`, `dependencies`) et laisse seulement `Type` à sa valeur zéro (liste vide). Dégradation gracieuse déjà couverte par le comportement actuel de `loadChange` (`_ = yaml.Unmarshal(...)`), pas de crash ni de perte des autres champs. Mitigation additionnelle : une tâche de vérification post-migration (`grep -rn "^\s*type: [a-z]" openspec/changes --include=.openspec.yaml` doit ne rien retourner) avant de considérer la migration terminée.
- **[Risque] Le script de migration tourne deux fois par erreur.** → Idempotent par construction : un fichier dont `tags.type` est déjà une liste YAML (et non un scalaire) n'est pas modifié par la conversion (elle ne s'applique qu'aux noeuds scalaires détectés).
- **[Trade-off] Le script de migration est un outil jetable ajouté puis supprimé dans le même change.** → Cohérent avec le Non-Goal "pas de compatibilité permanente" ; laisse un historique git clair (le commit de suppression référence le commit d'ajout) sans dead code durable.

## Migration Plan

1. Écrire et exécuter `backend/cmd/migratetypetags` **avant** de changer la struct `Tags` de production, en utilisant la struct legacy (`Type string`) en lecture et une struct locale `Type []string` en écriture pour ce script (indépendante de `Tags`, pour ne pas dépendre de l'ordre des commits).
2. Vérifier qu'aucun `.openspec.yaml` ne contient plus `type:` scalaire (`grep` de contrôle).
3. Mettre à jour `Tags.Type` en `[]string` dans `change.go`, `DeriveType` et `TagChange` dans `tagger.go`.
4. Mettre à jour le frontend (`useChanges.ts`, `ChangeCard.tsx`, `TimelineChangeCard.tsx`, `DetailPanel.tsx`, `KanbanPage.tsx`, `TimelinePage.tsx`).
5. Supprimer `backend/cmd/migratetypetags`.

Pas de rollback dédié : le changement de format et la migration des fichiers sont commités ensemble dans ce change ; un rollback se fait par `git revert` classique, sans état externe (base de données, cache) à réconcilier.
