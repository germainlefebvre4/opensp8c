# Spec Delta

## MODIFIED Requirements

### Requirement: Ouvrir le DetailPanel au clic sur une carte hors To Explore
L'utilisateur SHALL pouvoir cliquer sur une carte dans les colonnes **To Do**, **In Progress** ou **Done** pour ouvrir un panneau latéral (`DetailPanel`) affichant le détail complet du change. Par défaut le panel SHALL s'afficher dans un slot dédié à droite des colonnes Kanban (layout inline), sans masquer les colonnes ; sa largeur SHALL être fluide, comprise entre **320px** et **420px** selon l'espace disponible. Lorsque l'espace ne permet plus de conserver les colonnes visibles même après rétrécissement du panel et repli du slot Done/Archived (voir `kanban-board`), le panel SHALL s'afficher en **overlay** par-dessus la partie droite des colonnes. Un seul panneau peut être ouvert à la fois ; ouvrir un panneau ferme tout autre panneau précédemment ouvert (ExplorePanel inclus).

#### Scenario: Clic sur carte en colonne To Do
- **WHEN** l'utilisateur clique sur une carte dans la colonne **To Do**, **In Progress** ou **Done**
- **THEN** le `DetailPanel` s'ouvre à droite des colonnes, affichant le détail du change correspondant

#### Scenario: Panel inline ne masque pas les colonnes
- **WHEN** le DetailPanel est ouvert et l'espace suffit à conserver les colonnes visibles
- **THEN** les colonnes Kanban restent visibles et interactibles à gauche du panel

#### Scenario: Largeur fluide du panel
- **WHEN** l'espace disponible diminue alors que le DetailPanel est ouvert en mode inline
- **THEN** la largeur du panel se réduit progressivement jusqu'à 320px avant que d'autres paliers de l'échelle de dégradation soient déclenchés

#### Scenario: Panel en overlay quand l'espace manque
- **WHEN** le DetailPanel est ouvert et que les colonnes ne tiennent plus à côté du panel à sa largeur minimale, slot Done/Archived déjà replié
- **THEN** le panel s'affiche en overlay par-dessus les colonnes de droite, qui gardent leur largeur et ne sont plus poussées

#### Scenario: Exclusivité du panneau
- **WHEN** un panneau (DetailPanel ou ExplorePanel) est déjà ouvert et l'utilisateur clique sur une autre carte
- **THEN** le panneau précédent se ferme et le nouveau s'ouvre

#### Scenario: Fermeture du panneau
- **WHEN** l'utilisateur clique sur le bouton de fermeture du DetailPanel
- **THEN** le panneau se ferme et les colonnes reprennent toute la largeur
