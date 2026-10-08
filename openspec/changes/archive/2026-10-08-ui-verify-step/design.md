# Design

## Context

Voir `proposal.md` (Why). Ce change se pose sur `verification-stage`, décrit dans son propre `design.md` (marqueur `pending` / `failed` / `passed`, exécuteur dans le `Manager` piloté par `tick`, interface `verifyStep` et liste `verifySteps`, journal `verify`, rapport, bandeau du DetailPanel). Il s'applique après lui. État observé du dépôt :

- `verification-settings` est appliqué : `verification.Resolve` renvoie `Resolved{Conformity, UI, UIStartCommand, UIBaseURL}` ; `ValidateBaseURL` n'accepte aujourd'hui qu'une URL `http(s)` absolue, ce qui rejette `http://localhost:{port}` (`url.Parse` échoue sur un port non numérique).
- Les workspaces sont des projets arbitraires, pas seulement opensp8c : la plateforme ne connaît ni leur façon de se lancer, ni leur outillage de test navigateur. `@playwright/test` figure dans les devDependencies du frontend d'opensp8c, mais sans configuration ni test, et le worktree d'un change ne contient ni `node_modules` ni build.
- Les workers lancent déjà des subprocess d'agent avec une consigne système additionnelle (`humanReviewDirective`) et un répertoire de travail ; `session.ApplyProcessGroup` place une commande dans son propre groupe de processus et tue le groupe à l'annulation ; `validation.go` exécute déjà des commandes `sh -c` avec délai maximal et `WaitDelay`.
- Les tâches de validation humaine se reconnaissent au commentaire `<!-- human review required -->` (`human-review-tasks`) ; `ParseTaskListContent` renvoie `Task{Text, Done, HumanReview}` avec le texte sans marqueur. Une coche dans la branche passe par `ToggleBranchTask` (verrou du change, `openspec.ToggleTask`, `CommitFile`, un commit par coche).
- Un run `verify` écrit `conversations/<ws>/<change>/verify/<ts>.jsonl` ; `listDir` ignore les sous-dossiers, et le balayage de rétention supprime tout le dossier du change archivé.

## Goals / Non-Goals

**Goals:**
- Exécuter l'application du change dans un environnement isolé de l'utilisateur (port libre) et la nettoyer dans tous les cas.
- Garder la vérification fail-closed et l'agent en lecture seule, tout en cochant les tâches humaines vérifiées.
- Ajouter l'étape sans modifier l'exécuteur de `verification-stage` autrement que par la liste d'étapes, le comptage de concurrence et le nom d'étape des runs.

**Non-Goals:**
- Fournir ou imposer un outil de navigation, ou installer des dépendances dans le worktree.
- Réparer après échec, ou conserver des vidéos et des traces.

## Decisions

### D1. L'agent choisit l'outil de navigation, la plateforme possède tout le reste

La plateforme maîtrise le cycle de vie de l'application, le verrou, les preuves, le verdict et la coche des tâches ; elle ne prescrit pas comment l'agent pilote le navigateur (script Playwright jetable via `npx`, outil de navigation de l'hôte, etc.). La consigne système lui interdit de modifier un fichier du dépôt et lui dit où déposer ses captures. Si l'hôte n'offre aucun moyen de naviguer, l'agent ne peut pas conclure `PASS` honnêtement et l'étape échoue (verdict `FAIL` ou absent) : fail-closed plutôt qu'un faux positif.

*Alternatives :* (a) embarquer un exécuteur Playwright dans la plateforme : ajoute Node, les navigateurs et leur version à un binaire Go servi aussi en Docker, pour un outillage que chaque projet organise différemment ; (b) imposer Playwright dans le projet vérifié : exclut les projets qui n'en ont pas, et le worktree n'a pas `node_modules`.

### D2. Verrou global dans le paquet `pool`, attente hors du décompte

```go
var uiLock = newFairLock()   // un titulaire, file FIFO, Acquire(ctx) / Release
```

Le verrou est une variable de paquet : les ports, la base de données et le navigateur sont des ressources de la machine, pas d'un workspace (plusieurs `Manager` coexistent via le `Registry`). `Acquire(ctx)` rend la main sur annulation du contexte, ce qui retire le demandeur de la file. Les entrées de `m.verifications` portent un champ `waiting`, positionné autour de `Acquire` ; le calcul de la limite de concurrence dans `tickVerifications` ignore les entrées `waiting`, et l'état exposé devient `waiting` (`verification_state`), avec `verification_step = "ui"`. Le verrou est pris juste avant de choisir un port et libéré par `defer` après l'arrêt de l'application, donc jamais détenu pendant que l'application tourne encore.

*Alternative :* un verrou par workspace. Écartée : deux workspaces partagent le même navigateur et risquent de partager la même base externe.

### D3. `appRunner` : démarrer, attendre, toujours arrêter

```go
type appRunner struct{ cmd *exec.Cmd; port int; url string; exited chan error; tail *ringBuffer }
func startApp(ctx, dir, command, baseURL, workspacePath string, journal func(line)) (*appRunner, error)
func (a *appRunner) WaitReady(ctx, timeout) error
func (a *appRunner) Stop(grace time.Duration)
```

- Port : `net.Listen("tcp", "127.0.0.1:0")`, lecture du port, fermeture. Il existe une fenêtre de course avant que l'application ne le lie ; elle est acceptée (verrou exclusif, machine de développement) et se manifeste par une sortie prématurée de l'application, donc un échec explicite.
- Substitution : `{port}` est remplacé textuellement dans la commande et dans l'URL ; `PORT`, `OPENSP8C_UI_PORT`, `OPENSP8C_UI_URL` et `OPENSP8C_WORKSPACE_PATH` s'ajoutent à l'environnement hérité.
- Lancement : `sh -c`, répertoire du worktree, `session.ApplyProcessGroup(cmd)` pour tuer le groupe entier ; stdout et stderr sont lus ligne à ligne vers le journal du run et un tampon circulaire (les 40 dernières lignes) qui nourrit la raison d'un échec de démarrage.
- Disponibilité : `GET baseURL` toutes les 500 ms avec un client de 2 s ; tout statut `< 500` vaut « prêt » (une SPA redirige ou répond `404` sur `/` sans que l'application soit en panne). Délai de démarrage `uiStartTimeout = 2 * time.Minute` (variable, surchargeable par les tests). Une sortie du processus avant la disponibilité interrompt l'attente avec son code et la fin de sortie.
- Arrêt : `Stop` envoie `SIGTERM` au groupe, attend `teardownGrace`, puis `SIGKILL` au groupe, et attend la fin du processus. Il est appelé dans un `defer` posé dès que le processus est démarré, avant toute attente, de sorte que toutes les issues, annulation comprise, passent par lui.

*Alternative :* faire lancer l'application par l'agent. Écartée : l'agent peut oublier de l'arrêter, ou la tuer trop tôt ; un processus orphelin sur un port est exactement ce que le verrou doit empêcher.

### D4. Contrat de verdict et lignes de tâches

Le verdict reprend le principe de `conformityStep` (dernière ligne qui correspond à `^VERDICT:\s*(PASS|FAIL|SKIP)\s*$`, insensible à la casse) avec `SKIP` en plus, pour qu'un change sans effet visible dans l'interface (backend seul, documentation) ne soit pas bloqué par une étape qui n'a rien à observer. `SKIP` est refusé quand des tâches marquées non cochées subsistent : un agent paresseux ne peut pas s'en servir pour contourner ce que l'humain devrait faire à la main.

`TASK-VERIFIED: <texte>` : une ligne par tâche, comparée au texte (marqueur retiré, bords rognés, espaces internes inchangés) des tâches marquées non cochées de la branche. Une ligne sans correspondance est ignorée et tracée dans le rapport.

Le contrôle d'intégrité diffère de celui de la conformité : `git status --porcelain --untracked-files=no` avant et après, parce que l'application peut légitimement créer des fichiers non suivis (journaux, base locale) ; une modification de fichier suivi reste un échec. La base de comparaison est prise une fois l'application prête, après `uiStartCommand` (qui peut, lui, légitimement installer ou construire).

### D5. La plateforme coche, après le verdict, un commit par tâche

Après `PASS`, `tickVerifiedTasks(wt, change, verified []string)` relit le `tasks.md` de la branche, retient les tâches marquées non cochées dont le texte correspond, et appelle, pour chacune, la partie « écriture et commit » de `ToggleBranchTask`, extraite en `toggleTaskInWorktree(wt, change, index)` (sans la prise du verrou ni le contrôle du worker, déjà tenus par l'exécuteur). Le message de commit est celui des coches de branche (`Validate task N`). Le marqueur n'est pas touché par `openspec.ToggleTask`. En cas d'échec d'un commit, le fichier est restauré (comportement existant), l'étape échoue avec la raison, et les coches déjà commitées restent (elles sont indépendantes et réversibles par l'utilisateur).

Aucune coche sur `FAIL` ni `SKIP` : les tâches vérifiées d'une exécution qui échoue pourraient ne plus l'être après la correction, et une coche partielle ferait croire à un parcours terminé.

L'étape détient le verrou du change pendant les coches, comme pendant toute la vérification (`verification_busy` pour les coches manuelles, voir `verification-stage`).

### D6. Un run `verify` par étape, preuves à côté du journal

`runVerification` (de `verification-stage`) ouvre un run par étape, au lieu d'un run pour toute la vérification ; les marqueurs `verify_run_start` / `verify_run_end` portent `step`. `GET …/verification/report` choisit le run le plus récent (le plus récent est celui qui a décidé) ou, avec `?step=`, le plus récent de l'étape demandée.

Le dossier de preuves du run `ui` est `conversations/<ws>/<change>/verify/<ts>/`, créé avant l'agent et exposé par `OPENSP8C_VERIFY_ARTIFACTS`. Le nom du dossier est l'horodatage du run, le même que `<ts>.jsonl` : `listDir` ne liste que les `.jsonl`, donc le dossier n'apparaît pas comme un run, et la rétention, qui supprime `conversations/<ws>/<change>/`, l'emporte avec les journaux. `GET …/verification/artifacts/{run}/{name}` valide `run` (horodatage au format `runTimestampLayout`) et `name` (un simple nom de fichier, extension `png`, `jpg`, `jpeg` ou `webp`), joint les deux au dossier du change, vérifie par `filepath.Rel` que le chemin résolu reste dans ce dossier, refuse les liens symboliques, et répond avec le type de contenu déduit de l'extension, jamais de l'agent. Le listage applique les mêmes règles, plus 5 Mo par fichier et 50 fichiers au plus (ordre alphabétique).

### D7. `{port}` : validation et substitution au même endroit

`verification.ValidateBaseURL` remplace `:{port}` par `:0` avant `url.Parse` et refuse toute autre occurrence du jeton dans `uiBaseUrl` (donc `{port}://localhost` et `http://{port}` sont rejetés). `uiStartCommand` n'est pas validé au-delà de ce qui existe. Une fonction `verification.Substitute(s, port)` est la seule à remplacer le jeton, utilisée par `appRunner`. L'écran de Configuration affiche déjà `uiStartCommand` et `uiBaseUrl` : seul un texte d'aide mentionnant `{port}` est ajouté.

### D8. Ordre des étapes et issue

`verifySteps = []verifyStep{conformityStep{}, uiStep{}}`. `runVerification` s'arrête à la première étape échouée ; le marqueur passe à `passed` quand toutes les étapes activées ont réussi. Une relance (`rerun`) repart de la première étape activée, y compris la conformité déjà réussie : garder le résultat de la conformité d'une exécution à l'autre exigerait de décider de sa validité après des corrections, et la conformité est l'étape la moins chère.

## Risks / Trade-offs

- [Le worktree n'a pas les dépendances du projet (`node_modules`, build) : `uiStartCommand` peut échouer] → l'échec porte la fin de la sortie de l'application ; `OPENSP8C_WORKSPACE_PATH` permet à la commande de lier ou copier ce que le dépôt principal contient déjà ; la documentation recommande une commande qui installe ce qui manque. Délai de démarrage volontairement large (2 min).
- [Un agent déclare `PASS` sans avoir réellement navigué] → pas de garantie logicielle ; mitigations : lecture seule contrôlée, captures exigées par la consigne et affichées pour relecture, rapport complet conservé, et en `hitl-review` la revue humaine suit. Aucune coche automatique sans verdict `PASS` + déclaration explicite par tâche.
- [Port libre repris par un autre processus entre le choix et le lancement] → échec explicite de démarrage (sortie prématurée ou application non prête), relançable en un clic.
- [Le verrou global sérialise des workspaces sans lien] → assumé : la ressource est la machine. Les vérifications de conformité ne sont pas retardées.
- [Processus orphelins si le backend est tué pendant l'étape] → le groupe de processus est tué à l'annulation de contexte ; un `kill -9` du backend peut laisser l'application vivante. Hors périmètre ; un redémarrage choisit de toute façon un autre port.
- [Preuves volumineuses] → limites de taille et de nombre ; suppression avec les journaux du change.
- [Dépendance à `verification-stage` non encore appliqué] → les tâches commencent par vérifier la présence de `verifySteps`, de `verify_run_end` et de l'endpoint de rapport ; sinon appliquer `verification-stage` d'abord.

## Migration Plan

Aucune migration : l'étape `ui` n'est exécutée que lorsque `ui` est activée, ce qu'aucun réglage existant ne fait. Retour arrière : désactiver `ui` ; les dossiers de preuves restent inoffensifs et sont purgés avec les journaux.

## Open Questions

- Un contrôle préalable de la disponibilité d'un outil de navigation (par exemple `npx playwright --version`) avant de lancer l'agent permettrait d'échouer plus tôt et sans tokens. Ce n'est pas nécessaire pour livrer l'étape (l'agent échoue de toute façon), et peut s'ajouter plus tard sans changer les specs.
