# Design

## Context

Voir `proposal.md` pour la motivation. État actuel observé dans `internal/pool` :

- `CommitAll` (`worktree.go`) committe le travail du worker avec un message codé en dur, `feat(<nom>): apply OpenSpec change`.
- `CommitFile` (`worktree.go`) committe un seul fichier avec le message que lui passe l'appelant ; son seul appelant est `RequestCorrection` (`review_actions.go`), qui passe `chore(<nom>): add review correction`.
- `MergeInto` (`worktree.go`) fusionne avec `merge --no-ff -m "Merge change <nom>"` ; il est utilisé par l'approbation de revue et par la fusion en `full-autonomy` (via `integrateAndMerge`).
- Le merge d'intégration de la branche cible dans la branche du change utilise `merge --no-edit` : git produit lui-même le message.
- Un test (`review_actions_test.go`) compte les merges par le sujet `Merge change conc` ; un autre vérifie que le sujet du commit de correction contient « correction ».
- Les métadonnées d'un change vivent dans son dossier : `.openspec.yaml` (dont `tags.components`, généré par le tagger, vocabulaire ouvert) et `specs/<capacité>/spec.md`. `WorktreeController` connaît `repoRoot` (dépôt principal) et peut résoudre le worktree d'un change (`resolvePath`).
- L'historique de `main` suit `type(scope): Sujet` avec des scopes courts de domaine ; aucune source automatique ne fournit ces scopes courts de façon fiable (les composants sont des noms de capacités longs).

## Goals / Non-Goals

**Goals:**
- Une seule fonction construit le message de tout commit créé pour un change, pour que les formats ne dérivent pas et que les futurs commits (coche de tâche) la réutilisent.
- Un message déterministe, testable sans agent, qui ne peut pas faire échouer un commit.

**Non-Goals:**
- Choisir le type ou le scope par un agent (option écartée : coût d'un tour et non-déterminisme, voir D2).
- Réécrire des commits existants, ou modifier le merge d'intégration produit par git.
- Rendre le format configurable par workspace.

## Decisions

### D1. Un fichier `commitmsg.go` dans `internal/pool`, des fonctions pures et une résolution de métadonnées séparée

Le formatage (type, sujet, troncature, assemblage) est écrit en fonctions pures sur des chaînes : `commitType(name)`, `commitSubject(name)`, `normalizeScope(raw)`, `commitHeader(type, scope, subject)`, `changeCommitMessage(name, scope)`. Seule la résolution du scope touche au disque : `WorktreeController.changeScope(name)` lit les tags puis les dossiers de capacités, dans le dossier du change du dépôt principal puis dans celui du worktree. `CommitAll`, `MergeInto` et `RequestCorrection` appellent ces fonctions ; `CommitFile` garde sa signature (le message vient de l'appelant), ce qui permet au change `tasks-follow-the-branch` de lui passer un message de coche construit par la même bibliothèque.

*Alternative :* un paquet `internal/commitmsg` exporté. Écartée pour l'instant : tous les appelants sont dans `pool`, et l'export pourra se faire sans changer le comportement si un autre paquet en a besoin.

### D2. Scope S1 déterministe, plutôt qu'un message écrit par un agent

Ordre : premier élément non vide de `tags.components`, sinon premier dossier de `specs/` par ordre alphabétique, sinon aucun scope. Le scope obtenu est parfois long (`timeline-spec-matrix` au lieu de `timeline`) mais il est stable, vérifiable en test et disponible aussi pour les commits sans agent (coches). Les tags étant générés par le tagger, ils peuvent être absents ou imprécis : le repli sur `specs/` puis sur « aucun scope » garantit que le commit aboutit toujours.

*Alternatives :* (a) un agent propose `type(scope): sujet` en fin de run : message plus soigné et scopes courts, mais un tour d'agent en plus, un résultat non déterministe, et rien pour les commits faits par l'API ; (b) tronquer les noms de capacités à leur premier segment : `agent-pool-orchestrator` donnerait `agent`, ce qui est faux. Un message écrit par un agent reste une évolution possible, par-dessus ce format.

### D3. Type déduit d'une table sur le premier mot du nom

Table fixe (`fix`, `refactor`, `perf`, `docs`/`doc`, `test`/`tests`, `chore`, `ci`, `build`, `style`), `feat` par défaut. C'est une heuristique, assumée : elle donne `feat` pour `improve-…`/`add-…` et `fix` pour `fix-…`, ce qui couvre les noms de l'historique. Le commit de correction est toujours `chore` : il n'ajoute qu'une tâche à `tasks.md`.

### D4. Sujet formaté, en-tête borné à 72 caractères, nom complet dans le corps

Le sujet est le nom avec `-` et `_` remplacés par des espaces et la première lettre en majuscule. Si l'en-tête dépasse 72 caractères, le sujet est coupé sur une frontière de mot (jamais au milieu d'un mot) ; la ligne `Change: <nom>` du corps garde le nom intact, donc le message reste fidèle même tronqué. Le corps est l'endroit où les commits à venir ajoutent leurs lignes (par exemple `Task: …`).

### D5. Le merge `--no-ff` reprend l'en-tête et le corps du commit du worker

`MergeInto` construit `changeCommitMessage` et le passe à `git merge -m` (le message est passé en un seul argument, avec son corps, via un fichier ou plusieurs `-m`, sans passer par un shell). L'en-tête est donc identique à celui du commit du worker : un doublon entre la branche et le merge est accepté, le merge étant la ligne qui représente le change dans l'historique de la branche cible. Le merge d'intégration de la cible dans la branche (`--no-edit`) n'est pas touché.

*Alternative :* `Merge <sujet>` ou `chore(scope): Merge …` pour le merge. Écartée : l'utilisateur veut que la ligne qui atterrit dans `main` dise ce que le change apporte.

### D6. Aucune erreur de métadonnées ne remonte

La lecture de `.openspec.yaml` et de `specs/` ignore les erreurs (fichier absent, YAML invalide, dossier absent) et retourne « aucun scope » : un message sans scope est conventionnel et acceptable, un commit refusé à cause d'un tag illisible ne l'est pas.

## Risks / Trade-offs

- [Scope long ou peu parlant quand les tags sont absents ou génériques] → le repli sur `specs/` puis sur « aucun scope » évite les valeurs trompeuses ; un scope plus soigné pourra venir d'un agent sans changer le format.
- [Le type `feat` par défaut est faux pour un change dont le nom ne commence pas par un mot de la table (par exemple `resume-paused-worker-after-manual-tasks`, qui est un correctif)] → heuristique assumée ; renommer le change ou, plus tard, un type choisi par l'utilisateur. Le corps `Change:` permet de retrouver le contexte.
- [Doublon d'en-tête entre le commit du worker et le merge] → accepté (D5) ; les outils de changelog ignorent en général les merges.
- [Les tests existants dépendent des anciens messages] → à mettre à jour dans le même change (voir tasks).
- [Les commits déjà présents dans des branches en cours gardent l'ancien format] → pas de réécriture d'historique ; sans conséquence fonctionnelle.

## Migration Plan

Aucune migration. Les nouveaux commits suivent le nouveau format dès le redémarrage du backend ; les commits existants ne sont pas réécrits. Retour arrière : revert du change.
