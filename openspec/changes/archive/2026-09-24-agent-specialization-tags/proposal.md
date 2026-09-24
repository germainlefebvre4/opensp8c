# Proposal

## Why

Le pool d'agents assigne aujourd'hui les workers uniquement par disponibilité et par DAG de dépendances (`pool/scheduler.go`, `pool/manager.go`) : rien ne relie un change à une expertise technique particulière. Pour pouvoir un jour spécialiser l'affectation des agents sur les tâches à implémenter, il faut d'abord un vocabulaire de spécialisation cadré et stable à appliquer sur les changes — un champ ouvert et dérivé librement par LLM (comme `components` aujourd'hui) ne suffit pas : il faut une liste fermée mais extensible, pour que les valeurs restent exploitables comme critères de matching plus tard.

## What Changes

- Ajout d'un champ `agent_specialization` (tableau de slugs) dans la section `tags` de `.openspec.yaml`, aux côtés de `type`, `complexity` et `components` existants.
- Liste de base fermée de spécialisations, codée en dur côté backend (générique, indépendante de la stack d'un workspace donné) : `frontend`, `backend`, `database`, `api-design`, `devops`, `testing`, `security`, `documentation`, `ux-design`, `data`, `mobile`.
- Extension utilisateur : l'utilisateur peut ajouter ses propres valeurs par-dessus la liste de base. Ces valeurs additionnelles sont persistées dans `preferences.json` (nouveau champ `customAgentSpecializations`), exposées via les endpoints existants `GET`/`PATCH /api/preferences`.
- Le service de tagging (`tagger.go`) dérive `agent_specialization` via le même appel LLM que pour `complexity`/`components`, mais contraint la réponse à choisir uniquement parmi la liste de base + les extensions utilisateur (pas d'invention libre, contrairement à `components`). Override manuel possible, avec la même mécanique `_auto`/`_tagged_at` déjà en place pour le reste du bloc `tags`.
- Les triggers de tagging existants (batch au démarrage, déclenchement à l'archivage, endpoint `POST /.../retag`) dérivent aussi `agent_specialization` sans changement de leur déclenchement.
- Nouvel écran "Settings" dans la navigation principale (à côté de Kanban / Specs / Timeline / Agents), pour gérer les extensions de la liste de spécialisations. C'est le premier réglage de cet écran, pensé pour en accueillir d'autres par la suite.
- Affichage : badges `agent_specialization` sur `ChangeCard` (contrairement à `components`, qui n'est aujourd'hui affiché nulle part dans l'UI).

**Hors périmètre** : l'affectation des workers du pool d'agents sur la base de `agent_specialization` (matching agent ↔ tag). Cette exploration/change pose uniquement le vocabulaire et sa dérivation ; le matching reste un change futur.

## Capabilities

### New Capabilities
- `agent-specialization`: liste fermée et extensible de tags de spécialisation applicables aux changes, gestion des extensions utilisateur (persistées dans `preferences.json`) et écran de configuration dédié.

### Modified Capabilities
- `change-tags`: la section `tags` de `.openspec.yaml` gagne le champ `agent_specialization` ; la dérivation LLM existante (`Requirement: Dérivation automatique de la complexité et des composants via LLM`) est étendue pour aussi classifier `agent_specialization` parmi un vocabulaire fermé, en plus de continuer à dériver librement `components`.

## Impact

- Backend Go : `backend/internal/openspec/change.go` (champ `AgentSpecialization []string` sur `Tags`), nouveau fichier listant la liste de base (ex. `backend/internal/openspec/specializations.go`), `backend/internal/openspec/tagger.go` (prompt LLM étendu, vocabulaire fermé = base + extensions), handler des préférences (nouveau champ `customAgentSpecializations` dans `preferences.json`, `GET`/`PATCH /api/preferences`).
- Frontend React : nouvelle page `frontend/src/pages/SettingsPage.tsx`, entrée de navigation dans `Layout.tsx`, badges dans `ChangeCard.tsx`, clés i18n (`en`/`fr`).
- Pas de migration de données : les changes déjà tagués sans `agent_specialization` restent valides (champ absent = tableau vide). Le batch au démarrage et le trigger à l'archivage ne reprennent que les changes sans section `tags` du tout ; un change déjà tagué doit être backfillé via `POST /.../retag` manuel (voir design.md - Migration Plan).
