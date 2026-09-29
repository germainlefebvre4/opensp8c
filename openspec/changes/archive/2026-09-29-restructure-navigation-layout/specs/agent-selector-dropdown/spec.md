# Spec Delta

## MODIFIED Requirements

### Requirement: Dropdown visible malgré overflow-hidden des ancêtres
Le dropdown de sélection d'agent SHALL être entièrement visible même lorsque ses ancêtres portent `overflow: hidden`. Il SHALL utiliser `position: fixed` pour s'affranchir du clipping CSS. Le sélecteur étant placé dans la barre principale, le dropdown SHALL avoir une largeur minimale suffisante pour afficher le nom et la version de chaque agent sans troncature, et SHALL être aligné sur le bord droit de son bouton sans déborder de la fenêtre.

#### Scenario: Ouverture dans le sidebar
- **WHEN** l'utilisateur clique sur le bouton d'agent dans la barre principale
- **THEN** le dropdown complet (avec tous les agents) est visible à l'écran, sans être tronqué

#### Scenario: Alignement sous le bouton
- **WHEN** le dropdown s'ouvre
- **THEN** son bord supérieur est positionné directement sous le bouton (4px de marge) et son bord droit est aligné sur celui du bouton

#### Scenario: Bouton plus étroit que la liste
- **WHEN** le bouton d'agent est plus étroit que la largeur minimale du dropdown
- **THEN** le dropdown s'affiche à sa largeur minimale, sans troncature des libellés ni débordement hors de la fenêtre
