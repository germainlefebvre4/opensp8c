## ADDED Requirements

### Requirement: Mode Matrice dans la Timeline
La TimelinePage SHALL proposer un mode Matrice accessible via un toggle [Changes | Matrice] en haut de page. Ce mode affiche une grille spec × date représentant l'intensité d'activité de chaque spec dans le temps, ainsi qu'un panel droit de détail spec ouvert au clic.

#### Scenario: Activation du mode Matrice
- **WHEN** l'utilisateur clique sur "Matrice" dans le toggle de la TimelinePage
- **THEN** la grille spec × date s'affiche, chaque ligne correspondant à une spec, chaque colonne à un jour de la période couverte par les changes

#### Scenario: Mode Changes actif par défaut
- **WHEN** l'utilisateur navigue vers `/timeline`
- **THEN** le mode Changes est actif par défaut

#### Scenario: Deep-link vers Matrice avec spec pré-sélectionnée
- **WHEN** l'URL contient le paramètre `?spec=<name>`
- **THEN** le mode Matrice s'active et le panel droit s'ouvre directement sur la spec nommée

### Requirement: Grille spec × date avec intensité colorée
La grille du mode Matrice SHALL afficher une ligne par spec (de `openspec/specs/`) et une colonne par jour couvert par au moins un change. Chaque cellule indique par une intensité de couleur le nombre de changes ayant touché cette spec ce jour-là.

#### Scenario: Cellule vide
- **WHEN** aucun change n'a touché une spec un jour donné
- **THEN** la cellule est vide (fond neutre)

#### Scenario: Cellule avec un change
- **WHEN** un change a touché cette spec ce jour
- **THEN** la cellule affiche une couleur d'intensité basse

#### Scenario: Cellule avec plusieurs changes
- **WHEN** plusieurs changes ont touché cette spec le même jour
- **THEN** la cellule affiche une intensité proportionnelle au nombre de changes (max à 3+)

#### Scenario: Specs sans aucun change
- **WHEN** une spec n'a aucun change lié dans tout l'historique
- **THEN** la ligne de la spec s'affiche avec un style atténué ou un indicateur ⚠

### Requirement: Sélection d'une spec dans la grille
L'utilisateur SHALL pouvoir cliquer sur le nom d'une spec dans la grille (colonne de gauche) pour ouvrir le panel droit affichant la timeline des changes de cette spec.

#### Scenario: Clic sur un nom de spec
- **WHEN** l'utilisateur clique sur le nom d'une spec dans la colonne de gauche de la grille
- **THEN** le panel droit s'ouvre avec la liste des changes ayant touché cette spec, triée du plus récent au plus ancien

#### Scenario: Spec sélectionnée mise en évidence
- **WHEN** une spec est sélectionnée
- **THEN** la ligne correspondante dans la grille est mise en évidence visuellement

#### Scenario: Fermeture du panel de spec
- **WHEN** l'utilisateur ferme le panel droit
- **THEN** la grille reprend toute la largeur disponible et aucune spec n'est sélectionnée

### Requirement: Drill-down vers le DetailPanel depuis le panel de spec
L'utilisateur SHALL pouvoir cliquer sur un change dans le panel de spec pour ouvrir le DetailPanel de ce change, remplaçant le panel de spec dans le slot droit.

#### Scenario: Clic sur un change dans le panel de spec
- **WHEN** l'utilisateur clique sur un change dans la liste du panel de spec
- **THEN** le panel droit bascule vers le DetailPanel de ce change (proposal, design, tasks)

#### Scenario: Retour au panel de spec
- **WHEN** l'utilisateur ferme le DetailPanel depuis le mode Matrice
- **THEN** le panel droit revient à la liste des changes de la spec précédemment sélectionnée

### Requirement: Navigation depuis le panel de spec vers la SpecsPage
Le panel de spec du mode Matrice SHALL proposer un lien "Voir la spec →" permettant de naviguer vers le contenu de la spec dans la SpecsPage.

#### Scenario: Clic sur le lien "Voir la spec →"
- **WHEN** l'utilisateur clique sur "Voir la spec →" dans le panel de spec
- **THEN** l'application navigue vers `/specs?workspace=<id>&selected=<name>` affichant le contenu de la spec
