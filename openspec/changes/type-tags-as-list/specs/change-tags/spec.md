# Spec Delta

## MODIFIED Requirements

### Requirement: Tags sémantiques stockés dans `.openspec.yaml`
Chaque change SHALL pouvoir porter une section `tags` dans son `.openspec.yaml` contenant trois champs : `type` (liste de chaînes, chacune parmi `frontend`, `backend`, `batch`, sans doublon), `complexity` (entier 1–5) et `components` (tableau de chaînes kebab-case). Les champs `_auto` (booléen) et `_tagged_at` (date ISO) SHALL également être présents pour tracer l'origine de la dérivation. La section `tags` est optionnelle — son absence est valide et ne produit aucune erreur. `type` peut être une liste vide lorsqu'aucune catégorie n'a pu être déterminée.

#### Scenario: Change avec tags complets
- **WHEN** un `.openspec.yaml` contient une section `tags` avec `type`, `complexity` et `components`
- **THEN** le backend parse ces valeurs et les expose dans la réponse API du change (champ `tags`), avec `tags.type` sérialisé comme un tableau

#### Scenario: Change avec plusieurs valeurs de type
- **WHEN** un `.openspec.yaml` contient `tags.type: [frontend, backend]`
- **THEN** le backend expose `tags.type` comme un tableau contenant les deux valeurs, dans l'ordre où elles apparaissent dans le fichier

#### Scenario: Change sans section tags
- **WHEN** un `.openspec.yaml` ne contient pas de section `tags`
- **THEN** le champ `tags` dans la réponse API est `null` ou absent, sans erreur

### Requirement: Dérivation automatique du type applicatif par heuristique
Le service de tagging SHALL dériver le champ `type` en analysant les chemins de fichiers présents dans `tasks.md` du change : la présence de chemins préfixés par `frontend/` ajoute `frontend` à la liste, par `backend/` ajoute `backend`, par `scripts/` ou `batch/` ajoute `batch`. Chaque catégorie détectée apparaît au plus une fois dans la liste résultante, dans l'ordre frontend, backend, batch. Aucune valeur combinée (telle que l'ancienne valeur `fullstack`) n'est produite : la présence simultanée de plusieurs catégories se traduit par une liste à plusieurs éléments. En l'absence de chemin reconnaissable, le service tente une dérivation depuis le préfixe du nom du change ; s'il n'y parvient pas, `type` est une liste vide.

#### Scenario: Tasks.md avec chemins frontend uniquement
- **WHEN** `tasks.md` contient des lignes avec des chemins `frontend/...` et aucun chemin `backend/` ni `scripts/`/`batch/`
- **THEN** le champ `type` dérivé est `[frontend]`

#### Scenario: Tasks.md avec chemins mixtes frontend et backend
- **WHEN** `tasks.md` contient des chemins `frontend/` et `backend/`
- **THEN** le champ `type` dérivé est `[frontend, backend]`

#### Scenario: Tasks.md avec les trois catégories
- **WHEN** `tasks.md` contient des chemins `frontend/`, `backend/` et `scripts/` (ou `batch/`)
- **THEN** le champ `type` dérivé est `[frontend, backend, batch]`

#### Scenario: Tasks.md sans chemins reconnaissables
- **WHEN** `tasks.md` ne contient aucun chemin de fichier préfixé par un domaine connu
- **THEN** le service tente de dériver le type depuis le préfixe du nom du change, ou laisse `type` comme une liste vide
