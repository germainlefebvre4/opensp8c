## MODIFIED Requirements

### Requirement: Table des Matières (TOC) sticky
La vue Specs SHALL afficher une Table des Matières générée automatiquement à partir des headings (h1, h2, h3) du contenu Markdown, dans un panneau sticky à droite du contenu, uniquement en mode lecture.

#### Scenario: Génération de la TOC
- **WHEN** l'utilisateur sélectionne une spec contenant des headings Markdown
- **AND** la vue est en mode lecture
- **THEN** la TOC affiche la liste hiérarchisée des titres (h1, h2, h3) avec indentation selon le niveau

#### Scenario: Spec sans headings
- **WHEN** la spec sélectionnée ne contient aucun heading Markdown
- **THEN** la TOC n'est pas affichée (le panneau droit disparaît)

#### Scenario: TOC masquée en mode édition
- **WHEN** la vue est en mode édition
- **THEN** le panneau TOC n'est pas affiché
- **AND** l'espace est occupé par le panneau diff de l'éditeur
