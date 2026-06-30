## ADDED Requirements

### Requirement: Toggle de mode Contenu/Historique
La vue Specs SHALL proposer un toggle [Contenu | Historique] en haut de page permettant de basculer entre la vue de contenu existante et la nouvelle vue Historique.

#### Scenario: Activation du mode Historique
- **WHEN** l'utilisateur clique sur "Historique" dans le toggle
- **THEN** la vue bascule vers le mode Historique affichant la timeline des changes par spec
- **AND** le panneau de contenu de spec et la TOC sont remplacés par la vue historique

#### Scenario: Retour au mode Contenu
- **WHEN** l'utilisateur clique sur "Contenu" dans le toggle
- **THEN** la vue revient au mode Contenu avec la liste de sélection de spec et l'affichage du contenu

#### Scenario: Mode Contenu actif par défaut
- **WHEN** l'utilisateur navigue vers la vue Specs
- **THEN** le mode Contenu est actif par défaut
