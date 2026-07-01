## MODIFIED Requirements

### Requirement: Mode Matrice dans la Timeline
La TimelinePage SHALL proposer un mode Matrice accessible via un toggle [Changes | Matrice] en haut de page. Ce mode affiche une grille spec × bucket temporel représentant l'intensité d'activité de chaque spec dans le temps, ainsi qu'un panel droit de détail spec ouvert au clic.

#### Scenario: Activation du mode Matrice
- **WHEN** l'utilisateur clique sur "Matrice" dans le toggle de la TimelinePage
- **THEN** la grille spec × bucket temporel s'affiche, chaque ligne correspondant à une spec, chaque colonne à un bucket temporel (jour, semaine, mois ou trimestre selon la granularité sélectionnée) de la période couverte par les changes

#### Scenario: Mode Changes actif par défaut
- **WHEN** l'utilisateur navigue vers `/timeline`
- **THEN** le mode Changes est actif par défaut

#### Scenario: Deep-link vers Matrice avec spec pré-sélectionnée
- **WHEN** l'URL contient le paramètre `?spec=<name>`
- **THEN** le mode Matrice s'active et le panel droit s'ouvre directement sur la spec nommée

### Requirement: Grille spec × date avec intensité colorée
La grille du mode Matrice SHALL afficher une ligne par spec (de `openspec/specs/`) et une colonne par bucket temporel couvert par au moins un change, à la granularité sélectionnée. Chaque cellule indique par une intensité de couleur relative le nombre de changes ayant touché cette spec dans ce bucket.

#### Scenario: Cellule vide
- **WHEN** aucun change n'a touché une spec dans un bucket temporel donné
- **THEN** la cellule est vide (fond neutre)

#### Scenario: Intensité proportionnelle au maximum observé
- **WHEN** au moins une cellule de la vue courante contient des changes
- **THEN** chaque cellule non vide est colorée selon une échelle relative au nombre maximum de changes observé dans la vue courante (plus une cellule est proche de ce maximum, plus l'intensité est forte)

#### Scenario: Cellule au maximum observé
- **WHEN** une cellule contient le nombre de changes le plus élevé de la vue courante
- **THEN** elle est affichée avec l'intensité de couleur la plus forte de l'échelle

#### Scenario: Nombre exact au survol
- **WHEN** l'utilisateur survole une cellule non vide
- **THEN** une infobulle affiche le nombre exact de changes pour ce bucket, quelle que soit son intensité de couleur

#### Scenario: Specs sans aucun change
- **WHEN** une spec n'a aucun change lié dans tout l'historique
- **THEN** la ligne de la spec s'affiche avec un style atténué ou un indicateur ⚠

## ADDED Requirements

### Requirement: Sélection de la granularité temporelle
Le mode Matrice SHALL proposer un sélecteur de granularité temporelle (Jour, Semaine, Mois, Trimestre) permettant de choisir l'unité de regroupement des colonnes de la grille.

#### Scenario: Changement de granularité
- **WHEN** l'utilisateur sélectionne une granularité différente dans le sélecteur
- **THEN** la grille recalcule ses colonnes en regroupant les changes par la nouvelle granularité et se réaffiche

#### Scenario: Granularités disponibles
- **WHEN** l'utilisateur ouvre le sélecteur de granularité
- **THEN** les options proposées sont exactement Jour, Semaine, Mois et Trimestre

### Requirement: Regroupement des changes par bucket temporel
Pour une granularité donnée, la grille SHALL regrouper les changes de chaque spec par bucket temporel : jour calendaire, semaine ISO 8601 (lundi à dimanche), mois calendaire, ou trimestre calendaire (Q1 janvier-mars, Q2 avril-juin, Q3 juillet-septembre, Q4 octobre-décembre). Le nombre de changes affiché dans une cellule est la somme des changes de la spec dans ce bucket.

#### Scenario: Regroupement par semaine ISO
- **WHEN** la granularité "Semaine" est sélectionnée
- **THEN** chaque colonne représente une semaine ISO 8601 identifiée par son année-semaine (semaine du lundi au dimanche), et une cellule agrège tous les changes d'une spec tombant dans cette semaine

#### Scenario: Regroupement par trimestre
- **WHEN** la granularité "Trimestre" est sélectionnée
- **THEN** chaque colonne représente un trimestre calendaire et une cellule agrège tous les changes d'une spec tombant dans ce trimestre

#### Scenario: Tri chronologique des colonnes
- **WHEN** la grille affiche ses colonnes pour une granularité donnée
- **THEN** les colonnes sont triées du bucket le plus récent au plus ancien

### Requirement: Largeur des colonnes adaptée à l'espace disponible
La grille SHALL calculer la largeur des colonnes pour remplir la largeur disponible du conteneur, dans une fourchette lisible, plutôt que d'utiliser une largeur fixe ou un nombre de colonnes plafonné arbitrairement.

#### Scenario: La grille tient dans l'espace disponible
- **WHEN** le nombre de buckets à afficher, à la largeur minimale lisible, tient dans la largeur disponible du conteneur
- **THEN** la largeur de chaque colonne s'étire pour remplir exactement l'espace disponible, sans dépasser une largeur maximale lisible

#### Scenario: La grille dépasse l'espace disponible
- **WHEN** le nombre de buckets à afficher ne tient pas dans la largeur disponible même à la largeur minimale lisible
- **THEN** les colonnes sont affichées à leur largeur minimale et un défilement horizontal permet d'accéder aux buckets plus anciens, en affichant le bucket le plus récent en premier

### Requirement: Granularité par défaut au chargement
Au premier affichage du mode Matrice, la grille SHALL sélectionner automatiquement la granularité la plus fine dont l'ensemble des buckets de l'historique complet tient sans défilement horizontal dans la largeur mesurée du conteneur. L'utilisateur SHALL pouvoir ensuite changer manuellement de granularité.

#### Scenario: Sélection automatique à l'ouverture
- **WHEN** l'utilisateur ouvre le mode Matrice pour la première fois dans une session d'affichage
- **THEN** la granularité initiale est la plus fine parmi Jour, Semaine, Mois, Trimestre dont tous les buckets de l'historique tiennent dans la largeur mesurée sans défilement horizontal

#### Scenario: Redimensionnement sans changement de granularité
- **WHEN** la largeur disponible du conteneur change après l'affichage initial (redimensionnement de la fenêtre, ouverture ou fermeture du panel droit de détail)
- **THEN** la granularité sélectionnée ne change pas ; seule la largeur des colonnes se réajuste dans les bornes lisibles, avec défilement horizontal si nécessaire
