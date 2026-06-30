# Spec: change-tags

## Purpose

Système de tagging sémantique des changes OpenSpec. Permet d'associer à chaque change un type applicatif, un niveau de complexité et une liste de composants touchés, dérivés automatiquement ou saisis manuellement.

## Requirements

### Requirement: Tags sémantiques stockés dans `.openspec.yaml`
Chaque change SHALL pouvoir porter une section `tags` dans son `.openspec.yaml` contenant trois champs : `type` (chaîne parmi `frontend`, `backend`, `batch`, `fullstack`), `complexity` (entier 1–5) et `components` (tableau de chaînes kebab-case). Les champs `_auto` (booléen) et `_tagged_at` (date ISO) SHALL également être présents pour tracer l'origine de la dérivation. La section `tags` est optionnelle — son absence est valide et ne produit aucune erreur.

#### Scenario: Change avec tags complets
- **WHEN** un `.openspec.yaml` contient une section `tags` avec `type`, `complexity` et `components`
- **THEN** le backend parse ces valeurs et les expose dans la réponse API du change (champ `tags`)

#### Scenario: Change sans section tags
- **WHEN** un `.openspec.yaml` ne contient pas de section `tags`
- **THEN** le champ `tags` dans la réponse API est `null` ou absent, sans erreur

### Requirement: Dérivation automatique du type applicatif par heuristique
Le service de tagging SHALL dériver le champ `type` en analysant les chemins de fichiers présents dans `tasks.md` du change : la présence de chemins préfixés par `frontend/` indique `frontend`, par `backend/` indique `backend`, par `scripts/` ou `batch/` indique `batch`. La présence simultanée de `frontend/` et `backend/` produit `fullstack`. En l'absence de chemin reconnaissable, le service tente une dérivation depuis le préfixe du nom du change.

#### Scenario: Tasks.md avec chemins frontend uniquement
- **WHEN** `tasks.md` contient des lignes avec des chemins `frontend/...` et aucun chemin `backend/`
- **THEN** le champ `type` dérivé est `frontend`

#### Scenario: Tasks.md avec chemins mixtes
- **WHEN** `tasks.md` contient des chemins `frontend/` et `backend/`
- **THEN** le champ `type` dérivé est `fullstack`

#### Scenario: Tasks.md sans chemins reconnaissables
- **WHEN** `tasks.md` ne contient aucun chemin de fichier préfixé par un domaine connu
- **THEN** le service tente de dériver le type depuis le préfixe du nom du change, ou laisse `type` vide

### Requirement: Dérivation automatique de la complexité et des composants via LLM
Le service de tagging SHALL invoquer la CLI `claude --print` avec le contenu de `proposal.md` et `design.md` du change pour dériver `complexity` (entier 1–5, 1=correction triviale, 5=refactoring architectural) et `components` (liste de slugs kebab-case identifiant les zones fonctionnelles touchées). Le vocabulaire existant des composants du workspace SHALL être fourni en contexte au LLM pour normaliser les slugs contre les termes déjà utilisés.

#### Scenario: Dérivation réussie avec vocabulaire existant
- **WHEN** le tagger est déclenché pour un change ayant `proposal.md`, et que des composants existent déjà dans d'autres changes du workspace
- **THEN** le LLM réutilise les slugs existants pour les composants correspondants et ne crée de nouveaux slugs que si aucun terme existant ne correspond sémantiquement

#### Scenario: Dérivation avec vocabulaire vide (premier change)
- **WHEN** le tagger est déclenché pour le premier change et aucun composant n'existe encore dans le workspace
- **THEN** le LLM crée librement des slugs kebab-case à partir du contenu du change

#### Scenario: CLI claude indisponible
- **WHEN** la CLI `claude` n'est pas installée ou inaccessible
- **THEN** le tagging des champs `complexity` et `components` est ignoré silencieusement, sans affecter le reste du fonctionnement

### Requirement: Vocabulaire des composants extrait dynamiquement du workspace
Le service SHALL extraire le vocabulaire courant des composants en lisant tous les `.openspec.yaml` du workspace (changes actifs et archivés) et en collectant l'union de leurs champs `tags.components`. Ce vocabulaire est recalculé à chaque déclenchement du tagger, sans fichier de cache séparé.

#### Scenario: Extraction du vocabulaire courant
- **WHEN** le tagger est déclenché pour un change
- **THEN** il scanne tous les `.openspec.yaml` du workspace (actifs + archivés) pour collecter les composants existants avant d'appeler le LLM

#### Scenario: Workspace sans changes tagués
- **WHEN** aucun change du workspace ne possède encore de tags
- **THEN** le vocabulaire passé au LLM est vide et le LLM crée librement les premiers slugs

### Requirement: Batch de tagging rétroactif au démarrage du serveur
Au démarrage, le backend SHALL déclencher en arrière-plan un tagging automatique de tous les changes du workspace (actifs et archivés) qui ne possèdent pas encore de section `tags`. Le traitement SHALL se faire dans l'ordre chronologique de création (du plus ancien au plus récent) pour construire le vocabulaire progressivement. Ce batch est non-bloquant et ne retarde pas la disponibilité de l'API.

#### Scenario: Démarrage avec changes non tagués
- **WHEN** le serveur démarre et des changes sans section `tags` existent
- **THEN** un batch background tague ces changes dans l'ordre chronologique sans bloquer les requêtes API

#### Scenario: Démarrage avec tous les changes déjà tagués
- **WHEN** tous les changes du workspace ont une section `tags`
- **THEN** le batch se termine immédiatement sans appel LLM

### Requirement: Trigger automatique du tagging à l'archivage
Lorsqu'un change est archivé via l'API, le backend SHALL déclencher automatiquement son tagging si sa section `tags` est absente.

#### Scenario: Archivage d'un change sans tags
- **WHEN** l'utilisateur archive un change qui n'a pas encore de section `tags`
- **THEN** le tagging est déclenché automatiquement après l'archivage

### Requirement: Endpoint de retag manuel
Le backend SHALL exposer un endpoint `POST /api/workspaces/{id}/changes/{name}/retag` permettant de déclencher explicitement le tagging (ou re-tagging) d'un change, en ignorant le flag `_auto`.

#### Scenario: Retag d'un change déjà tagué automatiquement
- **WHEN** une requête `POST /api/workspaces/{id}/changes/{name}/retag` est effectuée
- **THEN** le service re-dérive les tags depuis le contenu actuel des fichiers et met à jour `.openspec.yaml`, en conservant `_auto: true`

#### Scenario: Retag d'un change avec tags manuels
- **WHEN** une requête `/retag` est effectuée sur un change dont `_auto: false`
- **THEN** les tags sont re-dérivés et `_auto` repasse à `true`
