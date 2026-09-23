# Spec Delta

## MODIFIED Requirements

### Requirement: Filtrage instantané des changes par nom et par tags
La saisie dans la barre de recherche SHALL filtrer instantanément les changes affichés dans toutes les colonnes Kanban (actives et Archived) sur `change.name` et sur les tags du change (`type` et `components`), de façon insensible à la casse. Un change est retenu si son nom contient la chaîne saisie, ou si au moins une valeur de son tableau `tags.type`, ou au moins une valeur de son tableau `tags.components`, contient la chaîne saisie.

#### Scenario: Filtre par sous-chaîne sur le nom
- **WHEN** l'utilisateur saisit "auth" dans la barre de recherche
- **THEN** seules les cartes dont le nom contient "auth" (ex. "add-user-auth", "oauth-refresh") sont visibles dans leurs colonnes respectives

#### Scenario: Filtre insensible à la casse
- **WHEN** l'utilisateur saisit "AUTH"
- **THEN** les mêmes cartes que pour "auth" sont affichées

#### Scenario: Filtre par type applicatif
- **WHEN** l'utilisateur saisit "frontend" dans la barre de recherche
- **THEN** les cartes dont le tableau `tags.type` contient une valeur correspondant à "frontend" sont affichées, en plus des cartes dont le nom contient "frontend"

#### Scenario: Filtre par type applicatif sur un change à plusieurs types
- **WHEN** l'utilisateur saisit "backend" et qu'un change a `tags.type: [frontend, backend]`
- **THEN** la carte de ce change est affichée, car "backend" correspond à l'une des valeurs de son tableau `type`

#### Scenario: Filtre par composant
- **WHEN** l'utilisateur saisit "explore-panel" dans la barre de recherche
- **THEN** les cartes dont le tableau `components` contient un slug correspondant à "explore-panel" sont affichées

#### Scenario: Filtre sans résultat
- **WHEN** la chaîne saisie ne correspond ni à un nom ni à un tag d'aucun change du workspace
- **THEN** toutes les colonnes sont affichées vides (aucune carte), avec leur badge count à 0

#### Scenario: Colonne active à zéro résultat
- **WHEN** le filtre exclut tous les changes d'une colonne active (ex. In Progress)
- **THEN** la colonne reste visible avec son badge count à 0 et son état vide

#### Scenario: Colonne Archived filtrée
- **WHEN** l'utilisateur saisit un terme de recherche
- **THEN** la colonne Archived n'affiche que les changes archivés dont le nom ou un tag correspond au filtre, au même titre que les colonnes actives
