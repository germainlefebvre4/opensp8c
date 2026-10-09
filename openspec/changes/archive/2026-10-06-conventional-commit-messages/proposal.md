# Proposal

## Why

Les commits que l'application crée pour un change n'ont pas de message fidèle à son contenu : le commit du worker s'appelle toujours `feat(<nom-kebab>): apply OpenSpec change`, le commit de correction `chore(<nom-kebab>): add review correction` et le merge `Merge change <nom-kebab>`. Le scope est le nom complet du change, le sujet est générique et le merge n'est pas conventionnel, alors que l'historique du dépôt suit `type(scope): Sujet` avec des scopes courts de domaine. Quand l'utilisateur relit `main` ou génère un changelog, ces lignes n'indiquent ni ce qui a changé ni où.

## What Changes

- **Format commun** : tout commit créé pour un change SHALL suivre Conventional Commits : `type(scope): Sujet`, suivi d'un corps portant au moins `Change: <nom-du-change>`. Le sujet est le nom du change formaté (tirets en espaces, première lettre en majuscule), tronqué sur une frontière de mot pour que l'en-tête ne dépasse pas 72 caractères (le nom complet reste dans le corps).
- **Type** : déduit du premier mot du nom du change (`fix`, `refactor`, `perf`, `docs`/`doc`, `test`, `chore`, `ci`, `build`, `style`), `feat` à défaut.
- **Scope** : le premier élément de `tags.components` du `.openspec.yaml` du change ; à défaut, le premier dossier de capacité (ordre alphabétique) de `specs/` du change ; à défaut, le scope est omis. Le scope est normalisé (minuscules, caractères non conformes remplacés par `-`).
- **Commit du worker** : `feat(<scope>): Improve matrix change drilldown nav` (le type suit la règle ci-dessus), corps `Change: <nom>`.
- **Commit de correction** : `chore(<scope>): Add review correction`, corps `Change: <nom>`.
- **Merge `--no-ff`** : même en-tête et même corps que le commit du worker, de sorte que la ligne qui atterrit dans l'historique de la branche cible dise ce que le change apporte.
- **Réutilisable** : la construction du message est une fonction unique que les futurs commits créés par l'application (coche de tâche, change `tasks-follow-the-branch`) réutilisent.

Hors périmètre : le message du merge d'intégration de la branche cible dans la branche du change (`git merge --no-edit`, message produit par git) ; la réécriture des commits existants ; le choix du type ou du scope par un agent ; un format configurable par workspace.

## Capabilities

### New Capabilities

- `change-commit-messages`: format Conventional Commits des commits créés par l'application pour un change (type, scope, sujet, corps) et son application aux commits du worker, de correction et au merge.

### Modified Capabilities

Aucune : les specs existantes ne définissent pas le message de ces commits.

## Impact

- Backend : nouveau fichier du paquet `internal/pool` (formatage du message, lecture des tags et des capacités du change), `internal/pool/worktree.go` (`CommitAll`, `MergeInto`), `internal/pool/review_actions.go` (commit de correction).
- Tests : `internal/pool/review_actions_test.go` (assertion sur `Merge change conc`), tests de `worktree_test.go` et `worker_test.go` qui lisent les messages.
- Docs : `docs/opensp8c/workflows.md` (section d'intégration et de fusion) pour la convention de message.
- Aucun changement d'API HTTP ni de frontend.
