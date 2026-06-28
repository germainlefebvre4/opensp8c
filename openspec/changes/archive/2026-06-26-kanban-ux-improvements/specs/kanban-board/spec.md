## MODIFIED Requirements

### Requirement: Afficher la carte d'un changement
Chaque changement SHALL être représenté par une carte épurée affichant uniquement : le nom du changement et la progression des tasks (barre de progression + compteur). La carte ne SHALL PAS afficher de boutons d'action inline — toutes les actions sont accessibles via le panneau latéral (DetailPanel ou ExplorePanel) ouvert au clic sur la carte.

#### Scenario: Carte sans tasks.md
- **WHEN** le changement n'a pas encore de fichier `tasks.md`
- **THEN** la progression est affichée comme "0 / 0 tasks" sans erreur

#### Scenario: Carte avec tasks.md
- **WHEN** le changement a un fichier `tasks.md` contenant des items `- [ ]` et `- [x]`
- **THEN** la carte affiche "N / M tasks" où N est le nombre de `[x]` et M le total

### Requirement: Ouvrir l'ExplorePanel au clic sur une carte To Explore
L'utilisateur SHALL pouvoir cliquer sur une carte dans la colonne **To Explore** pour ouvrir l'ExplorePanel (session de conversation). Le bouton "Explorer" existant est supprimé — la carte entière est la zone cliquable.

#### Scenario: Clic sur carte en colonne To Explore
- **WHEN** l'utilisateur clique sur une carte dans la colonne **To Explore**
- **THEN** l'ExplorePanel s'ouvre pour ce change, permettant d'entamer ou reprendre une session de conversation

### Requirement: Colonnes Kanban pleine hauteur
Les colonnes Kanban SHALL occuper toute la hauteur disponible de la zone de contenu, indépendamment du nombre de cartes qu'elles contiennent.

#### Scenario: Colonne vide
- **WHEN** une colonne ne contient aucune carte
- **THEN** la colonne s'étend sur toute la hauteur disponible et reste une zone de dépôt valide

#### Scenario: Colonnes de hauteurs différentes
- **WHEN** les colonnes contiennent des nombres différents de cartes
- **THEN** toutes les colonnes ont la même hauteur (celle de la colonne la plus haute ou de la zone disponible)

### Requirement: Application pleine largeur avec colonnes auto-adaptées
Le Kanban Board SHALL occuper toute la largeur disponible de la zone de contenu. Les colonnes SHALL se distribuer équitablement dans cette largeur, chaque colonne prenant une fraction égale de l'espace disponible.

#### Scenario: Redimensionnement de la fenêtre
- **WHEN** l'utilisateur redimensionne la fenêtre du navigateur
- **THEN** les colonnes s'adaptent automatiquement pour remplir la largeur disponible sans débordement horizontal

## REMOVED Requirements

### Requirement: Changer le statut depuis la carte
**Reason**: Les boutons d'action migrent dans le DetailPanel pour épurer la carte. Le drag & drop reste disponible pour les transitions rapides.
**Migration**: Utiliser le clic sur la carte pour ouvrir le DetailPanel, puis utiliser les boutons de transition de statut dans le panneau.
