# Agent Pool UI Specification

## Purpose

Fournit les contrôles Kanban permettant de configurer, lancer, suivre et réviser le travail de l'Agent Pool sur un workspace.

## Requirements

### Requirement: Modal de configuration et de lancement du pool
L'interface utilisateur SHALL inclure, dans le Kanban de chaque workspace, un bouton de lancement d'action qui ouvre une modale de configuration du Pool d'Agents pour ce workspace. Cette modale SHALL permettre à l'utilisateur d'ajuster le nombre de workers parallèles (de 1 à 5) et de sélectionner le Mode de Délégation (`full-autonomy` ou `hitl-review`). L'état affiché par ce bouton (actif ou inactif) SHALL toujours refléter le statut réel du pool de ce workspace tel que connu du backend, y compris juste après le chargement de la page ou un changement de workspace, plutôt qu'un état mémorisé localement dans l'interface. Lorsqu'un pool est déjà actif pour ce workspace, ce même bouton SHALL ouvrir le panneau d'état du pool (voir "Panneau d'état du pool actif") au lieu de la modale de configuration.

#### Scenario: Ouverture et configuration de la modale
- **WHEN** l'utilisateur clique sur le bouton "Lancer le Pool" alors qu'aucun pool n'est actif sur ce workspace
- **THEN** une modale s'affiche avec un sélecteur numérique pour la taille du pool, un commutateur pour le mode de délégation, et un bouton de confirmation "Démarrer"

#### Scenario: Validation et lancement du pool
- **WHEN** l'utilisateur clique sur "Démarrer" dans la modale de configuration
- **THEN** l'application envoie les paramètres au backend via l'API pour ce workspace, ferme la modale, et affiche un indicateur visuel de pool actif dans l'en-tête du Kanban de ce workspace

#### Scenario: Statut du bouton synchronisé au chargement de la page
- **WHEN** l'utilisateur ouvre ou recharge le Kanban d'un workspace pour lequel un pool est déjà actif côté backend
- **THEN** le bouton affiche immédiatement l'état "pool actif" pour ce workspace, sans nécessiter d'action de l'utilisateur

#### Scenario: Statut du bouton correct après changement de workspace
- **WHEN** l'utilisateur navigue du Kanban du workspace `A` (pool actif) vers le Kanban du workspace `B` (aucun pool actif)
- **THEN** le bouton du workspace `B` affiche l'état "aucun pool actif", indépendamment de l'état affiché précédemment pour le workspace `A`

#### Scenario: Clic sur le bouton pendant qu'un pool tourne déjà
- **WHEN** l'utilisateur clique sur le bouton d'en-tête alors qu'un pool est déjà en cours d'exécution pour ce workspace
- **THEN** le panneau d'état du pool s'ouvre à la place de la modale de configuration ; la modale de configuration ne s'affiche pas

### Requirement: Synchronisation de l'état du bouton avec l'état réel du backend
L'indicateur "pool actif/inactif" affiché sur le bouton d'en-tête du Kanban SHALL refléter l'état réel du backend (`GET /workspaces/{id}/pool/status`) et non un état local purement optimiste. Cet état SHALL être récupéré au chargement du Kanban et SHALL rester à jour tant que la page reste ouverte, sans que l'utilisateur ait besoin de recharger la page.

#### Scenario: Rechargement de la page pendant qu'un pool tourne
- **WHEN** un pool est en cours d'exécution côté backend et que l'utilisateur recharge la page du Kanban
- **THEN** le bouton d'en-tête affiche immédiatement l'état "pool actif" (et non l'état par défaut "Lancer le Pool"), sans action supplémentaire de l'utilisateur

#### Scenario: Arrêt du pool déclenché depuis une autre session
- **WHEN** le pool est arrêté (par exemple depuis une autre fenêtre ou session ouverte sur le même workspace) pendant que l'utilisateur consulte le Kanban
- **THEN** le bouton d'en-tête et le panneau d'état, si ouvert, reflètent l'arrêt du pool sans que l'utilisateur ait à rafraîchir manuellement la page

### Requirement: Panneau d'état du pool actif
Lorsqu'un pool est en cours d'exécution, l'utilisateur SHALL pouvoir consulter un panneau listant chaque worker actif avec le change auquel il est assigné, son statut (`idle`, `working`, `testing`, `healing`, ou `paused`), et un aperçu de l'activité en cours de l'agent (dernière sortie produite par sa session réelle). Ce panneau SHALL exposer une action "Stop Pool" qui arrête l'ensemble du pool. Ce panneau ne fournit pas d'action d'arrêt individuelle par worker.

#### Scenario: Affichage de la liste des workers actifs
- **WHEN** l'utilisateur ouvre le panneau d'état alors que 2 workers sur 3 sont assignés à des changes
- **THEN** le panneau affiche les 2 workers actifs avec le nom de leur change assigné et leur statut courant, et indique que le 3e worker est disponible (idle ou non affiché comme assigné)

#### Scenario: Arrêt du pool depuis le panneau
- **WHEN** l'utilisateur clique sur "Stop Pool" dans le panneau d'état
- **THEN** l'application envoie la requête d'arrêt au backend, le panneau se ferme, et le bouton d'en-tête repasse à l'état "Lancer le Pool"

#### Scenario: Aperçu de l'activité en cours d'un worker
- **WHEN** un worker est au statut `working` sur le changement `add-user-auth` et que sa session agent produit de la sortie
- **THEN** le panneau affiche, pour ce worker, un aperçu texte de la dernière activité de l'agent, mis à jour au fil de l'exécution

### Requirement: Modale de lancement pré-remplie avec la configuration résolue
La modale de configuration et de lancement du pool SHALL être pré-remplie avec la configuration résolue du workspace (défauts de Configuration et surcharges du workspace) pour la taille du pool, le mode de délégation et le nombre de tentatives. L'utilisateur SHALL pouvoir ajuster ces valeurs pour le lancement en cours sans modifier les valeurs enregistrées.

#### Scenario: Pré-remplissage avec une surcharge
- **WHEN** le workspace surcharge la taille du pool à 4 et que l'utilisateur ouvre la modale de lancement
- **THEN** le sélecteur de taille affiche 4 et le mode affiché est celui résolu pour ce workspace

#### Scenario: Ajustement ponctuel
- **WHEN** l'utilisateur change la taille à 2 dans la modale et clique sur « Démarrer »
- **THEN** le pool démarre avec une taille de 2
- **THEN** la surcharge enregistrée du workspace reste à 4

#### Scenario: Workspace sans surcharge
- **WHEN** le workspace n'a aucune surcharge et que Configuration fixe la taille par défaut à 2
- **THEN** la modale affiche une taille de 2

### Requirement: Capacité du pool visible sur le bouton d'en-tête
Lorsqu'un pool est actif pour le workspace affiché, le bouton d'en-tête du Kanban SHALL indiquer la capacité du pool sous la forme « actifs/taille », où « actifs » est le nombre de workers actuellement assignés à un change (y compris les workers en pause) et « taille » la taille configurée du pool. Cette indication SHALL être visible sans ouvrir le panneau d'état, y compris lorsque aucun worker n'est encore assigné (« 0/taille »). Lorsqu'aucun pool n'est actif, le bouton SHALL n'afficher aucune capacité. Cette indication SHALL rester à jour tant que la page est ouverte, sans rechargement manuel.

#### Scenario: Pool qui vient de démarrer sans worker assigné
- **WHEN** l'utilisateur démarre un pool de taille 3 sur un workspace et qu'aucun worker n'est encore assigné à un change
- **THEN** le bouton d'en-tête affiche l'état « pool actif » accompagné de la capacité « 0/3 »

#### Scenario: Capacité mise à jour lorsqu'un worker est assigné
- **WHEN** un worker est assigné à un change sur un pool actif de taille 3 alors que l'utilisateur consulte le Kanban
- **THEN** le bouton d'en-tête affiche « 1/3 » sans que l'utilisateur ait à recharger la page

#### Scenario: Aucun pool actif
- **WHEN** aucun pool n'est actif sur le workspace affiché
- **THEN** le bouton d'en-tête affiche « Lancer le Pool » sans indication de capacité

### Requirement: Action « Reprendre » sur les workers en pause
Le panneau d'état du pool actif SHALL afficher, pour chaque worker au statut `paused`, sa raison de blocage lisible et deux boutons : « Reprendre » (« Resume » en anglais) et « Reprendre en finalisant » (« Resume and finalize » en anglais). Un clic sur « Reprendre » SHALL demander au backend la reprise de ce seul worker, sans arrêter le pool ni interrompre les autres workers. Un clic sur « Reprendre en finalisant » SHALL demander la même reprise avec `finalize_only`. Le bouton « Reprendre en finalisant » SHALL être désactivé tant que le `tasks.md` du worktree du change n'est pas entièrement coché, et SHALL alors indiquer (texte ou info-bulle) le nombre de tâches restantes. Pendant une requête, les deux boutons du worker SHALL être désactivés pour éviter une double demande. Si la reprise échoue, le panneau SHALL afficher l'erreur retournée par le backend et conserver le worker en pause. Les boutons SHALL NOT apparaître pour les workers dans un autre statut.

#### Scenario: Bouton visible sur un worker en pause
- **WHEN** l'utilisateur ouvre le panneau d'état alors que le worker 1, sur le changement `add-user-auth`, est `paused`
- **THEN** la ligne de ce worker affiche sa raison de blocage et les boutons « Reprendre » et « Reprendre en finalisant »

#### Scenario: Reprise d'un worker
- **WHEN** l'utilisateur clique sur « Reprendre » pour le worker en pause du changement `add-user-auth`
- **THEN** l'application envoie la demande de reprise pour ce worker, le panneau reste ouvert, et la ligne reflète le nouveau statut du worker dès sa reprise par le pool, sans rechargement de la page

#### Scenario: Reprise en finalisant disponible
- **WHEN** le change du worker en pause a toutes ses tâches cochées dans son worktree et que l'utilisateur clique sur « Reprendre en finalisant »
- **THEN** l'application envoie la demande de reprise avec `finalize_only` pour ce worker et la ligne reflète le nouveau statut dès sa reprise par le pool

#### Scenario: Reprise en finalisant indisponible
- **WHEN** le change du worker en pause a encore 1 tâche non cochée dans son worktree
- **THEN** le bouton « Reprendre en finalisant » est désactivé et indique qu'il reste 1 tâche, tandis que « Reprendre » reste actif

#### Scenario: Aucun bouton hors pause
- **WHEN** un worker est au statut `working`, `testing` ou `healing`
- **THEN** sa ligne n'affiche aucun bouton de reprise

#### Scenario: Échec de la reprise
- **WHEN** le backend refuse la reprise, par exemple parce que le worker n'est plus en pause
- **THEN** le panneau affiche le message d'erreur et les boutons redeviennent actifs
