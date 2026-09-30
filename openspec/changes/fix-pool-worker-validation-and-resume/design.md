# Design

## Context

Voir `proposal.md` pour la motivation. État actuel du pool (`backend/internal/pool`) :

- `runValidation` est un stub : `go test ./...` avec `cmd.Dir = w.WorktreePath`. Son échec est renvoyé tel quel à l'agent par `invokeAgentHeal`, jusqu'à `max_attempts`.
- `WorktreeController.Provision` teste `branchExists == ""` sur la sortie standard de `git show-ref --verify --quiet`, qui n'écrit rien : la branche « n'existe » jamais, la branche `else` (réutilisation) est du code mort.
- `Manager.tick` filtre les changes candidates avec `activeWorkers` uniquement. `pauseWorker` copie le worker dans `pausedWorkers` puis `runWorker` le retire de `activeWorkers` : la change redevient candidate au tick suivant. `Stop()` vide `pausedWorkers`.
- Les réglages de pool (`size`, `delegationMode`, `maxAttempts`) vivent dans `preferences.json` : défauts globaux (`PoolSettings`) et surcharge par workspace (`PoolOverride`), résolus par `Preferences.ResolvePool(workspaceID)`, éditables via un formulaire partagé (`AgentPoolSettingsForm`) dans Configuration et Settings.
- Un worktree est un checkout git : les fichiers ignorés (`node_modules`) n'y existent pas tant que l'agent ou une commande ne les crée pas.

## Goals / Non-Goals

**Goals:**
- La validation s'exécute là où le projet le demande et couvre les projets à sous-répertoires (`backend/`, `frontend/`) comme les autres stacks, sans configuration obligatoire.
- Une erreur d'environnement ne consomme aucun tour d'agent et se voit dans l'UI.
- Toute reprise d'une change (pause, corrections, redémarrage) retrouve sa branche et son worktree.
- Une pause est un état stable : elle ne se défait que par une action explicite.

**Non-Goals:**
- Détecter d'autres stacks que Go et Node (Python, Rust, etc.) : elles passent par la commande configurée.
- Installer les dépendances (`npm ci`) automatiquement.
- Reprise automatique avec plafond de tentatives (écartée : option (b) de la discussion, voir décision 4).
- Persister les workers en pause au redémarrage du serveur : les snapshots restent en mémoire, comme aujourd'hui.
- Corriger l'activité affichée en JSON brut ni les tâches manuelles dans `tasks.md`.

## Decisions

### 1. Résolution de la commande de validation à l'exécution

`runValidation` résout la commande à chaque appel par `Preferences.ResolvePool(workspaceID).ValidationCommand` (surcharge workspace > défaut global > vide), puis, si vide, par l'auto-détection. Le champ est ajouté à `PoolSettings`, `PoolOverride` et `PoolPatch`, sur le modèle de `maxAttempts`, donc exposé sans nouvelle route par `GET /api/preferences` et l'API de settings de workspace, et édité par le formulaire partagé.

- *Pourquoi à l'exécution et pas dans `AgentPoolConfig` au `Start`* : la commande n'est pas un paramètre de lancement ponctuel ; la lire à chaque validation permet de la corriger pendant qu'un worker est en pause, puis de reprendre, sans redémarrer le pool.
- *Alternative écartée* : champ `validate` dans `config.yaml` par workspace. Ce fichier ne porte que l'identité des workspaces (nom, chemin) ; les réglages de pool sont déjà dans `preferences.json` avec héritage global/workspace.

### 2. Forme de la commande et exécution

- Commande configurée : chaîne exécutée par `sh -c` à la racine du worktree (permet `cd backend && go test ./... && cd ../frontend && npm test`). Une chaîne vide ou uniquement blanche est traitée comme absente.
- Auto-détection (`DetectValidationCommands(root)`) : examine la racine puis les sous-répertoires directs, hors répertoires cachés et `node_modules`, dans l'ordre alphabétique. `go.mod` donne `go test ./...` ; `package.json` avec `scripts.test` donne `npm test` **seulement si `node_modules` existe dans ce répertoire**. Chaque commande s'exécute dans son répertoire ; la sortie combinée de toutes est concaténée, et la validation échoue à la première commande en échec (les suivantes ne sont pas lancées).
- *Pourquoi conditionner Node à `node_modules`* : sans lui, `npm test` échoue sur `vitest: not found`, une erreur d'environnement qu'un tour de guérison ne peut pas résoudre. Le répertoire est ignoré et un message est journalisé ; l'utilisateur peut le couvrir via la commande configurée (`... && npm ci && npm test`).
- *Alternative écartée* : détecter uniquement à la racine. C'est exactement le bug observé sur `opensp8c`.

### 3. Classification des erreurs de validation

`runValidation` retourne une erreur typée `ValidationEnvError` (raison lisible) distincte de l'erreur de test. Sont « environnement » : aucune commande détectée ou configurée, exécutable introuvable (`exec.ErrNotFound` ou code 127 de `sh -c`), répertoire d'exécution absent. Tout autre échec d'une commande démarrée reste une erreur de test, renvoyée à l'agent.

Dans `runWorker`, une `ValidationEnvError` à la première validation ou après un tour de guérison appelle `pauseWorker` immédiatement, sans incrémenter `attempts`.

- *Pourquoi ne pas analyser la sortie de `go test`* : reconnaître `does not contain main module` par expression régulière serait fragile et propre à Go. Avec la sélection correcte du répertoire, cette erreur ne peut plus se produire ; la classification par code de sortie et erreur d'exécution est stable.
- *Compromis assumé* : une commande configurée qui échoue toujours pour une cause d'environnement non détectable (dépendance manquante) consommera ses `max_attempts` avant la pause.

### 4. Pause stable et reprise explicite (option (a))

- `Manager.tick` exclut de la distribution les changes présentes dans `pausedWorkers` (par `ActiveChange`), en plus des `activeWorkers`.
- Nouvelle méthode `Manager.ResumeWorker(workerID)` : sous le verrou, retire l'entrée de `pausedWorkers`, notifie, et le prochain tick (5 s au plus) redistribue la change à un worker libre. Elle retourne `ErrPoolNotRunning`, `ErrWorkerNotPaused` que le handler traduit en `409` et `404`.
- Route `POST /api/workspaces/{id}/pool/workers/{workerId}/resume` dans `PoolHandler`.
- `Stop()` continue de vider `pausedWorkers` : arrêter puis relancer le pool reste un moyen de tout reprendre.
- *Pourquoi ne pas démarrer le worker directement dans `ResumeWorker`* : garder `tick` comme unique point de distribution préserve l'ordre de priorité, les dépendances DAG et la limite `size`. Le délai de 5 s est acceptable.
- *Alternative écartée* : reprise automatique avec plafond (option (b)). Un défaut d'environnement provoquerait des relances en boucle ; le comportement observé (relance toutes les 5 s) en est la démonstration.
- L'action est placée dans le panneau d'état du Kanban (`AgentPoolModal`), qui porte déjà « Stop Pool ». La vue `agent-pool-visibility` reste en consultation seule.

### 5. `Provision` idempotent

- Existence de la branche : `git show-ref --verify --quiet refs/heads/<branche>` interprété par son **code de sortie** (0 existe, 1 absent, autre : erreur remontée). `runGit` expose déjà l'erreur enveloppant `exit status`; on ajoute une variante retournant le code de sortie.
- Si le répertoire du worktree existe et figure dans `git worktree list --porcelain` sur la bonne branche : réutilisé tel quel, sans `worktree add`. Sinon, si la branche existe : `git worktree add <chemin> <branche>` (après `git worktree prune` si l'entrée est périmée). Sinon : `git worktree add -b`.
- Aucune opération ne réinitialise ni ne nettoie l'arbre de travail, afin de préserver les modifications non commitées de l'agent.

### 6. Échec de provisionnement visible

Dans `runWorker`, l'erreur de `Provision` appelle `pauseWorker` avec une raison « Échec du provisionnement du worktree : … », au lieu de `return` seul. Combiné à la décision 4, la change n'est pas relancée en boucle.

## Risks / Trade-offs

- [Le frontend n'est pas validé sans `node_modules` dans le worktree] → Le journal signale le répertoire ignoré ; la documentation recommande une commande configurée incluant `npm ci`. Les tests Go, qui ont motivé le correctif, sont couverts.
- [Aucune commande détectée met en pause les dépôts sans tests, même docs-only] → Choix délibéré : fusionner en `full-autonomy` sans aucune validation est plus dangereux. La raison invite à configurer une commande (par exemple `true`).
- [Une pause n'est plus levée seule, donc une pause oubliée bloque la change] → La raison de blocage et le bouton « Reprendre » sont visibles dans le panneau ; l'arrêt du pool lève aussi les pauses.
- [`sh -c` dépend de la plate-forme] → Le backend cible Linux/macOS (Docker) ; aucune prise en charge de Windows n'existe dans le pool aujourd'hui.
- [Les snapshots en pause sont perdus au redémarrage du serveur] → Inchangé ; la change redevient candidate au démarrage suivant du pool et `Provision` retrouve sa branche.
- [Réutiliser un worktree sale peut mêler d'anciennes modifications à la reprise] → C'est le comportement voulu : l'agent reprend son travail, `/opsx:apply` repart de `tasks.md` (31/32 dans le cas observé).
