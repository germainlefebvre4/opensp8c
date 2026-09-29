# Proposal

## Why

Sur la page Specs, onglet Documentation, `POST /api/workspaces/{id}/docs/generate` répond 500 (`fork/exec .../claude: argument list too long`). `docsgen.BuildPrompt` concatène tous les `openspec/specs/*/spec.md` (≈328 Ko pour 57 specs) dans le prompt système, passé à la CLI de l'agent en un seul argument de ligne de commande, alors que Linux limite un argument à 128 Ko (`MAX_ARG_STRLEN`). La génération casse dès que le corpus de specs dépasse ce seuil, et continuera de casser en grossissant.

## What Changes

- Le prompt de génération ne contient plus le contenu des specs : il conserve le formalisme fixe et indique à l'agent de lire lui-même `openspec/specs/*/spec.md` dans le répertoire du workspace (déjà son répertoire de travail).
- `docsgen.BuildPrompt` cesse de lire et concaténer les fichiers de specs.
- `DocsFormalismPrompt` est reformulé (plus de « specs fournies ci-dessous »).
- La taille du prompt devient indépendante de la taille du corpus de specs.

## Capabilities

### New Capabilities

### Modified Capabilities
- `spec-documentation-generation`: nouvelle exigence — la génération doit aboutir quelle que soit la taille des specs du workspace (le contenu des specs n'est pas transmis via la ligne de commande de l'agent).

## Impact

- `backend/internal/docsgen/docsgen.go` (`BuildPrompt`) et son test `TestBuildPromptIncludesAllSpecFiles`.
- `backend/internal/agents/docs_prompt.go` (`DocsFormalismPrompt`) et son test.
- `backend/internal/api/handlers/docs.go` : inchangé fonctionnellement.
- Aucun changement d'API ni de dépendance.
