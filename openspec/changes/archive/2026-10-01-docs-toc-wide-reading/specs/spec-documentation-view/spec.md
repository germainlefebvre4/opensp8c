# Spec Delta

## ADDED Requirements

### Requirement: Table des matières de la page de documentation
Le sous-onglet "Documentation" SHALL afficher, à droite du contenu de la page sélectionnée, une table des matières listant les titres de niveau 2 et 3 de cette page, dans l'ordre du document. Le titre de niveau 1 (titre de la page) SHALL NOT y figurer. Chaque entrée SHALL être cliquable et faire défiler le contenu jusqu'à la section correspondante, et l'entrée de la section en cours de lecture SHALL être mise en évidence au fil du défilement. La table des matières SHALL se mettre à jour lorsque l'utilisateur change de page ou que le contenu est régénéré, et SHALL NOT être affichée lorsque la page ne contient aucun titre de niveau 2 ou 3.

#### Scenario: Clic sur une entrée
- **WHEN** l'utilisateur clique sur une entrée de la table des matières
- **THEN** le contenu défile jusqu'au titre correspondant et cette entrée est mise en évidence

#### Scenario: Suivi de la lecture
- **WHEN** l'utilisateur fait défiler le contenu de la page
- **THEN** l'entrée de la table des matières correspondant à la section visible est mise en évidence

#### Scenario: Changement de page
- **WHEN** l'utilisateur sélectionne une autre page de documentation
- **THEN** la table des matières affiche les titres de la nouvelle page

#### Scenario: Titre contenant du code inline
- **WHEN** un titre de la page contient du code inline (par exemple ``Agent role settings (`preferences` + `agents`)``)
- **THEN** son entrée dans la table des matières est cliquable et conduit à ce titre

#### Scenario: Ligne commençant par `#` dans un bloc de code
- **WHEN** un bloc de code de la page contient une ligne commençant par `#`
- **THEN** cette ligne n'apparaît pas dans la table des matières

#### Scenario: Page sans sous-titres
- **WHEN** la page affichée ne contient aucun titre de niveau 2 ou 3
- **THEN** aucune table des matières n'est affichée

### Requirement: Table des matières adaptative à la zone de lecture
La table des matières du sous-onglet "Documentation" SHALL avoir une largeur fluide, bornée par une largeur minimale et une largeur maximale, qui s'adapte à la largeur disponible de la zone du sous-onglet (et non à celle de la fenêtre du navigateur). Lorsque la zone devient trop étroite pour conserver au texte principal une largeur de lecture minimale à côté de la table des matières, celle-ci SHALL être masquée et le texte principal SHALL occuper l'espace libéré.

#### Scenario: Zone large
- **WHEN** la zone du sous-onglet est assez large pour afficher la liste des pages, le texte principal à sa largeur de lecture minimale et la table des matières à sa largeur minimale
- **THEN** la table des matières est affichée à droite du contenu, avec une largeur comprise entre ses bornes minimale et maximale

#### Scenario: Zone rétrécie
- **WHEN** la zone du sous-onglet est réduite (redimensionnement de la fenêtre ou ouverture de la barre latérale de l'application) en dessous du seuil de lecture
- **THEN** la table des matières est masquée et le texte principal occupe toute la largeur disponible

#### Scenario: Zone agrandie
- **WHEN** la zone du sous-onglet est élargie au-delà du seuil de lecture après avoir été masquée
- **THEN** la table des matières réapparaît sans rechargement de la page

### Requirement: Largeur de lecture élargie de la documentation
Le contenu d'une page de documentation (texte, tableaux, blocs de code et diagrammes) SHALL pouvoir occuper une largeur supérieure à celle de la mise en page précédente, jusqu'à un plafond d'environ 1024 px, et SHALL se réduire avec la zone lorsque celle-ci est plus étroite. Les diagrammes SHALL occuper au plus la largeur du contenu, sans être agrandis au-delà de leur taille naturelle.

#### Scenario: Grand écran
- **WHEN** la zone du sous-onglet est plus large que le plafond de largeur du contenu
- **THEN** le contenu occupe au plus ce plafond et n'est pas limité à l'ancienne largeur d'environ 768 px

#### Scenario: Diagramme plus large que l'ancienne mise en page
- **WHEN** une page contient un diagramme dont la taille naturelle dépasse environ 768 px et que la zone est assez large
- **THEN** le diagramme est rendu plus grand qu'avant, dans la limite de sa taille naturelle et de la largeur du contenu

#### Scenario: Petit diagramme
- **WHEN** une page contient un diagramme dont la taille naturelle est inférieure à la largeur du contenu
- **THEN** le diagramme conserve sa taille naturelle
