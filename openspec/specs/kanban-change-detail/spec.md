## Purpose

Permettre à l'utilisateur de consulter et piloter le détail complet d'un change (tâches, artifacts, tags, actions) depuis un panneau latéral inline (`DetailPanel`) ouvert au clic sur une carte Kanban.

## Requirements

### Requirement: Ouvrir le DetailPanel au clic sur une carte hors To Explore
L'utilisateur SHALL pouvoir cliquer sur une carte dans les colonnes **To Do**, **In Progress** ou **Done** pour ouvrir un panneau latéral (`DetailPanel`) affichant le détail complet du change. Le panel SHALL s'afficher dans un slot dédié à droite des colonnes Kanban (layout inline), sans masquer les colonnes. Un seul panneau peut être ouvert à la fois ; ouvrir un panneau ferme tout autre panneau précédemment ouvert (ExplorePanel inclus).

#### Scenario: Clic sur carte en colonne To Do
- **WHEN** l'utilisateur clique sur une carte dans la colonne **To Do**, **In Progress** ou **Done**
- **THEN** le `DetailPanel` s'ouvre dans un slot inline à droite des colonnes, affichant le détail du change correspondant

#### Scenario: Panel inline ne masque pas les colonnes
- **WHEN** le DetailPanel est ouvert
- **THEN** les colonnes Kanban restent visibles et interactibles à gauche du panel

#### Scenario: Exclusivité du panneau
- **WHEN** un panneau (DetailPanel ou ExplorePanel) est déjà ouvert et l'utilisateur clique sur une autre carte
- **THEN** le panneau précédent se ferme et le nouveau s'ouvre

#### Scenario: Fermeture du panneau
- **WHEN** l'utilisateur clique sur le bouton de fermeture du DetailPanel
- **THEN** le panneau se ferme et les colonnes reprennent toute la largeur

### Requirement: Afficher la liste des tâches dans le DetailPanel
Le `DetailPanel` SHALL afficher la liste complète des tâches du change avec leur état (faite / non faite), lue depuis le fichier `tasks.md` via le backend. Chaque tâche SHALL être rendue comme un checkbox interactif permettant de toggler son état directement depuis le panel.

#### Scenario: Change avec tasks.md
- **WHEN** le DetailPanel s'ouvre pour un change ayant un `tasks.md`
- **THEN** chaque tâche est affichée avec un checkbox dont l'état reflète `[ ]` ou `[x]`

#### Scenario: Change sans tasks.md
- **WHEN** le DetailPanel s'ouvre pour un change sans `tasks.md`
- **THEN** le panneau affiche un message indiquant qu'aucune tâche n'a été définie, sans erreur

#### Scenario: Clic sur un checkbox non coché
- **WHEN** l'utilisateur clique sur le checkbox d'une tâche non complétée
- **THEN** le checkbox est désactivé pendant la requête, puis passe à l'état coché une fois le serveur ayant confirmé la mise à jour

#### Scenario: Clic sur un checkbox coché
- **WHEN** l'utilisateur clique sur le checkbox d'une tâche complétée
- **THEN** le checkbox est désactivé pendant la requête, puis repasse à l'état non coché une fois le serveur ayant confirmé la mise à jour

#### Scenario: Erreur lors du toggle
- **WHEN** la requête PATCH échoue
- **THEN** le checkbox retrouve son état initial et une notification d'erreur est affichée

### Requirement: Afficher les artifacts dans le DetailPanel
Le `DetailPanel` SHALL afficher le contenu des artifacts `proposal.md` et `design.md` du change lorsqu'ils existent.

#### Scenario: Artifact présent
- **WHEN** le change possède un `proposal.md` ou `design.md`
- **THEN** le contenu de l'artifact est affiché dans le DetailPanel sous une section dédiée

#### Scenario: Artifact absent
- **WHEN** le change ne possède pas encore un des artifacts attendus
- **THEN** la section correspondante est absente ou indique que le fichier n'existe pas encore

### Requirement: Changer le statut depuis le DetailPanel
L'utilisateur SHALL pouvoir modifier le `kanban_status` d'un change directement depuis le DetailPanel via des boutons d'action.

#### Scenario: Changement de statut
- **WHEN** l'utilisateur clique sur un bouton de transition de statut dans le DetailPanel (ex. "→ In Progress")
- **THEN** le `kanban_status` est mis à jour dans `.openspec.yaml`, la carte se déplace dans la colonne correspondante, et le DetailPanel reste ouvert

### Requirement: Afficher l'onglet Actions dans le DetailPanel
Le `DetailPanel` SHALL afficher un onglet **Actions**, à côté des onglets Tâches, Proposal, Design, Log et Tags. Cet onglet SHALL regrouper les actions de cycle de vie disponibles pour le change ouvert. Son contenu SHALL varier selon le `kanban_status` du change : le bouton Supprimer est toujours présent (sauf statut `archived`), le bouton Archiver n'apparaît qu'au statut `done`, et les boutons "Approuver & Fusionner" / "Demander correction" n'apparaissent qu'au statut `to-review`. Pour tout change non archivé, l'onglet SHALL en outre afficher une section **Vérification** décrite par l'exigence « Réglage de la vérification dans le DetailPanel », qui n'est pas une action de cycle de vie.

#### Scenario: Onglet Actions toujours présent
- **WHEN** le DetailPanel s'ouvre pour un change, quel que soit son statut
- **THEN** l'onglet Actions est visible dans la barre d'onglets

#### Scenario: Contenu de l'onglet Actions au statut Done
- **WHEN** l'onglet Actions est actif pour un change en statut **Done**
- **THEN** les boutons Supprimer et Archiver sont tous deux affichés

#### Scenario: Contenu de l'onglet Actions au statut To Review
- **WHEN** l'onglet Actions est actif pour un change en statut **To Review**
- **THEN** les boutons Supprimer, "Approuver & Fusionner" et "Demander correction" sont affichés

#### Scenario: Contenu de l'onglet Actions aux autres statuts
- **WHEN** l'onglet Actions est actif pour un change en statut **To Do** ou **In Progress**
- **THEN** seul le bouton Supprimer est affiché parmi les boutons d'action de cycle de vie
- **THEN** la section Vérification est affichée en plus

#### Scenario: Change archivé
- **WHEN** l'onglet Actions est actif pour un change archivé
- **THEN** aucune action de cycle de vie ni section de réglage n'est modifiable

### Requirement: Archiver un change depuis le DetailPanel
L'utilisateur SHALL pouvoir déclencher l'archivage d'un change en statut **Done** depuis le bouton Archiver de l'onglet Actions du DetailPanel.

#### Scenario: Archivage depuis l'onglet Actions
- **WHEN** l'utilisateur clique sur "Archiver" dans l'onglet Actions du DetailPanel d'un change en statut **Done**
- **THEN** le change est archivé et le DetailPanel se ferme

#### Scenario: Erreur d'archivage
- **WHEN** l'archivage échoue
- **THEN** le message d'erreur est affiché dans l'onglet Actions

### Requirement: Supprimer un change depuis le DetailPanel
L'utilisateur SHALL pouvoir supprimer définitivement un change depuis le bouton Supprimer de l'onglet Actions, pour tout change dont le `kanban_status` est `todo`, `in-progress`, `to-review` ou `done`. La suppression n'est pas proposée pour un change en statut `archived`. La suppression SHALL être protégée par une confirmation explicite de l'utilisateur avant d'être déclenchée. Si un worker de l'Agent Pool Orchestrator est actif sur ce change, le bouton Supprimer SHALL être désactivé. Si le change possède un ghost record associé encore actif (exploration non "figée"), ce ghost record SHALL être supprimé avec le change.

#### Scenario: Clic sur Supprimer ouvre une confirmation
- **WHEN** l'utilisateur clique sur le bouton Supprimer dans l'onglet Actions
- **THEN** une boîte de dialogue de confirmation s'affiche, nommant le change à supprimer

#### Scenario: Confirmation déclenche la suppression
- **WHEN** l'utilisateur confirme la suppression
- **THEN** le change est supprimé, le DetailPanel se ferme, et la carte disparaît du Kanban

#### Scenario: Annulation de la confirmation
- **WHEN** l'utilisateur annule la boîte de dialogue de confirmation
- **THEN** rien n'est supprimé et le DetailPanel reste ouvert sur le change

#### Scenario: Bouton désactivé si un worker est actif
- **WHEN** un worker de l'Agent Pool Orchestrator est actif sur le change ouvert
- **THEN** le bouton Supprimer est désactivé

#### Scenario: Suppression cascade le ghost associé
- **WHEN** l'utilisateur confirme la suppression d'un change ayant un ghost record associé encore actif
- **THEN** le change et son ghost record associé sont tous deux supprimés, et la carte ghost disparaît également de la colonne To Explore

#### Scenario: Erreur de suppression
- **WHEN** la requête de suppression échoue
- **THEN** un message d'erreur est affiché dans l'onglet Actions et le change n'est pas supprimé

### Requirement: Afficher les artifacts en Markdown rendu dans le DetailPanel
Les onglets **Proposal** et **Design** du DetailPanel SHALL offrir un toggle permettant de basculer entre l'affichage en texte brut (raw) et l'affichage en Markdown rendu. Le mode est partagé entre les deux onglets. Le mode par défaut est le rendu Markdown.

#### Scenario: Toggle vers raw text
- **WHEN** l'utilisateur active le mode "Raw" via le toggle
- **THEN** le contenu de l'onglet actif (Proposal ou Design) s'affiche en texte préformaté (`<pre>`), et l'onglet non actif affichera également le mode raw au prochain clic

#### Scenario: Toggle vers Markdown rendu
- **WHEN** l'utilisateur active le mode "Rendu" via le toggle
- **THEN** le contenu de l'onglet actif est affiché en Markdown interprété avec styles typographiques (titres, listes, code)

#### Scenario: Persistance du mode au changement d'onglet
- **WHEN** l'utilisateur change d'onglet (de Proposal à Design ou inversement)
- **THEN** le mode Raw/Rendu actif est conservé

#### Scenario: Toggle absent sur l'onglet Tâches
- **WHEN** l'onglet "Tâches" est actif
- **THEN** le toggle Raw/Rendu n'est pas affiché

### Requirement: Afficher les tags dans le DetailPanel
Le `DetailPanel` SHALL afficher une section **Tags** lorsqu'un change possède une section `tags` dans son `.openspec.yaml`. La section affiche un badge par valeur du tableau `tags.type` (ou aucun badge de type si la liste est vide), le niveau de complexité, et la liste des composants touchés. Si le change n'a pas encore de tags, la section est absente sans erreur.

#### Scenario: Change avec tags complets
- **WHEN** le DetailPanel s'ouvre pour un change possédant `tags.type: [frontend]`, `tags.complexity` et `tags.components`
- **THEN** la section Tags est affichée avec un badge type "frontend", l'indicateur de complexité (points ou étoiles sur 5), et les chips de composants

#### Scenario: Change avec plusieurs valeurs de type
- **WHEN** le DetailPanel s'ouvre pour un change possédant `tags.type: [frontend, backend]`
- **THEN** la section Tags affiche deux badges type distincts, un pour "frontend" et un pour "backend"

#### Scenario: Change sans tags
- **WHEN** le DetailPanel s'ouvre pour un change sans section `tags`
- **THEN** la section Tags est absente du panneau, sans message d'erreur

#### Scenario: Bouton de retag dans le DetailPanel
- **WHEN** l'utilisateur clique sur l'icône de rafraîchissement des tags dans le DetailPanel
- **THEN** une requête `POST /api/workspaces/{id}/changes/{name}/retag` est déclenchée et les tags se mettent à jour une fois la dérivation terminée

### Requirement: Endpoint de détail d'un change
Le backend SHALL exposer un endpoint `GET /api/workspaces/{id}/changes/{name}` retournant le détail complet d'un change : métadonnées, liste des tâches avec texte et état, contenu des artifacts `proposal.md` et `design.md`, et tags sémantiques si présents.

#### Scenario: Change existant
- **WHEN** une requête `GET /api/workspaces/{id}/changes/{name}` est effectuée pour un change existant
- **THEN** la réponse contient `name`, `kanban_status`, `tasks_done`, `tasks_total`, `tasks` (tableau d'objets `{ text, done }`), `artifacts` (`{ proposal, design }` avec chaîne vide si absent), et `tags` (objet optionnel `{ type: string[], complexity, components[] }` ou `null` si absent)

#### Scenario: Change inexistant
- **WHEN** la requête cible un change qui n'existe pas
- **THEN** le backend retourne HTTP 404

### Requirement: Endpoint de suppression d'un change
Le backend SHALL exposer un endpoint `DELETE /api/workspaces/{id}/changes/{name}` qui supprime définitivement le dossier `openspec/changes/{name}/`. Si un worker de l'Agent Pool Orchestrator est actif sur ce change, l'endpoint SHALL refuser la suppression. Si un ghost record du workspace porte le même nom que le change, l'endpoint SHALL également le supprimer (arrêt de session, retrait de `preferences.json`, suppression du brouillon et des logs associés).

#### Scenario: Suppression réussie
- **WHEN** une requête `DELETE /api/workspaces/{id}/changes/{name}` cible un change existant sans worker actif
- **THEN** le backend supprime le dossier du change et retourne HTTP 204

#### Scenario: Change inexistant
- **WHEN** la requête cible un change qui n'existe pas
- **THEN** le backend retourne HTTP 404

#### Scenario: Worker actif sur le change
- **WHEN** la requête cible un change sur lequel un worker de l'Agent Pool Orchestrator est actif
- **THEN** le backend retourne HTTP 409 et ne supprime rien

#### Scenario: Ghost record associé
- **WHEN** la requête cible un change pour lequel un ghost record du même workspace porte le même nom
- **THEN** le backend supprime aussi ce ghost record avant de retourner HTTP 204

### Requirement: Liste des tâches reflétant le worktree du worker actif
La liste des tâches renvoyée par `GET /api/workspaces/{id}/changes/{name}` et affichée dans le `DetailPanel` SHALL provenir du `tasks.md` de la branche du change dès qu'une source de travail existe. Tant qu'un worker du pool est actif sur un change (y compris en pause), elle SHALL être lue depuis le `tasks.md` du worktree du worker, de sorte que `tasks`, `tasks_done`, `tasks_total` et `kanban_status` soient cohérents avec la carte du Kanban. Lorsqu'aucun worker ne tient le change mais que `feature/<change>` existe, y compris pour un change en revue, `tasks`, `tasks_done` et `tasks_total` SHALL être lus depuis le `tasks.md` du worktree du change s'il existe, sinon depuis la branche, sans nécessiter de worktree ; `kanban_status` SHALL rester dérivé du marqueur de revue, de l'état « lancé » et du `tasks.md` du dépôt principal, et SHALL NOT être recalculé à partir de la branche. Lorsque le `tasks.md` de la branche ou du worktree est absent ou ne contient aucune tâche, la liste SHALL provenir du dépôt principal. La lecture SHALL être en lecture seule : elle ne crée ni worktree ni commit.

#### Scenario: Détail d'un change pendant le run
- **WHEN** l'utilisateur ouvre le DetailPanel d'un change dont le worker a coché 3 tâches dans son worktree
- **THEN** ces 3 tâches sont affichées comme faites et `tasks_done` vaut 3

#### Scenario: Détail d'un change en revue
- **WHEN** l'utilisateur ouvre le DetailPanel d'un change en revue dont la branche a 8 tâches cochées sur 10 et dont le `tasks.md` du dépôt principal est à `0/10`
- **THEN** la liste affiche les tâches de la branche, `tasks_done` vaut 8, `tasks_total` vaut 10 et `kanban_status` reste `to-review`

#### Scenario: Détail d'un change en revue sans worktree
- **WHEN** le worktree d'un change en revue a été supprimé et que la branche existe
- **THEN** la liste des tâches est lue depuis la branche, aucun worktree n'est créé et aucun commit n'est ajouté

#### Scenario: Détail d'un change à branche seule
- **WHEN** un change rétrogradé en Ready (worktree et commits conservés) porte une branche à 7 tâches cochées sur 10
- **THEN** le DetailPanel affiche les 7 tâches cochées et `tasks_done` vaut 7, tandis que `kanban_status` reste `ready`

#### Scenario: Branche sans liste de tâches
- **WHEN** la branche du change existe mais que son `tasks.md` est absent ou sans tâche
- **THEN** la liste des tâches provient du dépôt principal

#### Scenario: Détail d'un change dont le worker est terminé
- **WHEN** le worker du change a été libéré
- **THEN** la liste des tâches du DetailPanel provient du `tasks.md` de la branche du change si elle existe (en revue, c'est la branche qui est lue), et du `tasks.md` du dépôt principal sinon

### Requirement: Actions de reprise dans le DetailPanel d'un change en pause
`GET /api/workspaces/{id}/changes/{name}` SHALL exposer, lorsqu'un worker du pool tient le change, l'identifiant du worker (`worker_id`) et, s'il est en pause, sa raison de blocage lisible (`worker_blocked_reason`). Le `DetailPanel` d'un change avec `worker_paused = true` SHALL afficher un bandeau contenant cette raison de blocage et deux boutons, « Reprendre » et « Reprendre en finalisant », ayant le même effet que sur la carte et dans le panneau d'état du pool. « Reprendre en finalisant » SHALL être désactivé tant que `tasks_done` est inférieur à `tasks_total`, avec le nombre de tâches restantes indiqué, et SHALL devenir actif dès que la dernière tâche est cochée depuis ce panneau, sans rechargement de la page. Un échec de reprise SHALL afficher le message du backend dans le bandeau. Le bandeau SHALL NOT apparaître pour un change sans worker en pause.

#### Scenario: Bandeau d'un change en pause
- **WHEN** l'utilisateur ouvre le DetailPanel du changement `add-user-auth`, dont le worker est en pause avec la raison « Validation réussie mais tâches restantes incomplètes (9/10) dans tasks.md »
- **THEN** le panneau affiche cette raison, un bouton « Reprendre » actif et un bouton « Reprendre en finalisant » désactivé indiquant 1 tâche restante

#### Scenario: Cocher la dernière tâche débloque la finalisation
- **WHEN** l'utilisateur coche la dernière tâche depuis le DetailPanel d'un change dont le worker est en pause
- **THEN** la tâche est enregistrée dans le worktree du worker (voir `task-toggle`) et « Reprendre en finalisant » devient actif

#### Scenario: Reprise en finalisant depuis le détail
- **WHEN** l'utilisateur clique sur « Reprendre en finalisant » dans le bandeau
- **THEN** l'application demande la reprise avec `finalize_only`, et le bandeau disparaît dès que le worker n'est plus en pause

#### Scenario: Pas de bandeau sans pause
- **WHEN** l'utilisateur ouvre le DetailPanel d'un change tenu par un worker actif ou par aucun worker
- **THEN** aucun bandeau ni bouton de reprise n'est affiché

### Requirement: Signalement des tâches de validation humaine dans le DetailPanel
Dans la liste des tâches du `DetailPanel`, une tâche dont `human_review` est vrai SHALL porter un badge « Validation humaine » (« Human validation » en anglais), visible qu'elle soit cochée ou non, et son texte SHALL être affiché sans le commentaire du marqueur. Le checkbox d'une tâche marquée SHALL rester interactif dans les conditions où celui des autres tâches l'est. Pour un change en To Review, l'onglet Tâches SHALL aussi afficher, au-dessus de la liste, le nombre de tâches restant à valider lorsqu'il est supérieur à zéro. Les libellés SHALL être disponibles en français et en anglais.

#### Scenario: Tâche marquée non cochée
- **WHEN** le DetailPanel affiche une tâche `4.2 Parcours manuel` avec `human_review` à vrai et `done` à faux
- **THEN** la tâche porte le badge « Validation humaine », son texte ne montre pas le commentaire du marqueur et son checkbox est décoché et actif

#### Scenario: Tâche marquée cochée
- **WHEN** l'utilisateur coche cette tâche
- **THEN** le badge reste affiché et la tâche apparaît comme faite

#### Scenario: Compteur de tâches à valider en revue
- **WHEN** l'utilisateur ouvre l'onglet Tâches d'un change en To Review qui a 2 tâches non cochées
- **THEN** un message indique « 2 tâches à valider » au-dessus de la liste

#### Scenario: Aucune tâche restante
- **WHEN** toutes les tâches d'un change en To Review sont cochées
- **THEN** aucun message de tâches à valider n'est affiché
### Requirement: Réglage de la vérification dans le DetailPanel
La section Vérification de l'onglet Actions SHALL permettre de définir, pour le change ouvert, chacune des étapes `conformity` et `ui` avec les choix « Hérité », « Activé » et « Désactivé ». Lorsque « Hérité » est sélectionné, la section SHALL indiquer la valeur héritée (Configuration et workspace). Un choix SHALL être enregistré immédiatement via `PATCH /api/workspaces/{id}/changes/{name}/verification`, et une erreur SHALL être affichée sans modifier la sélection enregistrée. La section SHALL être absente pour un change archivé.

#### Scenario: Valeur héritée affichée
- **WHEN** le workspace active `ui` et que le change n'a pas de réglage
- **THEN** le choix « Hérité » est sélectionné pour `ui` et indique la valeur héritée « Activé »

#### Scenario: Désactivation pour ce change
- **WHEN** l'utilisateur choisit « Désactivé » pour `ui`
- **THEN** une requête de réglage est envoyée avec `ui` à `false` et la section affiche le choix enregistré

#### Scenario: Retour à l'héritage
- **WHEN** l'utilisateur choisit « Hérité » pour une étape qu'il avait désactivée
- **THEN** une requête de réglage est envoyée avec `null` pour cette étape et la valeur héritée est affichée

#### Scenario: Erreur d'enregistrement
- **WHEN** l'enregistrement échoue
- **THEN** un message d'erreur est affiché et le choix précédent est conservé

### Requirement: Le détail d'un change expose ses réglages de vérification
La réponse de `GET /api/workspaces/{id}/changes/{name}` SHALL inclure un objet `verification` contenant `override` (valeurs définies par le change, champs absents si héritées), `inherited` (valeurs effectives sans le réglage du change) et `resolved` (valeurs effectives pour le change), chacune avec les booléens `conformity` et `ui` lorsque applicable.

#### Scenario: Change sans réglage
- **WHEN** un change n'a pas de réglage de vérification et que la Configuration n'en définit pas
- **THEN** `override` est vide et `inherited` et `resolved` valent `false` pour les deux étapes

#### Scenario: Change avec réglage
- **WHEN** un change définit `ui` à `true` alors que le workspace ne l'active pas
- **THEN** `override.ui` vaut `true`, `inherited.ui` vaut `false` et `resolved.ui` vaut `true`
