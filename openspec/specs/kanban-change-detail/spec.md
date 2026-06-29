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

### Requirement: Archiver un change depuis le DetailPanel
L'utilisateur SHALL pouvoir déclencher l'archivage d'un change en statut **Done** depuis le DetailPanel.

#### Scenario: Archivage depuis le panneau
- **WHEN** l'utilisateur clique sur "Archiver" dans le DetailPanel d'un change en statut **Done**
- **THEN** le change est archivé et le DetailPanel se ferme

#### Scenario: Erreur d'archivage
- **WHEN** l'archivage échoue
- **THEN** le message d'erreur est affiché dans le DetailPanel

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

### Requirement: Endpoint de détail d'un change
Le backend SHALL exposer un endpoint `GET /api/workspaces/{id}/changes/{name}` retournant le détail complet d'un change : métadonnées, liste des tâches avec texte et état, et contenu des artifacts `proposal.md` et `design.md`.

#### Scenario: Change existant
- **WHEN** une requête `GET /api/workspaces/{id}/changes/{name}` est effectuée pour un change existant
- **THEN** la réponse contient `name`, `kanban_status`, `tasks_done`, `tasks_total`, `tasks` (tableau d'objets `{ text, done }`), et `artifacts` (`{ proposal, design }` avec chaîne vide si absent)

#### Scenario: Change inexistant
- **WHEN** la requête cible un change qui n'existe pas
- **THEN** le backend retourne HTTP 404
