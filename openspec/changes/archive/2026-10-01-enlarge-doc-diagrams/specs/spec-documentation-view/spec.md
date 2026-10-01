# Spec Delta

## ADDED Requirements

### Requirement: Agrandissement d'un diagramme au clic
Dans le sous-onglet "Documentation", un diagramme Mermaid rendu visuellement SHALL pouvoir être agrandi : un clic sur le diagramme (ou son activation au clavier) SHALL l'ouvrir dans une fenêtre modale occupant l'essentiel de l'écran, où le diagramme est mis à l'échelle pour remplir l'espace disponible en conservant ses proportions, y compris lorsque sa taille naturelle est inférieure à cet espace. La modale SHALL pouvoir être fermée par la touche Échap, par un bouton de fermeture explicite et par un clic sur le fond, et la fermeture SHALL laisser la page de documentation dans l'état où elle se trouvait (page sélectionnée et position de défilement inchangées). Le diagramme en ligne SHALL signaler qu'il est cliquable (curseur adapté, focus clavier visible) et porter un libellé accessible traduit dans la langue de l'interface. Un bloc Mermaid affiché en repli comme bloc de code brut SHALL NOT être cliquable.

#### Scenario: Ouverture au clic
- **WHEN** l'utilisateur clique sur un diagramme Mermaid rendu dans une page de documentation
- **THEN** une modale s'ouvre et affiche ce diagramme, agrandi pour occuper l'essentiel de l'espace de l'écran

#### Scenario: Ouverture au clavier
- **WHEN** l'utilisateur place le focus sur un diagramme Mermaid rendu et active la touche Entrée ou Espace
- **THEN** la modale d'agrandissement s'ouvre sur ce diagramme

#### Scenario: Petit diagramme
- **WHEN** l'utilisateur agrandit un diagramme dont la taille naturelle est inférieure à l'espace de la modale
- **THEN** le diagramme est agrandi au-delà de sa taille naturelle, proportions conservées, pour remplir l'espace disponible

#### Scenario: Fermeture par Échap
- **WHEN** la modale est ouverte et que l'utilisateur appuie sur la touche Échap
- **THEN** la modale se ferme et la page de documentation est affichée telle qu'avant l'ouverture

#### Scenario: Fermeture par le bouton ou le fond
- **WHEN** la modale est ouverte et que l'utilisateur clique sur le bouton de fermeture ou sur le fond situé hors du diagramme
- **THEN** la modale se ferme et la page de documentation est affichée telle qu'avant l'ouverture

#### Scenario: Plusieurs diagrammes dans une page
- **WHEN** une page contient plusieurs diagrammes et que l'utilisateur clique sur l'un d'eux
- **THEN** la modale n'affiche que le diagramme cliqué

#### Scenario: Bloc Mermaid invalide
- **WHEN** un bloc ```mermaid``` ne peut pas être interprété et est affiché en repli comme bloc de code brut
- **THEN** ce bloc n'est pas cliquable et aucune modale ne s'ouvre à son clic
