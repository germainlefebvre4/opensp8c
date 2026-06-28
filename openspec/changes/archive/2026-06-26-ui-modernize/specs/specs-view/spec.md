## MODIFIED Requirements

### Requirement: Lister les spécifications actées
L'application SHALL afficher la liste de toutes les spécifications actées du workspace actif, lues depuis `openspec/specs/`. Chaque spec est représentée par son nom (nom du répertoire) et le chemin vers son fichier `spec.md`.

#### Scenario: Workspace avec des specs
- **WHEN** l'utilisateur ouvre la vue Specs d'un workspace contenant des entrées dans `openspec/specs/`
- **THEN** la liste affiche toutes les specs trouvées, triées alphabétiquement par nom

#### Scenario: Workspace sans specs
- **WHEN** le dossier `openspec/specs/` est vide ou absent
- **THEN** la vue affiche un message "Aucune spécification actée pour ce workspace"

### Requirement: Consulter le contenu d'une spécification
L'utilisateur SHALL pouvoir sélectionner une spec dans la liste pour en afficher le contenu Markdown rendu, aligné à gauche, avec des styles typographiques lisibles.

#### Scenario: Affichage du contenu d'une spec
- **WHEN** l'utilisateur clique sur une spec dans la liste
- **THEN** le contenu de son fichier `spec.md` est affiché en Markdown rendu dans un panneau de détail, avec texte aligné à gauche et styles prose (titres hiérarchisés, listes, code)

#### Scenario: Pas de chevauchement typographique
- **WHEN** un titre h1 ou h2 du Markdown dépasse la largeur du panneau de contenu
- **THEN** le texte revient à la ligne sans chevauchement ni débordement

## ADDED Requirements

### Requirement: Table des Matières (TOC) sticky
La vue Specs SHALL afficher une Table des Matières générée automatiquement à partir des headings (h1, h2, h3) du contenu Markdown, dans un panneau sticky à droite du contenu.

#### Scenario: Génération de la TOC
- **WHEN** l'utilisateur sélectionne une spec contenant des headings Markdown
- **THEN** la TOC affiche la liste hiérarchisée des titres (h1, h2, h3) avec indentation selon le niveau

#### Scenario: Spec sans headings
- **WHEN** la spec sélectionnée ne contient aucun heading Markdown
- **THEN** la TOC n'est pas affichée (le panneau droit disparaît)

### Requirement: Navigation par TOC
L'utilisateur SHALL pouvoir cliquer sur un item de la TOC pour naviguer directement à la section correspondante dans le contenu.

#### Scenario: Clic sur un item de TOC
- **WHEN** l'utilisateur clique sur un item de la Table des Matières
- **THEN** le panneau de contenu défile pour afficher la section correspondante en haut de vue

### Requirement: Mise en évidence de la section active dans la TOC
La TOC SHALL mettre en évidence l'item correspondant à la section de la spec actuellement visible à l'écran.

#### Scenario: Scroll dans le contenu
- **WHEN** l'utilisateur fait défiler le contenu de la spec
- **THEN** l'item de TOC correspondant à la section visible en haut de l'écran est mis en évidence visuellement (couleur ou indicateur)

#### Scenario: Item actif change au scroll
- **WHEN** l'utilisateur fait défiler jusqu'à une nouvelle section
- **THEN** la mise en évidence se déplace vers l'item de TOC de la nouvelle section visible
