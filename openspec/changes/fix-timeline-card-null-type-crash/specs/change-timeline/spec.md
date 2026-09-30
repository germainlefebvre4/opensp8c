## ADDED Requirements

### Requirement: Robustesse de l'entrée timeline face à des tags incomplets
Une entrée de la timeline SHALL s'afficher sans erreur lorsque les tags du change n'ont pas de `type` exploitable (champ absent, `null` ou tableau vide). Dans ce cas, aucun badge de type n'est rendu, et le reste de l'entrée (nom, statut, date, complexité, chips de specs et de composants) reste affiché. Un tag incomplet sur un change ne SHALL jamais empêcher l'affichage de la page Timeline ni des autres entrées.

#### Scenario: Tags sans `type`
- **WHEN** un change a une section `tags` dont le champ `type` est absent ou `null`
- **THEN** l'entrée s'affiche sans badge de type et sans erreur, avec ses autres métadonnées

#### Scenario: Tags avec `type` vide
- **WHEN** un change a `tags.type: []`
- **THEN** l'entrée s'affiche sans badge de type et sans erreur

#### Scenario: Un change à tags incomplets parmi d'autres
- **WHEN** la timeline contient un change à tags incomplets et d'autres changes correctement tagués
- **THEN** tous les changes sont affichés et la page reste utilisable
