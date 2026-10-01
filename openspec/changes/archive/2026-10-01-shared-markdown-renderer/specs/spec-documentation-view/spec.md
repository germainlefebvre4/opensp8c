# Spec Delta

## MODIFIED Requirements

### Requirement: Rendu Markdown avec diagrammes Mermaid
Le contenu d'une page de documentation sélectionnée SHALL être rendu en Markdown, y compris les tableaux et le code inline conformément à la capacité `markdown-rendering`, et tout bloc de code de langage `mermaid` qu'il contient SHALL être rendu comme un diagramme visuel plutôt que comme un bloc de code brut.

#### Scenario: Page contenant un diagramme Mermaid valide
- **WHEN** l'utilisateur sélectionne une page de documentation contenant un bloc ```mermaid``` valide
- **THEN** le diagramme correspondant est rendu visuellement dans le panneau de contenu

#### Scenario: Bloc Mermaid invalide
- **WHEN** un bloc ```mermaid``` présent dans une page ne peut pas être interprété
- **THEN** la vue affiche le bloc de code brut en repli, sans interrompre l'affichage du reste de la page

#### Scenario: Page contenant un tableau et du code inline
- **WHEN** l'utilisateur sélectionne une page de documentation contenant un tableau Markdown et du code inline
- **THEN** le tableau est affiché comme un tableau visuel et le code inline est affiché stylé, sans pipes ni backticks visibles
