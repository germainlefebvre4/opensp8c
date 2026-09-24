# Design

## Context

Voir `proposal.md` - Why/What Changes pour la motivation. Rappels de code utiles pour situer ce design :

- `backend/internal/openspec/change.go` : struct `Tags{Type, Complexity, Components, Auto, TaggedAt}`, (dé)sérialisée depuis/vers `.openspec.yaml`.
- `backend/internal/openspec/tagger.go` : `DeriveType` (heuristique tasks.md), `ExtractVocabulary` (scan de tous les `.openspec.yaml` pour le vocabulaire `components`), `LLMDeriveComplexityAndComponents` (appel `claude -p`, 60s timeout, parsing JSON), `TagChange` (orchestration + écriture, dégradation silencieuse si le LLM échoue).
- `backend/internal/api/handlers/preferences.go` (préférences globales, `GET`/`PATCH /api/preferences`) : gère déjà `defaultAgent`, `env`, `systemEnv`, le toggle "mode question native" — un réglage global de plus (`customAgentSpecializations`) suit exactement ce pattern.
- `frontend/src/components/ChangeCard.tsx:195-213` : rendu actuel des badges `type`/`complexity` ; `components` n'est aujourd'hui affiché nulle part.
- `frontend/src/components/Layout.tsx:50-54` : nav actuelle (Kanban / Specs / Timeline / Agents), pas d'écran Settings.

## Goals / Non-Goals

**Goals:**
- Un vocabulaire de spécialisation fermé (base codée en dur + extensions utilisateur), distinct du vocabulaire ouvert de `components`.
- Réutiliser le pipeline de tagging existant (batch démarrage, trigger archivage, `/retag`) sans dupliquer son orchestration.
- Réutiliser le mécanisme de préférences globales existant pour stocker les extensions, plutôt qu'introduire un nouveau fichier.

**Non-Goals:**
- Le matching/affectation des workers du pool sur la base de `agent_specialization` (change futur).
- La personnalisation de la liste de base par workspace/projet (elle est globale à l'application, décision actée en amont).
- Une UI de gestion des extensions ailleurs que dans le nouvel écran Settings (pas d'ajout depuis `ChangeCard` ou la modale de review, par exemple).

## Decisions

### Le vocabulaire fermé vit dans le backend, pas dans le prompt du frontend
La liste de base est une constante Go (`backend/internal/openspec/specializations.go`, ex. `var BaseAgentSpecializations = []string{...}`). Le vocabulaire combiné (base + `customAgentSpecializations` lu depuis `preferences.json`) est calculé côté backend et exposé via `GET /api/agent-specializations`, consommé à la fois par le frontend (écran Settings, badges) et par `tagger.go` (construction du prompt LLM). Alternative écartée : dupliquer la liste de base côté frontend (TypeScript) — rejetée pour éviter la dérive entre les deux copies.

### Le tagger filtre les valeurs hors vocabulaire après coup, plutôt que de faire confiance au prompt
Le prompt liste explicitement le vocabulaire fermé et demande au LLM de n'en sortir aucune valeur en dehors, mais `TagChange` filtre quand même la réponse (`agent_specialization = intersection(réponse LLM, vocabulaire fermé)`) avant écriture dans `.openspec.yaml` — comme le fait déjà `LLMDeriveComplexityAndComponents` en clampant `complexity` à 1-5 malgré la consigne donnée au prompt. Une valeur hallucinée est donc silencieusement écartée plutôt que de corrompre le champ fermé.

### `agent_specialization` réutilise le flag `_auto`/`_tagged_at` partagé du bloc `tags`, pas un flag par champ
Cohérent avec le modèle actuel où `_auto` s'applique à l'ensemble du bloc `tags` (voir `change-tags` - Requirement "Endpoint de retag manuel"). Introduire un flag par champ (`_auto: {type: ..., components: ..., agent_specialization: ...}`) casserait la compatibilité de `.openspec.yaml` déjà écrits par ce projet et ajouterait de la complexité non demandée par la proposition.

### L'écran Settings est un nouveau point de navigation minimal, pas un système de settings générique
Le proposal prévoit explicitement que Settings accueillera d'autres réglages plus tard, mais ce change n'introduit qu'une page avec une seule section (spécialisations). Pas de sous-navigation, de catégories ou de schéma de configuration extensible construits par anticipation — cela restera à concevoir quand un deuxième réglage y sera ajouté (YAGNI).

### Pas de validation de format stricte au niveau du modèle de données
`customAgentSpecializations` comme `agent_specialization` restent `[]string` sans regex de validation imposée côté Go au-delà d'un kebab-case basique déjà pratiqué pour `components`. La déduplication contre la liste de base (Requirement "Extension de la liste par l'utilisateur") se fait par comparaison de chaîne exacte.

## Risks / Trade-offs

- **Liste de base globale, pas par workspace** → un utilisateur avec des projets de stacks très différentes (ex. Go+React et Python) partage le même vocabulaire de spécialisation entre les deux. Accepté explicitement : la liste de base reste générique/disciplinaire (frontend, backend, database, devops, testing, security, ...) plutôt que liée à un framework, et les extensions utilisateur permettent de compenser au cas par cas.
- **Filtrage silencieux des valeurs hors vocabulaire** → si le LLM se trompe systématiquement, `agent_specialization` peut rester vide sans que l'utilisateur en soit informé autrement que par une valeur manquante sur la carte. Mitigation : comportement volontairement aligné sur la dégradation silencieuse déjà en place pour `complexity`/`components` quand `claude` est indisponible ; pas de nouveau mode d'erreur à gérer côté UI dans ce change.
- **Suppression d'une extension déjà utilisée sur des changes tagués** → les changes existants gardent une valeur qui n'est plus dans le vocabulaire courant. Accepté (cf. Requirement "Extension de la liste par l'utilisateur" - pas de purge rétroactive) ; un futur `/retag` sur ce change re-dériverait proprement dans le vocabulaire réduit.

## Migration Plan

Aucune migration de données : `agent_specialization` absent d'un `.openspec.yaml` existant est traité comme un tableau vide (voir `change-tags` - Requirement "Tags sémantiques stockés"). À noter, tel quel : le batch au démarrage (`change-tags` - Requirement "Batch de tagging rétroactif") et le trigger à l'archivage ne se déclenchent que pour les changes qui n'ont **aucune** section `tags` — un change déjà tagué avant ce champ (ex. `kanban-delete-change`, `agent-pool-status-visibility`, tous deux dans ce repo avec `tags` déjà présent) n'est donc pas repris automatiquement par ces deux triggers existants, ni modifié par ce change. Backfill possible via un `/retag` manuel sur chacun. Décider d'étendre le déclenchement automatique à "tags présents mais `agent_specialization` manquant" est explicitement hors périmètre ici (cf. Non-Goals) ; ce change se limite à faire fonctionner correctement `/retag` avec le nouveau champ.
