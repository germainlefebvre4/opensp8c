## ADDED Requirements

### Requirement: Bouton de maximisation dans le header du panel d'exploration
Le header du panel d'exploration (qu'il soit named ou anonymous) SHALL afficher un bouton de maximisation/minimisation (icônes Maximize2 / Minimize2) à côté des boutons de mode de rendu et avant le bouton de fermeture (X).

#### Scenario: Clic sur maximiser agrandit le panel
- **WHEN** le panel est en taille normale et l'utilisateur clique sur le bouton de maximisation
- **THEN** le panel passe en mode maximisé, occupant toute la hauteur de l'espace disponible sous la barre de navigation globale, et l'icône du bouton change pour indiquer le retour à la taille normale

#### Scenario: Clic sur minimiser restaure la taille normale
- **WHEN** le panel est en mode maximisé et l'utilisateur clique sur le bouton de minimisation
- **THEN** le panel quitte le mode maximisé et reprend sa hauteur redimensionnée précédente (ou la hauteur par défaut)

#### Scenario: Le drag handle est désactivé en mode maximisé
- **WHEN** le panel est en mode maximisé
- **THEN** le drag handle du bord supérieur n'est plus draggable et l'icône de curseur de redimensionnement vertical n'est pas affichée

## MODIFIED Requirements

### Requirement: Redimensionnement vertical du bottom panel
L'utilisateur SHALL pouvoir ajuster la hauteur du bottom panel via un drag handle positionné sur son bord supérieur. La hauteur SHALL être contrainte entre 200px (minimum) et 90% de la hauteur de la fenêtre (maximum).

#### Scenario: Drag vers le haut agrandit le panel
- **WHEN** l'utilisateur clique sur le drag handle et déplace la souris vers le haut
- **THEN** la hauteur du panel augmente au fur et à mesure du déplacement, dans la limite du maximum (90% de la hauteur de la fenêtre)

#### Scenario: Drag vers le bas rétrécit le panel
- **WHEN** l'utilisateur clique sur le drag handle et déplace la souris vers le bas
- **THEN** la hauteur du panel diminue au fur et à mesure du déplacement, dans la limite du minimum (200px)

#### Scenario: Relâchement de la souris fixe la hauteur
- **WHEN** l'utilisateur relâche le bouton de la souris pendant un drag
- **THEN** la hauteur du panel est fixée à la valeur courante et le drag s'arrête
