# Proposal

## Why

L'onglet **Agents** d'un workspace liste les workers du pool, mais seulement avec une ligne de statut et un aperçu d'une seule chaîne de texte (`Worker.Activity`, tronquée à 200 caractères). Tout le reste de ce que produit l'agent (narration, appels d'outils, résultats) est lu sur stdout puis jeté : le subprocess du worker est démarré sans journal de session. Il est donc impossible de suivre ce qu'un agent fait en ce moment, et encore moins de comprendre a posteriori pourquoi un worker s'est mis en pause ou a échoué, car un worker terminé disparaît de la vue et l'identifiant d'un worker est recyclé d'un run à l'autre.

## What Changes

- Chaque exécution d'un worker du pool est **persistée** comme un run de conversation de kind `pool` (JSONL horodaté, même format que les runs `chat` et `ff`) : tours envoyés à l'agent, lignes de stdout, stderr. Le run est ouvert au démarrage du subprocess et fermé à la fin du worker, quelle qu'en soit l'issue (fusion, revue, pause, arrêt du pool).
- Le `ts` du run devient l'**identité stable** d'une exécution : il est exposé sur le worker (`run_ts`) dans l'endpoint de statut par workspace et dans la liste globale des pools.
- Un **signal SSE** `pool_run_appended` (limité à environ 1 par seconde et par change, portant le nom du change) est émis quand le journal d'un run reçoit de nouvelles lignes, pour que le panneau de détail se mette à jour en direct.
- Nouvel endpoint de **liste des runs pool du workspace**, tous changes confondus, avec pour chacun : change, `ts`, nombre de lignes, statut de fin (en cours, terminé, mis en pause, arrêté) et raison de blocage le cas échéant.
- L'onglet **Agents** du workspace devient interactif :
  - les lignes de workers sont cliquables et sélectionnent un run ;
  - une nouvelle section **« Runs récents »** liste les runs passés du workspace (workers disparus, terminés, en pause, arrêtés) ;
  - un **panneau de détail latéral** affiche le run sélectionné : frise chronologique, liste d'activité (narration, appels d'outils avec entrées et résultats, statuts, durées) et sélecteur de run pour le change concerné. Le panneau est identique pour un run actif (mis à jour en direct) et pour un run terminé.
- L'onglet Agents se **rafraîchit de lui-même** (montage du flux SSE du workspace), au lieu de dépendre du fait que le Kanban ait été ouvert avant.
- Les écritures concurrentes sur `Worker.Activity` et `Worker.Status` sont protégées par le verrou du manager.
- **Modification des exigences existantes** : l'onglet Agents n'est plus décrit comme une vue sans aucune interaction ni limitée aux workers actuellement actifs ; il reste sans contrôle de pool (ni démarrage, ni arrêt, ni action individuelle sur un worker). La vue globale de Configuration (`agent-pool-visibility`) reste inchangée.

## Capabilities

### New Capabilities
- `agent-run-detail`: persistance des runs des workers du pool, identité stable d'un run, flux live et lecture d'un run, liste des runs récents d'un workspace, et panneau de détail (frise, activité, appels d'outils, sélecteur de run) dans l'onglet Agents.

### Modified Capabilities
- `workspace-agent-pool-tab`: workers sélectionnables, section « Runs récents », panneau de détail, rafraîchissement automatique de l'onglet ; la vue reste sans contrôle de pool.
- `agent-pool-orchestrator`: le run d'un worker est journalisé sans jamais bloquer son exécution ; l'état d'un worker est lu et écrit sous verrou.
- `conversation-store`: nouveau kind `pool`, et liste des runs d'un kind pour l'ensemble d'un workspace.
- `workspace-events`: nouvel événement SSE `pool_run_appended`.

## Impact

- **Backend**
  - `internal/pool` : `Manager` reçoit le `conversation.Store` (déjà détenu par `session.Manager`) ; `runWorker` ouvre et ferme le run ; `runTurn` écrit les tours `in` et les lignes `out` dans le journal ; `Worker` gagne `RunTS` ; verrouillage de `Activity` et `Status`.
  - `internal/conversation` : liste des runs d'un kind pour un workspace.
  - `internal/api/handlers/pool.go` et `router.go` : endpoint de liste des runs pool ; lecture d'un run via l'endpoint existant `GET .../conversations/{kind}/{ts}`.
  - `internal/watcher` : événement `pool_run_appended`.
  - `internal/activity` : `ParseConversationLines` conserve en plus le résultat tronqué de chaque appel d'outil dans `Meta` ; il ignore les marqueurs de début et de fin de run.
- **Frontend**
  - `AgentsPage`, `usePoolStatus`, nouveaux hooks de runs récents et de run sélectionné, nouveau composant de panneau de détail réutilisant `ActivityTimelineBar`, `ToolCallRow` et les helpers de `exploreChat`.
  - `useWorkspaceLiveState` monté aussi dans l'onglet Agents.
  - Locales `en` et `fr` (`agents.json`).
- **Effet de bord voulu** : `GET /changes/{name}/activity` fusionnant déjà tous les runs de conversation du change, les runs pool apparaissent aussi dans l'onglet Conversation du `DetailPanel`, sans autre travail.
- **Données** : nouveaux fichiers sous `<config-dir>/conversations/<wsID>/<change>/pool/`. La rétention existante (change archivé depuis plus de TTL jours) s'applique telle quelle, aucun nouveau réglage.
- **Coordination** : le change en cours `agent-role-model-settings` touche aussi `internal/pool`, `AgentPoolModal` et `agent-pool-ui` ; ce change ne modifie pas les mêmes exigences, mais les deux touchent `worker.go` et `manager.go`, donc l'ordre d'implémentation est à soigner.
