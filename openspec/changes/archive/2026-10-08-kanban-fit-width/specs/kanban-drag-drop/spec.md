# Spec Delta

## ADDED Requirements

### Requirement: Rail Done/Archived cible de drop valide
Lorsque le slot Done/Archived est replié en rail, la colonne **Done** SHALL rester une cible de drop valide pour les transitions autorisées vers Done. Pendant un drag dont la source est autorisée à rejoindre Done, le rail SHALL se signaler comme cible valide avec le même traitement visuel que les autres colonnes valides (surbrillance, plus accentuée au survol) et afficher le libellé de la colonne Done, sans modifier la largeur du slot ni déplacer les autres colonnes. Pendant un drag dont la source n'est pas autorisée à rejoindre Done, le rail SHALL ne présenter aucune surbrillance. Lorsque le DetailPanel est en overlay, il SHALL s'effacer pendant la durée d'un drag afin que toutes les colonnes soient des cibles de drop accessibles. Les règles existantes de transitions autorisées et de drop refusé vers Done (tâches restantes à valider) s'appliquent inchangées.

#### Scenario: Drop sur le rail replié
- **WHEN** le slot Done/Archived est replié en rail et l'utilisateur dépose depuis To Review une carte autorisée à rejoindre Done sur le rail
- **THEN** la transition vers Done s'applique comme sur une colonne Done dépliée

#### Scenario: Rail signalé comme cible valide pendant un drag
- **WHEN** l'utilisateur fait glisser une carte dont la transition vers Done est autorisée alors que le rail est replié
- **THEN** le rail est mis en surbrillance comme cible valide et affiche le libellé Done, sans que la largeur du slot ni la position des autres colonnes ne change

#### Scenario: Aucune surbrillance pour une transition interdite
- **WHEN** l'utilisateur fait glisser une carte dont la transition vers Done n'est pas autorisée alors que le rail est replié
- **THEN** le rail n'est pas mis en surbrillance et le drop est refusé comme sur une colonne Done dépliée

#### Scenario: Overlay du DetailPanel pendant un drag
- **WHEN** le DetailPanel est en overlay et l'utilisateur démarre un drag
- **THEN** le panel s'efface pendant le drag et réapparaît à sa fin
