## ADDED Requirements

### Requirement: Barre de recherche au-dessus des colonnes Kanban
Le Kanban Board SHALL afficher une barre de recherche textuelle en haut de la zone de colonnes, pleine largeur, toujours visible lorsque le Kanban est affiché. La barre comprend un champ texte avec placeholder et un bouton de réinitialisation (`×`) visible uniquement quand la recherche est non vide.

#### Scenario: Barre visible au chargement du Kanban
- **WHEN** l'utilisateur ouvre le Kanban Board
- **THEN** la barre de recherche est affichée au-dessus des colonnes, avec un champ vide et sans bouton `×`

#### Scenario: Bouton de réinitialisation visible
- **WHEN** l'utilisateur a saisi au moins un caractère dans le champ de recherche
- **THEN** un bouton `×` est visible dans ou à côté du champ

### Requirement: Filtrage instantané des changes par nom
La saisie dans la barre de recherche SHALL filtrer instantanément les changes affichés dans toutes les colonnes Kanban (actives et Archived) sur `change.name`, de façon insensible à la casse. Seuls les changes dont le nom contient la chaîne saisie sont affichés.

#### Scenario: Filtre par sous-chaîne
- **WHEN** l'utilisateur saisit "auth" dans la barre de recherche
- **THEN** seules les cartes dont le nom contient "auth" (ex. "add-user-auth", "oauth-refresh") sont visibles dans leurs colonnes respectives

#### Scenario: Filtre insensible à la casse
- **WHEN** l'utilisateur saisit "AUTH"
- **THEN** les mêmes cartes que pour "auth" sont affichées

#### Scenario: Filtre sans résultat
- **WHEN** la chaîne saisie ne correspond à aucun change dans le workspace
- **THEN** toutes les colonnes sont affichées vides (aucune carte), avec leur badge count à 0

#### Scenario: Colonne active à zéro résultat
- **WHEN** le filtre exclut tous les changes d'une colonne active (ex. In Progress)
- **THEN** la colonne reste visible avec son badge count à 0 et son état vide

#### Scenario: Colonne Archived filtrée
- **WHEN** l'utilisateur saisit un terme de recherche
- **THEN** la colonne Archived n'affiche que les changes archivés dont le nom correspond au filtre, au même titre que les colonnes actives

### Requirement: Réinitialisation du filtre
L'utilisateur SHALL pouvoir réinitialiser le filtre, ce qui rétablit l'affichage complet de toutes les colonnes.

#### Scenario: Clic sur le bouton de réinitialisation
- **WHEN** l'utilisateur clique sur le bouton `×`
- **THEN** le champ de recherche est vidé, le bouton `×` disparaît, et toutes les cartes sont de nouveau affichées

#### Scenario: Suppression manuelle du texte
- **WHEN** l'utilisateur efface manuellement tout le texte du champ de recherche
- **THEN** toutes les cartes sont de nouveau affichées et le bouton `×` disparaît
