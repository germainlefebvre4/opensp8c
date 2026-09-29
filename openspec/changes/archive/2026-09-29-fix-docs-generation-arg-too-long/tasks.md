# Tasks

## 1. Prompt de génération sans specs inlinées

- [x] 1.1 Reformuler `DocsFormalismPrompt` (`backend/internal/agents/docs_prompt.go`) : remplacer « specs provided below » par l'instruction de lister et lire tous les `openspec/specs/*/spec.md` du projet (triés par nom) avant de rédiger ; mettre à jour le commentaire du const. Vérifier avec `go test ./internal/agents/...` (adapter `TestDocsFormalismPromptLoads` pour exiger la mention de `openspec/specs/`).
- [x] 1.2 Simplifier `docsgen.BuildPrompt` (`backend/internal/docsgen/docsgen.go`) pour ne retourner que `agents.DocsFormalismPrompt`, sans lire les specs ; ajuster le commentaire de la fonction. Vérifier avec `go build ./...`.
- [x] 1.3 Remplacer `TestBuildPromptIncludesAllSpecFiles` (`backend/internal/docsgen/docsgen_test.go`) par des tests vérifiant que le prompt ne contient pas le contenu des specs, reste sous ~100 Ko avec un corpus de plusieurs centaines de Ko dans un répertoire temporaire, et ne renvoie pas d'erreur pour un workspace sans spec. Vérifier avec `go test ./internal/docsgen/...`.

## 2. Vérification d'intégration

- [x] 2.1 Lancer `go test ./...` dans `backend/` et vérifier que les tests de handlers docs (`docs_test.go`, `language_launch_test.go`) passent toujours.
- [x] 2.2 Depuis l'UI, page Specs > onglet Documentation, lancer la génération sur ce workspace (57 specs) et vérifier qu'il n'y a plus de 500 et que les pages apparaissent sous `docs/opensp8c/`.
