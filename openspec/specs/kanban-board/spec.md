## Purpose

Spec du Kanban Board : colonnes, cartes de changement, ouverture de l'ExplorePanel, mise en page plein écran, et rafraîchissement automatique via SSE.

## Requirements

### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher les changements OpenSpec répartis en six slots horizontaux d'égale largeur : **To Explore**, **Ready**, **To Do**, **In Progress**, **To Review**, et **Done/Archived**. Le slot **Done/Archived** contient verticalement la colonne Done (en haut, `flex-1 min-h-0`, prioritaire sur l'espace vertical) et la colonne Archived (en bas, hauteur plafonnée à 40 % du slot via `max-h`, avec scroll interne). Les colonnes actives lisent depuis `openspec/changes/` (hors `archive/`). La colonne Archived lit depuis `openspec/changes/archive/` via un endpoint dédié. L'endpoint `/changes` SHALL inclure les champs `days_since_activity` (int), `is_stale` (bool), et `tags` (objet optionnel `{ type, complexity, components[] }`) pour chaque change actif. Pour un change dont un worker du pool est actif (y compris en pause), la progression (`tasks_done`, `tasks_total`) et la colonne SHALL être dérivées du `tasks.md` du worktree du worker, et non de celui du dépôt principal ; la colonne SHALL alors être plafonnée à **In Progress**. Un changement portant un marqueur de revue (voir `change-review-state`) SHALL être affiché dans la colonne **To Review**, quel que soit l'avancement de ses tasks dans le workspace principal ; ce statut prime sur la dérivation depuis le worktree.

#### Scenario: Chargement du Kanban avec changements archivés
- **WHEN** l'utilisateur ouvre le Kanban Board ou change de workspace actif
- **THEN** l'application charge les changements actifs depuis `/workspaces/{id}/changes` et les changements archivés depuis `/workspaces/{id}/archived-changes`, et les affiche dans leurs colonnes respectives — Done en haut du slot (priorité flex), Archived en bas (hauteur plafonnée), et To Review entre In Progress et Done/Archived

#### Scenario: Réponse API avec champs stale et tags
- **WHEN** l'endpoint `/workspaces/{id}/changes` retourne la liste des changes
- **THEN** chaque change inclut `days_since_activity` (entier, -1 si pas de tasks.md), `is_stale` (booléen), et `tags` (objet optionnel ou `null`)

#### Scenario: Rétrécissement vertical — Done conserve son espace
- **WHEN** la hauteur disponible du Kanban diminue (bottom panel ouvert, fenêtre réduite)
- **THEN** la colonne Archived se compresse en premier (dans la limite de son plafond), Done conserve l'espace résiduel disponible

#### Scenario: Chargement du Kanban sans changements archivés
- **WHEN** le répertoire `openspec/changes/archive/` est vide ou absent
- **THEN** la colonne Archived est affichée vide en bas du slot Done, sans erreur

#### Scenario: Changement sans tasks.md
- **WHEN** un changement n'a pas de fichier `tasks.md` (ou tasks_total == 0)
- **THEN** il est affiché dans la colonne **To Explore**

#### Scenario: Changement avec tasks non démarrées et non lancé
- **WHEN** un changement a un `tasks.md` avec des tasks mais aucune cochée (`tasks_done == 0`), et n'est pas marqué "lancé" (voir `kanban-ready-column`)
- **THEN** il est affiché dans la colonne **Ready**

#### Scenario: Changement avec tasks non démarrées et lancé
- **WHEN** un changement a un `tasks.md` avec des tasks mais aucune cochée (`tasks_done == 0`), et est marqué "lancé"
- **THEN** il est affiché dans la colonne **To Do**

#### Scenario: Changement partiellement complété
- **WHEN** un changement a au moins une task cochée mais pas toutes
- **THEN** il est affiché dans la colonne **In Progress**

#### Scenario: Changement entièrement complété en mode Autonomie Totale
- **WHEN** toutes les tasks d'un changement sont cochées ET qu'aucun worker du pool n'est actif sur ce change (merge effectué en mode `full-autonomy`, ou changement hors exécution du pool)
- **THEN** il est affiché dans la colonne **Done**

#### Scenario: Changement entièrement complété en mode Review Humaine
- **WHEN** un worker du pool d'agents exécuté en mode `hitl-review` termine un changement avec l'issue `awaiting-review` (travail committé dans `feature/<change>`), alors que le `tasks.md` du workspace principal n'est pas encore coché
- **THEN** il est affiché dans la colonne **To Review** (et non dans **To Do**) et n'est déplacé vers **Done** qu'après approbation de l'utilisateur et fusion de sa branche

#### Scenario: Changement en revue après redémarrage
- **WHEN** le pool d'agents ou le backend est arrêté puis redémarré alors qu'un changement est en attente de revue
- **THEN** il reste affiché dans la colonne **To Review**

#### Scenario: Progression en direct pendant le run d'un worker
- **WHEN** un worker est actif sur un change et que l'agent a coché 2 tâches sur 5 dans le `tasks.md` du worktree, alors que celui du dépôt principal n'en a aucune cochée
- **THEN** le change est affiché dans la colonne **In Progress** avec `tasks_done = 2` et `tasks_total = 5`, et son badge worker reste visible

#### Scenario: Change entièrement coché dans le worktree avant le merge
- **WHEN** un worker est actif sur un change dont toutes les tâches sont cochées dans le `tasks.md` du worktree, la validation ou le merge n'étant pas terminé
- **THEN** le change reste dans la colonne **In Progress** avec une progression de 100 %, et ne passe à **Done** qu'une fois le worker libéré après le merge

#### Scenario: Worker actif sans tâche cochée dans le worktree
- **WHEN** un worker est actif sur un change lancé dont aucune tâche n'est cochée dans le worktree
- **THEN** le change est affiché dans la colonne **To Do** avec son badge worker

#### Scenario: Worktree sans tasks.md exploitable
- **WHEN** un worker est actif sur un change mais que le worktree n'est pas encore provisionné, ou que son `tasks.md` est absent ou ne contient aucune tâche
- **THEN** la progression et la colonne sont dérivées du `tasks.md` du dépôt principal

#### Scenario: Fin du worker sans merge
- **WHEN** un worker est annulé ou libéré sans que son travail ait été fusionné
- **THEN** la progression et la colonne du change redeviennent celles du `tasks.md` du dépôt principal

### Requirement: Colonne Archived — affichage paginé, lecture seule et collapsible
La colonne **Archived** SHALL afficher les changements par ordre antéchronologique (les plus récents en premier), limités à **3** par défaut. Un bouton "Afficher plus" SHALL permettre d'en charger 3 supplémentaires à chaque clic. La colonne Archived SHALL exposer un bouton **collapse/expand** (chevron) dans son header, permettant de masquer entièrement la liste de cartes tout en conservant le header visible. L'état collapsed est local et non persisté. Les cartes de la colonne Archived SHALL avoir un traitement visuel atténué (teintes slate/grises). Aucune action n'est disponible sur les cartes archivées — elles sont en lecture seule.

#### Scenario: Colonne Archived avec plus de 3 changements
- **WHEN** le répertoire archive contient plus de 3 changements
- **THEN** les 3 plus récents sont affichés et un bouton "Afficher plus" est visible en bas de la colonne

#### Scenario: Clic sur "Afficher plus"
- **WHEN** l'utilisateur clique sur "Afficher plus"
- **THEN** 3 changements supplémentaires s'ajoutent à l'affichage, et le bouton disparaît si tous les changements sont désormais visibles

#### Scenario: Colonne Archived avec 3 changements ou moins
- **WHEN** le répertoire archive contient 3 changements ou moins
- **THEN** tous sont affichés, sans bouton "Afficher plus"

#### Scenario: Collapse de la colonne Archived
- **WHEN** l'utilisateur clique sur le chevron collapse dans le header de la colonne Archived
- **THEN** la liste de cartes et le bouton "Afficher plus" sont masqués, seul le header reste visible, le chevron indique l'état collapsed

#### Scenario: Expand de la colonne Archived
- **WHEN** l'utilisateur clique sur le chevron expand dans le header de la colonne Archived (état collapsed)
- **THEN** la liste de cartes réapparaît avec le nombre de cartes visibles précédent

#### Scenario: Clic sur une carte archivée
- **WHEN** l'utilisateur clique sur une carte dans la colonne Archived
- **THEN** le DetailPanel s'ouvre en lecture seule, sans bouton d'action d'archivage

### Requirement: Séparateur visuel entre Done et Archived
Un séparateur visuel horizontal SHALL être rendu entre les colonnes **Done** et **Archived** à l'intérieur du slot partagé, pour signaler la frontière entre changements actifs et archivés.

#### Scenario: Rendu du séparateur horizontal
- **WHEN** le Kanban est affiché
- **THEN** une ligne horizontale tenue sépare visuellement la section Done de la section Archived dans le même slot de colonne

### Requirement: Afficher la carte d'un changement
Chaque changement SHALL être représenté par une carte épurée affichant : le nom du changement, la progression des tasks (barre de progression + compteur), et — lorsque les tags sont disponibles — un badge de type applicatif et un indicateur de complexité (points sur 5). Les cartes en colonne **Done** SHALL afficher une action rapide **"Sync & Archive"** au survol. Les cartes en colonne **Archived** n'affichent aucune action. Les cartes en colonnes **To Explore**, **Ready**, **To Do**, **In Progress** et **To Review** SHALL être draggables selon les transitions autorisées. Les cartes en colonnes **Done** et **Archived** SHALL être non-draggables. Quand un subprocess ff est actif pour un changement, sa carte SHALL afficher un spinner à la place du contenu normal et le drag SHALL être désactivé pour cette carte. En cas d'erreur ff (`ff_failed`), la carte SHALL afficher un indicateur d'erreur et le drag est réactivé. Si un changement dans la colonne **Ready** ou **To Do** posséde une exploration fantôme active du même nom, il SHALL être affiché comme un change brouillon (draft) visuellement distinct, incluant un bouton d'action explicite "Figer". Lorsque `worker_active` vaut `true` pour un changement, sa carte SHALL afficher un badge indiquant qu'un worker du pool d'agents est actuellement assigné à ce changement, positionné sur la même ligne que le compteur de tâches, aux côtés du badge stale s'il est également présent, ainsi qu'un bouton d'action d'arrêt rapide permettant d'interrompre le worker actif et de rétrograder la tâche vers la colonne Ready.

#### Scenario: Carte sans tasks.md
- **WHEN** le changement n'a pas encore de fichier `tasks.md`
- **THEN** la progression est affichée comme "0 / 0 tasks" sans erreur

#### Scenario: Carte avec tasks.md
- **WHEN** le changement a un fichier `tasks.md` contenant des items `- [ ]` et `- [x]`
- **THEN** la carte affiche "N / M tasks" où N est le nombre de `[x]` et M le total

#### Scenario: Carte avec tags disponibles
- **WHEN** le changement possède une section `tags` avec `type` et `complexity`
- **THEN** la carte affiche un badge compact pour le type (ex. "🖥 frontend") et un indicateur de complexité (ex. "●●○○○") entre le nom et la barre de progression

#### Scenario: Carte sans tags
- **WHEN** le changement ne possède pas de section `tags`
- **THEN** la carte s'affiche normalement sans badge ni indicateur de complexité, sans erreur

#### Scenario: Survol d'une carte Done
- **WHEN** l'utilisateur survole une carte dans la colonne Done
- **THEN** un bouton "Sync & Archive" apparaît sur la carte

#### Scenario: Carte en colonne Archived
- **WHEN** la carte est dans la colonne Archived
- **THEN** aucun bouton d'action n'est visible, au survol ou autrement

#### Scenario: Carte avec ff actif
- **WHEN** l'événement SSE `ff_started` est reçu pour un changement
- **THEN** la carte de ce changement affiche uniquement un spinner, sans nom ni progression, et le drag est désactivé

#### Scenario: Carte avec ff échoué
- **WHEN** l'événement SSE `ff_failed` est reçu pour un changement
- **THEN** la carte affiche un indicateur d'erreur (icône ou texte "ff échoué") et le drag est réactivé

#### Scenario: Carte Done non-draggable
- **WHEN** l'utilisateur tente de drag une carte en colonne Done
- **THEN** la carte ne peut pas être saisie (drag désactivé sur cette carte)

#### Scenario: Carte Archived non-draggable
- **WHEN** l'utilisateur tente de drag une carte en colonne Archived
- **THEN** la carte ne peut pas être saisie (drag désactivé sur cette carte)

#### Scenario: Change "brouillon" (unsolidified draft) dans la colonne Ready ou To Do
- **WHEN** un changement est dans la colonne "Ready" ou "To Do" ET qu'un ghost card actif possède le même nom dans la liste des changements
- **THEN** la carte s'affiche avec une opacité légèrement atténuée, une bordure pointillée, un badge violet "projet" et un bouton explicite "Figer" qui permet de consolider le change

#### Scenario: Carte avec un worker du pool d'agents actif
- **WHEN** un changement a `worker_active = true`
- **THEN** sa carte affiche un badge signalant qu'un worker du pool d'agents y est actuellement assigné, visible que la carte affiche ou non un badge stale

#### Scenario: Carte sans worker actif
- **WHEN** un changement a `worker_active = false` ou absent
- **THEN** aucun badge de worker actif n'est affiché sur la carte

#### Scenario: Bouton d'arrêt de worker sur la carte
- **WHEN** un changement a `worker_active = true`
- **THEN** sa carte affiche un bouton d'action permettant d'arrêter le worker assigné à ce changement

#### Scenario: Clic sur le bouton d'arrêt de worker
- **WHEN** l'utilisateur clique sur le bouton d'arrêt du worker sur la carte et confirme l'action
- **THEN** l'application envoie une requête de rétrogradation forcée au backend, le worker est interrompu, son worktree est préservé, et la carte se déplace en colonne "Ready"

#### Scenario: Carte To Review draggable
- **WHEN** l'utilisateur saisit une carte de la colonne To Review
- **THEN** la carte peut être déplacée vers **Done** ou **In Progress** uniquement (voir `kanban-drag-drop`)

### Requirement: Ouvrir l'ExplorePanel au clic sur une carte To Explore
L'utilisateur SHALL pouvoir cliquer sur une carte dans la colonne **To Explore** pour ouvrir le bottom panel de conversation. Le panel SHALL s'afficher sous les colonnes Kanban (layout flex-col), sans masquer ni comprimer les colonnes. La carte entière est la zone cliquable.

#### Scenario: Clic sur carte en colonne To Explore
- **WHEN** l'utilisateur clique sur une carte dans la colonne **To Explore**
- **THEN** le bottom panel de chat s'ouvre sous les colonnes Kanban, les colonnes restant visibles et interactibles au-dessus

#### Scenario: Bottom panel ne masque pas les colonnes
- **WHEN** le bottom panel est ouvert
- **THEN** les colonnes Kanban restent visibles et interactibles dans la partie supérieure de l'écran

### Requirement: Colonnes Kanban pleine hauteur
Les colonnes Kanban SHALL occuper toute la hauteur disponible de la zone de contenu, indépendamment du nombre de cartes qu'elles contiennent.

#### Scenario: Colonne vide
- **WHEN** une colonne ne contient aucune carte
- **THEN** la colonne s'étend sur toute la hauteur disponible et reste une zone de dépôt valide

#### Scenario: Colonnes de hauteurs différentes
- **WHEN** les colonnes contiennent des nombres différents de cartes
- **THEN** toutes les colonnes ont la même hauteur (celle de la colonne la plus haute ou de la zone disponible)

### Requirement: Application pleine largeur avec colonnes auto-adaptées
Le Kanban Board SHALL occuper toute la largeur disponible de la zone de contenu, que le DetailPanel soit ouvert ou non. Lorsque le DetailPanel est ouvert, les colonnes SHALL partager l'espace horizontal avec lui selon un layout flex : colonnes en `flex: 1` et DetailPanel en largeur fixe (`420px`). Le bottom panel d'exploration n'affecte pas la largeur des colonnes. Les colonnes SHALL être scrollables horizontalement si leur largeur minimale combinée dépasse l'espace disponible.

#### Scenario: Redimensionnement de la fenêtre sans panel
- **WHEN** l'utilisateur redimensionne la fenêtre du navigateur et aucun panel n'est ouvert
- **THEN** les colonnes s'adaptent automatiquement pour remplir toute la largeur disponible sans débordement horizontal

#### Scenario: DetailPanel ouvert — colonnes réduites
- **WHEN** le DetailPanel est ouvert
- **THEN** les colonnes occupent l'espace restant après le slot de 420px du DetailPanel, avec un scroll horizontal si nécessaire

#### Scenario: DetailPanel fermé — colonnes pleine largeur
- **WHEN** le DetailPanel est fermé
- **THEN** les colonnes reprennent toute la largeur disponible

#### Scenario: Bottom panel ouvert — largeur colonnes inchangée
- **WHEN** le bottom panel d'exploration est ouvert
- **THEN** les colonnes conservent leur largeur (le bottom panel n'affecte que la hauteur disponible)

### Requirement: Rafraîchissement automatique du Kanban
Le Kanban SHALL se rafraîchir automatiquement pour refléter les changements apportés aux fichiers OpenSpec par des outils externes (Claude Code, openspec CLI). Le rafraîchissement SHALL se faire via les événements SSE du stream `/api/workspaces/{id}/events` — sans polling périodique. À réception d'un événement `change_updated`, le frontend SHALL invalider la liste des changes ET le détail du change concerné. À réception d'un événement `change_created` ou `change_deleted`, le frontend SHALL invalider uniquement la liste des changes. En cas d'indisponibilité du stream SSE, les données affichées restent celles du dernier fetch réussi (pas de fallback polling).

#### Scenario: Mise à jour d'un change via Claude Code
- **WHEN** Claude Code modifie `tasks.md` d'un change et que l'événement `change_updated` est reçu via SSE
- **THEN** le frontend recharge la liste des changes et le détail du change concerné, et le kanban se met à jour immédiatement sans action utilisateur

#### Scenario: Création d'un nouveau change via la CLI
- **WHEN** la CLI crée un nouveau répertoire de change et que l'événement `change_created` est reçu via SSE
- **THEN** le frontend recharge la liste des changes et la nouvelle carte apparaît dans la colonne appropriée

#### Scenario: Archivage d'un change via la CLI
- **WHEN** la CLI archive un change (déplace le répertoire) et que l'événement `change_deleted` est reçu via SSE
- **THEN** le frontend recharge la liste des changes et la carte disparaît des colonnes actives

#### Scenario: Déconnexion SSE — données conservées
- **WHEN** le stream SSE est interrompu (réseau, redémarrage serveur)
- **THEN** les données du kanban restent affichées dans leur dernier état connu, sans message d'erreur intrusif

### Requirement: Distinction entre worker actif et worker en pause sur un change
Chaque change renvoyé par `GET /changes` SHALL exposer deux indicateurs exclusifs : `worker_active`, vrai uniquement lorsqu'un worker du pool exécute le change, et `worker_paused`, vrai uniquement lorsque le worker qui tient le change est en pause. Un change tenu par un worker en pause SHALL avoir `worker_active = false` et `worker_paused = true`. La carte d'un change avec `worker_paused = true` SHALL afficher un badge de pause distinct du badge de worker actif, et SHALL NOT afficher le bouton d'arrêt de worker, réservé à `worker_active = true`.

#### Scenario: Change tenu par un worker en pause
- **WHEN** le worker du changement `add-user-auth` est en pause
- **THEN** la liste des changes expose `worker_active = false` et `worker_paused = true` pour ce change, et sa carte affiche le badge de pause sans bouton d'arrêt

#### Scenario: Change tenu par un worker qui s'exécute
- **WHEN** le worker du changement `add-user-auth` exécute un tour d'agent
- **THEN** la liste des changes expose `worker_active = true` et `worker_paused = false`, et la carte affiche le badge de worker actif et le bouton d'arrêt

#### Scenario: Pause levée
- **WHEN** la pause du worker du changement `add-user-auth` est levée par une reprise ou par la rétrogradation du change
- **THEN** la liste des changes n'expose plus `worker_paused = true` pour ce change et sa carte n'affiche plus le badge de pause

### Requirement: Actions de reprise sur la carte d'un change en pause
Chaque change renvoyé par `GET /changes` et tenu par un worker du pool SHALL exposer l'identifiant de ce worker (`worker_id`) ; le champ SHALL être absent lorsque aucun worker ne tient le change. La carte d'un change avec `worker_paused = true` SHALL afficher, en plus du badge de pause, un bouton « Reprendre » et un bouton « Reprendre en finalisant » qui demandent respectivement la reprise ordinaire et la reprise avec `finalize_only` du worker `worker_id`. « Reprendre en finalisant » SHALL être désactivé, avec une info-bulle indiquant le nombre de tâches restantes, tant que `tasks_done` est inférieur à `tasks_total` (les compteurs de la carte reflétant déjà le worktree du worker). Ces boutons SHALL NOT ouvrir le DetailPanel ni amorcer un drag de la carte, et SHALL NOT apparaître sur une carte sans worker en pause. En cas d'échec, l'application SHALL afficher le message d'erreur du backend dans une notification et conserver la carte inchangée.

#### Scenario: Carte d'un change en pause
- **WHEN** le worker 2 du changement `add-user-auth` est en pause avec 9 tâches cochées sur 10
- **THEN** la liste des changes expose `worker_id = 2` pour ce change, et sa carte affiche le badge de pause, « Reprendre » actif et « Reprendre en finalisant » désactivé avec l'info-bulle « Il reste 1 tâche »

#### Scenario: Reprise en finalisant depuis la carte
- **WHEN** toutes les tâches du change en pause sont cochées et que l'utilisateur clique sur « Reprendre en finalisant » sur sa carte
- **THEN** l'application demande la reprise du worker avec `finalize_only`, le DetailPanel ne s'ouvre pas et la carte quitte l'état de pause dès la reprise par le pool

#### Scenario: Reprise ordinaire depuis la carte
- **WHEN** l'utilisateur clique sur « Reprendre » sur la carte d'un change en pause
- **THEN** l'application demande la reprise du worker sans `finalize_only`

#### Scenario: Aucune action sans pause
- **WHEN** un change est tenu par un worker actif ou n'est tenu par aucun worker
- **THEN** sa carte n'affiche aucun bouton de reprise

#### Scenario: Refus du backend
- **WHEN** le backend répond `409` à la demande de reprise depuis la carte
- **THEN** une notification affiche le message retourné et la carte reste en pause
