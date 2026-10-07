# Spec Delta

## ADDED Requirements

### Requirement: La colonne Verifying est pilotée par le système
Les cartes de la colonne **Verifying** SHALL NE PAS être draggables, et la colonne SHALL NE PAS être une cible de drop valide : aucune transition par drag-and-drop ne SHALL partir de `verifying` ni y arriver. Les actions sur un change en vérification (relancer, finaliser, demander des corrections) SHALL passer par les boutons de la carte et du DetailPanel. Pendant le drag d'une autre carte, la colonne Verifying SHALL être présentée comme une cible invalide.

#### Scenario: Drag d'une carte Verifying
- **WHEN** l'utilisateur tente de saisir une carte de la colonne Verifying
- **THEN** le drag ne démarre pas

#### Scenario: Drop sur Verifying
- **WHEN** l'utilisateur dépose une carte de To Do ou d'In Progress sur la colonne Verifying
- **THEN** le drop est refusé et la carte retourne à sa colonne d'origine

#### Scenario: Cible invalide pendant un drag
- **WHEN** l'utilisateur fait glisser une carte au-dessus de la colonne Verifying
- **THEN** la colonne n'est pas surlignée comme cible valide
