# Design

## Context

- Un worker (`internal/pool/worker.go`) lance **un seul subprocess d'agent** pour toute la boucle apply, validation, guérison. `runTurn` lit stdout ligne par ligne, n'en garde qu'un extrait dans `Worker.Activity`, et jette le reste. `StartSubprocess` reçoit `nil` comme `sessionLog`, et ce paramètre ne journalise de toute façon que **stderr** : les lignes `in` et `out` sont écrites par l'appelant (comme le fait `session/manager.go` pour le chat).
- Le `conversation.Store` sait déjà persister des runs JSONL par kind (`chat`, `ff`), les lister par change et les relire ; `activity.ParseConversationLines` en dérive narration et appels d'outils avec durée, et `GET /changes/{name}/activity` fusionne déjà tous les kinds d'un change.
- Les événements SSE (`watcher.Event`) n'ont pas de payload, seulement `Type`, `Name`, `Error`. `activity_appended` est émis par `activity.Store.Append`, pas par l'écriture d'un journal de conversation.
- L'identifiant d'un worker est recyclé et un worker terminé disparaît du statut. Le `ts` de `OpenRun` est la seule identité stable d'une exécution.
- `useWorkspaceLiveState` (SSE) n'est monté que par le Kanban ; `usePoolStatus` ne fait pas de polling.

Motivation : voir `proposal.md`. Comportements : voir `specs/agent-run-detail`.

## Goals / Non-Goals

**Goals:**
- Un journal complet et fiable de chaque exécution de worker, sans changer son déroulement.
- Un suivi en direct avec une latence d'environ une seconde, en réutilisant SSE + refetch plutôt qu'un nouveau transport.
- Réutiliser le maximum de l'existant : store, parseur, frise, `ToolCallRow`.

**Non-Goals:**
- Streaming token par token (texte qui s'écrit en direct) : écarté, voir Décision 2.
- Toute action sur un worker (arrêt individuel, relance, message) ; le panneau est en lecture seule.
- Nouveau réglage de rétention ou purge propre aux runs pool.
- Détail des runs de la vue globale de Configuration (`agent-pool-visibility` inchangé).
- Afficher les prompts envoyés à l'agent dans le panneau (ils sont journalisés, mais le parseur les ignore).

## Decisions

### 1. Un run = un fichier JSONL `pool/<ts>.jsonl`, avec marqueurs de début et de fin

Le worker ouvre `OpenRun(ws, change, "pool", ts)` juste avant `StartSubprocess`, l'enveloppe dans un `SessionLog`, et l'écrit à trois endroits : `in` dans `runTurn` avant d'écrire sur stdin du subprocess, `out` pour chaque ligne de stdout, et stderr via le paramètre existant `sessionLog`. Une ligne de marqueur de début (`dir: "meta"`, `{"type":"pool_run_start","worker_id":…,"delegation_mode":…}`) et une ligne de fin (`{"type":"pool_run_end","outcome":…,"reason":…}`) encadrent le run. L'écriture est **non bloquante pour le worker** : une erreur d'écriture est loguée, jamais propagée.

*Pourquoi les marqueurs dans le fichier plutôt qu'un fichier annexe* : un seul fichier par run, atomique par ligne (le `SessionLog` sérialise déjà les écritures), lisible en cas d'arrêt brutal. L'issue se déduit de la dernière ligne ; l'absence de marqueur de fin et de worker actif donne `interrupted` sans état supplémentaire.

*Alternatives* : un `.meta.json` à côté (deux fichiers à garder cohérents, pas d'atomicité) ; une base SQLite (nouvelle dépendance pour un besoin qu'un fichier couvre).

### 2. Live par « signal SSE léger + refetch », pas de flux dédié

`runTurn` appelle un `noteRunAppended(change)` après chaque ligne écrite. Il déclenche `watcher.Event{Type:"pool_run_appended", Name: change}` au plus une fois par seconde et par change, avec un envoi de rattrapage différé après la dernière ligne d'une rafale (trailing edge) et un envoi immédiat en fin de run. Le client invalide la requête du run affiché et de la liste des runs.

*Pourquoi* : c'est le choix « option A » de l'exploration. Latence acceptable (1 s), aucun nouveau transport, aucun état de connexion par worker, et le même chemin sert le live et l'historique. Pas de champ `ts` dans l'événement : le client sait déjà quel run il affiche, et invalide simplement les requêtes du change.

*Alternatives* : WebSocket ou SSE par worker avec lignes brutes (vrai temps réel mais nouveau transport, buffering, reconnexion et duplication du chemin de lecture) ; polling seul (charge inutile quand rien ne bouge). Si le besoin de token par token apparaît, il s'ajoute plus tard sans casser ce modèle.

### 3. Identité d'un run : `RunTS` sur `Worker`, exposé par `run_ts`

`Worker` gagne `RunTS` (le `ts` de `OpenRun`), sérialisé en `run_ts`. Les deux endpoints existants (`/pool/status`, `/api/pools`) l'exposent tels quels puisqu'ils sérialisent déjà `Worker`. Le client clé une sélection sur `(change, ts)`, jamais sur l'id de worker.

### 4. Détail d'un run côté serveur : entrées fusionnées dans la fenêtre du run

`GET /pool/runs/{change}/{ts}` charge le fichier via `Store.Load`, dérive les entrées avec `ParseConversationLines`, puis y ajoute les entrées de l'`activity.Store` du change comprises entre le début et la fin du run (ou « maintenant » pour un run en cours), puis trie par horodatage. On récupère ainsi les transitions `pool.worker_status` (testing, healing) et les commits git, ce qui donne l'histoire des tentatives de guérison sans journaliser la sortie de `go test` dans un format neuf.

Deux runs d'un même change ne se chevauchent pas (un seul worker par change), donc la fenêtre est sans ambiguïté.

Le parseur est étendu à un seul endroit : le résultat d'un appel d'outil est conservé, tronqué (4 Ko), dans `Meta["result"]`, pour alimenter le dépliage dans `ToolCallRow`. Cela bénéficie aussi à l'onglet Conversation. Le parseur ignore les lignes `dir: "meta"` (elles ne contiennent ni delta, ni `tool_use`, ni `tool_result`, donc rien à changer), ce qui est couvert par un test.

*Alternative* : renvoyer les lignes brutes et parser dans le navigateur avec `exploreChat.ts` (deux parseurs pour la même donnée, dérive assurée, et calcul des durées à dupliquer).

### 5. Liste des runs : parcours des dossiers `pool/` du workspace

`Store.ListKindAcrossChanges(wsID, "pool")` parcourt `conversations/<ws>/*/pool/*.jsonl` (hors `_explore`), lit la première et la dernière ligne de chaque fichier pour en tirer worker, issue et raison, trie par `ts` décroissant, et coupe à 50. Le nombre de lignes vient de `List` déjà en place. L'issue `running` est déterminée en croisant avec les workers actifs du `Manager` (même `run_ts`) ; sans marqueur de fin et sans worker actif : `interrupted`.

*Pourquoi pas un index* : le volume attendu est faible (runs par change, 50 max renvoyés) et le parcours de fichiers évite un second état à tenir cohérent. À revoir si la liste devient lente.

### 6. Écriture de l'issue de fin dans `runWorker`

Un `defer` unique dans `runWorker` écrit le marqueur de fin avec l'issue calculée à la sortie : `completed` (merge), `awaiting-review` (transition vers revue), `paused` (raison du worker), `stopped` (contexte annulé par `Stop`). Il s'exécute avant la suppression du worker de `activeWorkers`, puis émet le dernier `pool_run_appended`.

### 7. Synchronisation de l'état du worker

`Worker.Activity` et `Worker.Status` sont aujourd'hui écrits sans verrou. Les mutations passent par de petites méthodes du `Manager` qui prennent `m.mu` (et rappellent `broadcastLocked` quand c'est utile), y compris les mises à jour d'activité dans `runTurn`. `Status()` continue de copier sous le verrou. Un test avec `-race` couvre le scénario.

### 8. Frontend

- `AgentsPage` monte `useWorkspaceLiveState(workspaceId)` (ou une variante réduite) pour recevoir `pool_updated` et `pool_run_appended`, et garde l'état de sélection `{change, ts}` dans l'URL de l'onglet (paramètre de requête) afin qu'un rechargement conserve la sélection.
- La table des workers passe en lignes cliquables ; une section « Runs récents » utilise un nouveau hook (`usePoolRuns`, clé `['pool-runs', ws]`).
- Un composant `AgentRunPanel` (panneau latéral) utilise un hook `usePoolRun(ws, change, ts)` (clé `['pool-run', ws, change, ts]`) et réutilise `ActivityTimelineBar`, `getActivityColor` et le filtre par légende du `DetailPanel`, ainsi que `ToolCallRow` pour les appels d'outils.
- `pool_run_appended` invalide `['pool-run', ws, change]` et `['pool-runs', ws]` ; `pool_updated` continue d'invalider `['pool-status', ws]`.

## Risks / Trade-offs

- **Volume des journaux** : un run pool est plus long et plus bavard qu'un chat → la rétention existante (change archivé depuis TTL jours) borne la croissance, mais un change actif longtemps accumule des runs. Mitigation : la liste est plafonnée à 50 côté API ; un plafond de taille par run est renvoyé aux questions ouvertes.
- **Refetch complet à chaque signal** : un run long renvoie toutes ses entrées chaque seconde → Mitigation : le signal est limité à 1/s ; le client ne refetche que quand le panneau est ouvert sur un run `running` ; si la charge devient sensible, ajouter un paramètre `since` (offset de ligne) sans changer le contrat de base.
- **Fenêtre temporelle pour les entrées non-agent** : elle s'appuie sur l'horloge du serveur et sur l'absence de chevauchement de runs d'un même change → Mitigation : un test couvre deux runs successifs du même change.
- **Perte du journal sur échec disque** : l'exécution ne doit pas dépendre du journal, donc les erreurs d'écriture sont loguées et ignorées (spec) ; le run peut alors être incomplet sans que ce soit signalé à l'utilisateur.
- **Contenu sensible dans les journaux** : la sortie de l'agent peut contenir des secrets lus dans le worktree (variables d'environnement, fichiers). Ils sont déjà écrits dans les journaux `chat` et `ff` avec la même politique locale, et ce change n'ajoute pas de partage réseau ; les journaux restent sous `<config-dir>`, comme aujourd'hui.
- **Conflit avec `agent-role-model-settings`** : les deux modifient `worker.go`, `manager.go` et la signature du démarrage du subprocess → Mitigation : implémenter l'un puis rebaser l'autre ; la journalisation ne dépend pas du choix de modèle.

## Migration Plan

Aucune migration de données : les runs pool n'existent qu'à partir du déploiement et les pools déjà actifs au moment de la mise à jour n'ont pas de run pour leurs workers en cours. Un pool doit être redémarré pour que ses workers soient journalisés. Retour arrière : le retrait du code laisse des fichiers `pool/` inertes, purgés avec le change par la rétention existante.

## Open Questions

- Faut-il plafonner la taille d'un run (nombre de lignes ou octets) avec une troncature explicite ? Sans effet sur les specs tant que le plafond reste très au-dessus d'un run normal.
- Les prompts envoyés à l'agent (dont les erreurs de build renvoyées lors de la guérison) méritent-ils une entrée d'activité dédiée dans le panneau ? Ajoutable ensuite sans changer le format du journal.
