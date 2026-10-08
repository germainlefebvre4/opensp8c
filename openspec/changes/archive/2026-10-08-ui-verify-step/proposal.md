# Proposal

## Why

`verification-stage` fournit l'étape persistante de vérification, avec la conformité comme seule étape. Il reste ce que la validation locale et la conformité ne couvrent pas : vérifier que le change fonctionne dans l'application lancée, c'est-à-dire ce que l'humain fait aujourd'hui à la main (« parcours manuel dans l'application »). Ce change ajoute la seconde étape de la file, la vérification UI : la plateforme lance l'application depuis le worktree du change, un agent de rôle `verifier` exécute les scénarios dans un navigateur, et les tâches de validation humaine qu'il a vérifiées sont cochées sans intervention. L'étape est désactivée par défaut (coût en tokens, voir `verification-settings`).

## What Changes

- **Étape `ui`** ajoutée à la liste ordonnée des étapes de `verification-stage`, après la conformité : elle ne s'exécute que si la conformité a réussi et que `ui` est résolue à activée. Si `uiStartCommand` ou `uiBaseUrl` manque, l'étape échoue aussitôt avec une raison explicite, sans lancer d'agent.
- **Verrou exclusif** : une seule étape `ui` à la fois sur tout le backend (ports, base de données et navigateur ne se partagent pas). Une vérification qui attend le verrou est dans l'état `waiting`, ne compte pas dans la limite de concurrence des vérifications et ne bloque donc pas les vérifications de conformité.
- **Cycle de vie de l'application** géré par la plateforme : lancement de `uiStartCommand` dans le worktree avec un port libre choisi par la plateforme (substitué dans la commande et dans `uiBaseUrl` via `{port}`, et exporté en `PORT`), attente que l'URL réponde, arrêt systématique du groupe de processus quelle que soit l'issue.
- **Scénarios soumis à l'agent** : les scénarios WHEN / THEN des specs delta du change et les tâches non cochées portant le marqueur `<!-- human review required -->`. L'agent choisit son outil de navigation et conclut par `VERDICT: PASS`, `FAIL` ou `SKIP` (aucun comportement observable dans l'interface), avec une ligne `TASK-VERIFIED:` par tâche marquée vérifiée.
- **Coche des tâches par la plateforme** : après un verdict `PASS`, la plateforme coche dans la branche, un commit par coche, les tâches marquées que l'agent a déclarées vérifiées. L'agent n'écrit jamais dans `tasks.md`. Un `SKIP` qui laisserait des tâches marquées non cochées est un échec.
- **Preuves** : captures d'écran enregistrées dans le dossier du run (`OPENSP8C_VERIFY_ARTIFACTS`), listées par l'endpoint de rapport, servies par un endpoint dédié et affichées en vignettes dans le bandeau de vérification du DetailPanel.
- **Paramètre `{port}`** : `uiBaseUrl` et `uiStartCommand` acceptent le jeton `{port}` ; la validation de `uiBaseUrl` l'accepte.

Hors périmètre : choisir ou installer l'outil de navigation (c'est l'agent qui choisit parmi ce que l'hôte offre), installer les dépendances du projet dans le worktree (à la charge de `uiStartCommand`), la réparation automatique après échec, la vérification UI de plusieurs navigateurs ou de plusieurs résolutions, la conservation des vidéos.

## Capabilities

### New Capabilities

- `ui-verification-step`: étape `ui` de la file de vérification : prérequis de configuration, verrou exclusif et état `waiting`, cycle de vie de l'application, scénarios soumis, contrat de verdict, coche des tâches vérifiées, preuves, rapport par étape.

### Modified Capabilities

- `verification-settings`: `uiBaseUrl` et `uiStartCommand` acceptent le jeton `{port}`.
- `human-review-tasks`: une tâche marquée peut être cochée par la vérification UI, et le marqueur est alors conservé.
- `kanban-change-detail`: les captures de la vérification UI sont affichées dans le DetailPanel.

## Impact

- Prérequis : `verification-stage` appliqué (liste `verifySteps`, exécuteur, marqueur, rapport, bandeau du DetailPanel) ; `verification-settings` l'est déjà (paramètres de lancement et `Resolve`).
- Backend : `internal/pool` (nouveaux `verify_ui.go` : étape, verrou, `appRunner`, parsing des lignes de verdict ; `verify.go` : ordre des étapes, comptage hors `waiting`, un run `verify` par étape), `internal/verification` (validation de `uiBaseUrl` avec `{port}`), `internal/openspec` (extraction des scénarios d'un change), `internal/api/handlers/verification.go` (liste des captures dans le rapport, endpoint de capture, paramètre `step`), `internal/api/router.go`.
- Frontend : bandeau de vérification du `DetailPanel` (vignettes, état `waiting`), `ChangeCard` (badge `waiting`), `lib/api.ts`, `hooks/useVerificationReport.ts`, locales fr/en (`kanban`, `detailPanel`).
- Données : un dossier de captures par run sous `conversations/<ws>/<change>/verify/<ts>/`, supprimé avec les journaux du change par le balayage de rétention existant ; commits de coche dans la branche du change.
- API : `GET …/verification/artifacts/{run}/{name}` ; champs `artifacts` et paramètre `step` sur `GET …/verification/report` ; état `waiting` dans `verification_state`.
- Environnement exposé à `uiStartCommand` : `PORT`, `OPENSP8C_UI_PORT`, `OPENSP8C_UI_URL`, `OPENSP8C_WORKSPACE_PATH`.
- Consommation de tokens : nulle tant que `ui` est désactivée.
