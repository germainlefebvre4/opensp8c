# agent-run-detail Specification

## Purpose
Permet de suivre en direct l'activité d'un agent du pool et de consulter l'historique de ses exécutions, y compris celles dont le worker a disparu, terminées, mises en pause ou interrompues, depuis l'onglet Agents d'un workspace.

## Requirements

### Requirement: Persistance du run de chaque worker
Le backend SHALL persister chaque exécution d'un worker du pool comme un run de conversation de kind `pool`, associé au workspace et au change traité. Ce run SHALL contenir, dans l'ordre chronologique, les tours envoyés à l'agent, les lignes de sortie de l'agent et les erreurs du subprocess. Le run SHALL être ouvert au démarrage du subprocess de l'agent et SHALL être conservé après la fin du worker, quelle qu'en soit l'issue. Le backend SHALL y consigner un marqueur de début (worker, change, mode de délégation) et, lorsque le worker se termine, un marqueur de fin portant l'issue du run et, le cas échéant, la raison de blocage.

#### Scenario: Run créé au démarrage d'un worker
- **WHEN** un worker du pool démarre le subprocess de l'agent pour le change `add-user-auth`
- **THEN** un nouveau run de kind `pool` est créé pour ce change, avec l'horodatage de démarrage pour identifiant, et son marqueur de début est écrit

#### Scenario: Sortie de l'agent consignée au fil de l'eau
- **WHEN** l'agent produit une ligne de sortie ou un appel d'outil pendant l'exécution du worker
- **THEN** cette ligne est ajoutée au run sans attendre la fin du worker

#### Scenario: Run conservé après la fin du worker
- **WHEN** le worker se termine parce que le change est fusionné, passé en revue, mis en pause ou parce que le pool est arrêté
- **THEN** le run reste consultable et son marqueur de fin porte l'issue correspondante

#### Scenario: Échec de démarrage du subprocess
- **WHEN** le subprocess de l'agent ne peut pas démarrer pour un worker
- **THEN** un run est tout de même consigné avec l'issue `paused` et la raison de l'échec, afin que l'échec soit consultable dans l'historique

### Requirement: Issue d'un run
Chaque run de kind `pool` SHALL avoir une issue parmi : `running` (le worker est actif), `completed` (change finalisé en mode `full-autonomy`), `awaiting-review` (change transmis en revue en mode `hitl-review`), `paused` (worker bloqué, avec sa raison), `stopped` (arrêt du pool par l'utilisateur) ou `interrupted` (aucun marqueur de fin alors qu'aucun worker actif n'est associé à ce run, par exemple après l'arrêt brutal du backend).

#### Scenario: Run en cours
- **WHEN** un run possède un marqueur de début, aucun marqueur de fin, et est associé à un worker actif du pool
- **THEN** son issue est `running`

#### Scenario: Run interrompu par un arrêt brutal
- **WHEN** le backend redémarre et qu'un run ne possède aucun marqueur de fin
- **THEN** son issue est `interrupted`

#### Scenario: Run en pause avec raison
- **WHEN** un worker se met en pause après épuisement de ses tentatives de guérison
- **THEN** l'issue du run est `paused` et sa raison de blocage est celle du worker

### Requirement: Liste des runs récents d'un workspace
Le backend SHALL exposer `GET /api/workspaces/{id}/pool/runs` retournant les runs de kind `pool` du workspace, tous changes confondus, du plus récent au plus ancien et limités aux 50 plus récents. Pour chaque run, la réponse SHALL inclure le nom du change, l'identifiant de run (`ts`), l'identifiant du worker, l'issue, la raison de blocage lorsqu'elle existe, la date de début, la date de fin lorsque le run est terminé, et le nombre de lignes consignées. La réponse SHALL contenir un tableau vide, et non `null`, lorsqu'aucun run n'existe.

#### Scenario: Runs de plusieurs changes
- **WHEN** le workspace a des runs pool pour les changes `add-user-auth` et `fix-docs`
- **THEN** la réponse liste tous ces runs du plus récent au plus ancien, chacun associé à son change

#### Scenario: Aucun run
- **WHEN** aucun pool n'a jamais exécuté de worker sur ce workspace
- **THEN** la réponse contient un tableau vide sans erreur

#### Scenario: Plus de 50 runs
- **WHEN** le workspace compte plus de 50 runs pool
- **THEN** la réponse ne contient que les 50 plus récents

### Requirement: Lecture du détail d'un run
Le backend SHALL exposer `GET /api/workspaces/{id}/pool/runs/{change}/{ts}` retournant le détail d'un run : change, `ts`, worker, issue, raison de blocage, dates de début et de fin, et la liste chronologique des entrées d'activité du run. Ces entrées SHALL fusionner la narration de l'agent et les appels d'outils dérivés du run (avec durée, entrée et résultat tronqué de l'outil) avec les entrées d'activité non-agent du change (transitions de statut du worker, commits) dont l'horodatage est compris dans la période du run. Un run inconnu SHALL produire une réponse `404`.

#### Scenario: Run terminé
- **WHEN** le client demande le détail d'un run terminé
- **THEN** la réponse contient toutes ses entrées d'activité, sa date de fin et son issue finale

#### Scenario: Run en cours
- **WHEN** le client demande le détail d'un run dont le worker est actif
- **THEN** la réponse contient les entrées produites jusqu'à cet instant, une issue `running` et aucune date de fin

#### Scenario: Entrées d'un autre run du même change exclues
- **WHEN** le change a deux runs consécutifs et que le client demande le détail du premier
- **THEN** la réponse n'inclut aucune entrée non-agent horodatée dans la période du second run

#### Scenario: Run introuvable
- **WHEN** le client demande un run dont l'identifiant ne correspond à aucun run du change
- **THEN** la réponse est `404 Not Found`

### Requirement: Identifiant de run exposé sur chaque worker
Le backend SHALL exposer sur chaque worker actif, dans l'endpoint de statut du pool d'un workspace et dans la liste globale des pools, l'identifiant de son run en cours (`run_ts`), de sorte que le client puisse relier un worker à son run sans se fier à l'identifiant du worker, qui peut être réutilisé d'un run à l'autre.

#### Scenario: Deux runs successifs avec le même identifiant de worker
- **WHEN** le worker 1 termine le change `A`, puis le worker 1 est réattribué au change `B`
- **THEN** le worker expose un `run_ts` différent pour `A` et pour `B`, et chaque run reste consultable séparément

### Requirement: Onglet Agents interactif avec panneau de détail
L'onglet Agents d'un workspace SHALL permettre de sélectionner par clic une ligne de worker actif ou de run récent. La sélection SHALL ouvrir un panneau de détail latéral, à côté de la liste, qui reste visible pendant la consultation du détail afin de changer de sélection d'un clic. Le panneau SHALL afficher : le change, le worker, l'issue du run avec sa raison de blocage le cas échéant, la durée, une frise chronologique des entrées et la liste chronologique de ces entrées (narration de l'agent, appels d'outils avec durée, transitions de statut, commits). Un appel d'outil SHALL pouvoir être déplié pour afficher son entrée et son résultat. La sélection SHALL pouvoir être annulée pour fermer le panneau.

#### Scenario: Sélection d'un worker actif
- **WHEN** l'utilisateur clique sur la ligne du worker qui traite `add-user-auth`
- **THEN** le panneau de détail s'ouvre sur le run en cours de ce worker

#### Scenario: Sélection d'un run récent
- **WHEN** l'utilisateur clique sur un run terminé dans la section des runs récents
- **THEN** le panneau de détail affiche ce run avec son issue finale et sa date de fin

#### Scenario: Changement de sélection
- **WHEN** un panneau est ouvert sur un run et que l'utilisateur clique sur une autre ligne
- **THEN** le panneau affiche le run de la nouvelle sélection sans fermer ni recharger la liste

#### Scenario: Dépliage d'un appel d'outil
- **WHEN** l'utilisateur déplie un appel d'outil dans la liste d'activité
- **THEN** l'entrée de l'outil et son résultat (éventuellement tronqué) sont affichés

### Requirement: Sélecteur de run pour un change
Le panneau de détail SHALL proposer un sélecteur listant les runs du change du run affiché, du plus récent au plus ancien, avec pour chacun sa date de début et son issue, afin de passer d'une exécution à une autre du même change.

#### Scenario: Change avec plusieurs exécutions
- **WHEN** le change `add-user-auth` a été exécuté trois fois et que le panneau affiche la dernière
- **THEN** le sélecteur propose les trois runs et choisir un run plus ancien affiche son détail

#### Scenario: Change avec une seule exécution
- **WHEN** le change n'a été exécuté qu'une fois
- **THEN** le sélecteur ne propose que ce run

### Requirement: Section des runs récents
L'onglet Agents SHALL afficher, sous la liste des workers actifs, une section « Runs récents » listant les runs pool du workspace, y compris ceux dont le worker n'existe plus. Chaque ligne SHALL indiquer le change, la date de début, la durée, l'issue et, pour une issue `paused`, la raison de blocage. Lorsque aucun run n'existe, la section SHALL afficher un état vide.

#### Scenario: Run terminé visible après la disparition du worker
- **WHEN** un worker termine le change `fix-docs` et disparaît de la liste des workers actifs
- **THEN** la section des runs récents affiche ce run avec son issue

#### Scenario: Aucun run
- **WHEN** aucun run pool n'existe pour le workspace
- **THEN** la section des runs récents affiche un état vide

### Requirement: Suivi en direct d'un run actif
Pendant que le panneau de détail affiche un run dont l'issue est `running`, l'arrivée de nouvelles lignes dans ce run SHALL se refléter dans le panneau sans rechargement de la page, à un rythme d'au plus une mise à jour par seconde. Lorsque le run se termine pendant la consultation, le panneau SHALL afficher son issue finale sans action de l'utilisateur.

#### Scenario: Nouvel appel d'outil pendant la consultation
- **WHEN** le panneau est ouvert sur un run actif et que l'agent lance un nouvel appel d'outil
- **THEN** cet appel apparaît dans la liste d'activité et sur la frise sans rechargement

#### Scenario: Fin du run pendant la consultation
- **WHEN** le worker se termine alors que son run est affiché dans le panneau
- **THEN** le panneau affiche l'issue finale du run et cesse d'indiquer qu'il est en direct

#### Scenario: Panneau ouvert sur un run terminé
- **WHEN** le panneau affiche un run terminé
- **THEN** aucun indicateur de suivi en direct n'est affiché

### Requirement: Rafraîchissement automatique de l'onglet Agents
L'onglet Agents SHALL refléter le démarrage et l'arrêt d'un pool, les changements de statut de ses workers, l'apparition de nouveaux runs et la fin de runs, sans que l'utilisateur ait à recharger la page ni à avoir consulté le Kanban au préalable.

#### Scenario: Ouverture directe de l'onglet Agents
- **WHEN** l'utilisateur ouvre l'onglet Agents sans être passé par le Kanban et qu'un worker change de statut
- **THEN** la liste des workers et la section des runs récents reflètent ce changement

### Requirement: Panneau de détail en lecture seule
Le panneau de détail SHALL rester en lecture seule : il ne SHALL PAS proposer d'arrêter, relancer ou envoyer un message à un worker, ni d'agir sur le pool.

#### Scenario: Aucune action sur le worker
- **WHEN** le panneau de détail est ouvert sur un run actif
- **THEN** aucun bouton d'arrêt, de relance ni aucune zone de saisie n'est présent
