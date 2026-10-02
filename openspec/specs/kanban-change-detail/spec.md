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
Le `DetailPanel` SHALL afficher un onglet **Actions**, à côté des onglets Tâches, Proposal, Design, Log et Tags. Cet onglet SHALL regrouper les actions de cycle de vie disponibles pour le change ouvert. Son contenu SHALL varier selon le `kanban_status` du change : le bouton Supprimer est toujours présent (sauf statut `archived`), le bouton Archiver n'apparaît qu'au statut `done`, et les boutons "Approuver & Fusionner" / "Demander correction" n'apparaissent qu'au statut `to-review`.

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
- **THEN** seul le bouton Supprimer est affiché

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
Tant qu'un worker du pool est actif sur un change (y compris en pause), la liste des tâches renvoyée par `GET /api/workspaces/{id}/changes/{name}` et affichée dans le `DetailPanel` SHALL être lue depuis le `tasks.md` du worktree du worker, de sorte que `tasks`, `tasks_done`, `tasks_total` et `kanban_status` soient cohérents avec la carte du Kanban. Lorsque le `tasks.md` du worktree est absent ou ne contient aucune tâche, la liste SHALL provenir du dépôt principal.

#### Scenario: Détail d'un change pendant le run
- **WHEN** l'utilisateur ouvre le DetailPanel d'un change dont le worker a coché 3 tâches dans son worktree
- **THEN** ces 3 tâches sont affichées comme faites et `tasks_done` vaut 3

#### Scenario: Détail d'un change dont le worker est terminé
- **WHEN** le worker du change a été libéré
- **THEN** la liste des tâches du DetailPanel provient à nouveau du `tasks.md` du dépôt principal
