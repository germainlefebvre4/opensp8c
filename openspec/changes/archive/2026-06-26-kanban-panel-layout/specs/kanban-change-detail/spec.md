## MODIFIED Requirements

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

### Requirement: Ouvrir l'ExplorePanel au clic sur une carte To Explore
L'utilisateur SHALL pouvoir cliquer sur une carte dans la colonne **To Explore** pour ouvrir l'ExplorePanel (session de conversation). Le panel SHALL s'afficher dans un slot dédié à droite des colonnes Kanban (layout inline), sans masquer les colonnes. La carte entière est la zone cliquable.

#### Scenario: Clic sur carte en colonne To Explore
- **WHEN** l'utilisateur clique sur une carte dans la colonne **To Explore**
- **THEN** l'ExplorePanel s'ouvre dans un slot inline à droite des colonnes, permettant d'entamer ou reprendre une session de conversation

#### Scenario: ExplorePanel inline ne masque pas les colonnes
- **WHEN** l'ExplorePanel est ouvert
- **THEN** les colonnes Kanban restent visibles et interactibles à gauche du panel

## ADDED Requirements

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
