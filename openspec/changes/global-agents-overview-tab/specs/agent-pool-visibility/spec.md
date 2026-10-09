## MODIFIED Requirements

### Requirement: Onglet de navigation Agents
La vue de visibilité globale, tous workspaces confondus, des agents et pools d'agents SHALL être exposée dans un onglet « Agents » de la barre principale (entre « Projects » et « Configuration »), à la route `/agents-overview`, indépendante du workspace actuellement sélectionné. Elle SHALL NE PAS être affichée dans Configuration. Le libellé « Agents » est partagé avec l'onglet Agents du workspace (`/agents?workspace=<id>`), qui reste distinct.

#### Scenario: Accès à l'onglet depuis n'importe quel workspace
- **WHEN** l'utilisateur clique sur « Agents » dans la barre principale alors qu'il consultait le Kanban ou les Specs d'un workspace quelconque
- **THEN** l'application affiche la vue Agents globale à `/agents-overview`, listant tous les workspaces, indépendamment du workspace précédemment actif

#### Scenario: Vue absente de Configuration
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool
- **THEN** la liste des pools actifs tous workspaces confondus n'y est pas affichée

### Requirement: Liste des pools et workers actifs tous workspaces confondus
La vue Agents globale SHALL lister **tous** les workspaces, regroupés par workspace : d'abord ceux dont le pool est en cours d'exécution, puis ceux dont le pool est inactif, chaque groupe conservant l'ordre de la sidebar. Pour un pool actif, l'en-tête SHALL indiquer le nom du workspace, « Pool actif », le nombre de workers provisionnés (taille du pool), le nombre de workers actifs (assignés à un change, y compris en pause) et le mode de délégation. Pour chaque worker actif, la vue SHALL indiquer le nom du changement qu'il traite, son statut, un aperçu de l'activité en cours, sa durée d'exécution et la raison du blocage lorsque son statut est `paused`. Un worker actif correspond à un run actif : la vue SHALL se limiter aux runs actifs, sans historique des runs passés. Pour un workspace sans pool en cours, la vue SHALL afficher « Pool inactif » sans aucun compteur ni run. Si aucun workspace n'est configuré, la vue SHALL afficher l'état vide habituel invitant à ajouter un projet.

#### Scenario: Plusieurs pools actifs sur des workspaces différents
- **WHEN** un pool de taille 3 avec 2 workers actifs tourne sur le workspace `A` et un pool de taille 1 avec 1 worker actif tourne sur le workspace `B`
- **THEN** la vue affiche les deux workspaces avec « Pool actif », provisionnés 3/actifs 2 pour `A` et provisionnés 1/actifs 1 pour `B`, le mode de délégation, et le détail de leurs runs actifs (changement, statut, aperçu d'activité, durée)

#### Scenario: Aucun pool actif
- **WHEN** aucun pool n'est en cours d'exécution sur aucun workspace
- **THEN** la vue affiche chaque workspace avec « Pool inactif », sans compteurs ni tableau de runs, et conserve leurs boutons de navigation, au lieu d'un état vide global

#### Scenario: Ordre des workspaces
- **WHEN** la sidebar ordonne les workspaces `alpha`, `beta`, `gamma`, `delta` et que seuls `beta` et `delta` ont un pool actif
- **THEN** la vue les affiche dans l'ordre `beta`, `delta`, `alpha`, `gamma`

#### Scenario: Aucun run passé
- **WHEN** un workspace a des runs terminés mais aucun worker assigné
- **THEN** la vue n'affiche aucun de ces runs passés

#### Scenario: Mise à jour de la liste
- **WHEN** un pool démarre, s'arrête, ou qu'un de ses workers change de statut, d'activité ou de changement traité pendant que l'utilisateur consulte la vue
- **THEN** la vue reflète ce changement sans rechargement manuel de la page

#### Scenario: Worker en pause avec raison affichée
- **WHEN** un worker actif d'un pool passe au statut `paused`
- **THEN** la vue affiche, pour ce worker, la raison lisible de son blocage

### Requirement: Navigation vers le workspace depuis la liste
L'en-tête de chaque workspace de la vue Agents globale SHALL proposer deux boutons : « Kanban », qui ouvre le Kanban du workspace (`/?workspace=<id>`), et « Agents », qui ouvre l'onglet Agents du workspace (`/agents?workspace=<id>`). Un clic sur la ligne d'un run actif SHALL ouvrir sur place, dans la même page, le panneau de détail du run (sans quitter la vue), identifié dans l'URL par les paramètres `runWorkspace=<id>` et `run=<change>/<ts>`. Ce panneau SHALL proposer un lien vers `/agents?workspace=<id>&run=<change>/<ts>`. La vue SHALL rester en lecture seule : elle ne SHALL PAS permettre de démarrer, d'arrêter ou de reprendre un pool ou un worker.

#### Scenario: Navigation vers le Kanban d'un workspace listé
- **WHEN** l'utilisateur clique sur « Kanban » dans l'en-tête du workspace `A`
- **THEN** l'application affiche le Kanban de `A`

#### Scenario: Navigation vers l'onglet Agents d'un workspace listé
- **WHEN** il clique sur « Agents » dans l'en-tête de `A`
- **THEN** l'application affiche l'onglet Agents de `A`

#### Scenario: Détail d'un run sur place
- **WHEN** l'utilisateur clique sur la ligne du run actif `add-auth` du workspace `A`
- **THEN** le panneau de détail de ce run s'ouvre dans la vue Agents globale, l'URL contient `runWorkspace=A` et `run=add-auth/<ts>`, et la liste reste affichée

#### Scenario: Lien vers la page du workspace
- **WHEN** le panneau de détail d'un run du workspace `A` est ouvert et l'utilisateur clique sur le lien vers le workspace
- **THEN** l'application affiche `/agents?workspace=A&run=<change>/<ts>` avec ce run sélectionné

#### Scenario: Panneau à jour
- **WHEN** de nouvelles entrées sont ajoutées au run ouvert dans le panneau
- **THEN** le panneau les affiche sans rechargement manuel

## REMOVED Requirements

### Requirement: Capacité du pool et workers disponibles visibles sans worker assigné
**Reason**: Remplacée par « Workers disponibles visibles sans worker assigné », qui couvre la vue Agents globale à la place de la section Configuration > Agent Pool.

**Migration**: Voir l'exigence « Workers disponibles visibles sans worker assigné ».

## ADDED Requirements

### Requirement: Workers disponibles visibles sans worker assigné
La vue Agents globale et l'onglet Agents d'un workspace SHALL afficher pour tout pool actif sa capacité (workers provisionnés et actifs, sous la forme « X/Y workers actifs » dans l'onglet Agents du workspace) ainsi que son mode de délégation, y compris lorsqu'aucun worker n'est assigné à un change. Lorsqu'un pool est actif sans aucun worker assigné, la vue SHALL indiquer le nombre de workers disponibles (égal à la taille du pool) et qu'ils attendent un change prêt, au lieu de l'état vide « aucun agent actif ». Lorsqu'un pool est actif avec des workers assignés mais moins que sa taille, la vue SHALL indiquer sous le tableau le nombre de workers restant disponibles. L'état vide « aucun agent actif » SHALL être réservé, pour l'onglet Agents d'un workspace, au cas où aucun pool ne tourne sur ce workspace ; la vue globale affiche « Pool inactif » dans ce cas. Le nombre de workers disponibles SHALL être calculé comme la taille du pool moins le nombre de workers assignés.

#### Scenario: Pool actif sans worker assigné dans l'onglet Agents
- **WHEN** un pool de taille 3 vient d'être démarré sur le workspace `A` et aucun worker n'est encore assigné, et l'utilisateur ouvre l'onglet Agents de `A`
- **THEN** la vue affiche « 0/3 workers actifs », le mode de délégation, et « 3 workers disponibles, en attente d'un change prêt », et n'affiche pas « aucun agent actif »

#### Scenario: Pool actif sans worker assigné dans la vue globale
- **WHEN** un pool de taille 3 est actif sur le workspace `A` sans worker assigné et l'utilisateur ouvre la vue Agents globale
- **THEN** la section de `A` affiche « Pool actif », 3 workers provisionnés, 0 actif, le mode de délégation et « 3 workers disponibles » à la place d'un tableau vide

#### Scenario: Pool partiellement occupé
- **WHEN** un pool de taille 3 a 2 workers assignés à des changes
- **THEN** la vue affiche 2 workers actifs sur 3 provisionnés, le tableau des 2 runs, et « 1 worker disponible » sous le tableau

#### Scenario: Aucun pool actif sur le workspace
- **WHEN** aucun pool ne tourne sur le workspace affiché dans l'onglet Agents
- **THEN** la vue affiche l'état vide « aucun agent actif » sans indication de workers disponibles

#### Scenario: Mise à jour du compteur
- **WHEN** un worker est assigné à un change ou se libère pendant que l'utilisateur consulte la vue
- **THEN** les compteurs de workers actifs et « N workers disponibles » reflètent ce changement sans rechargement manuel
