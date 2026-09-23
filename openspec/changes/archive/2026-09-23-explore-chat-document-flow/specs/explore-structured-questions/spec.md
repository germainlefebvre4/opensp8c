# Spec Delta

## MODIFIED Requirements

### Requirement: Affichage d'une question sous forme de carte détachée

Le frontend SHALL afficher toute question détectée dans une carte visuellement distincte du reste du fil de conversation : fond blanc, bordure et ombre légère, séparée des tours de texte ordinaires du flux plein-largeur (`explore-message-layout`) qui n'ont ni fond ni bordure.

#### Scenario: Rendu d'une question
- **WHEN** un événement de question est reçu par le frontend
- **THEN** une carte dédiée s'affiche dans le fil de chat, à la place d'un tour de texte ordinaire, contenant le texte de la question
