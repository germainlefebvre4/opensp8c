## ADDED Requirements

### Requirement: Réordonner les workspaces par drag-and-drop
L'utilisateur SHALL pouvoir réordonner les workspaces dans la sidebar par glisser-déposer. Le nouvel ordre SHALL être persisté côté serveur dans `config.yaml` et reflété à tous les rechargements ultérieurs.

#### Scenario: Déplacement d'un workspace vers le haut
- **WHEN** l'utilisateur glisse un workspace au-dessus d'un autre dans la sidebar
- **THEN** le workspace apparaît à sa nouvelle position immédiatement (optimistic update) et l'ordre est persisté via `PATCH /api/workspaces/order`

#### Scenario: Déplacement d'un workspace vers le bas
- **WHEN** l'utilisateur glisse un workspace en dessous d'un autre dans la sidebar
- **THEN** le workspace apparaît à sa nouvelle position immédiatement et l'ordre est persisté via `PATCH /api/workspaces/order`

#### Scenario: Annulation du drag (drop hors zone)
- **WHEN** l'utilisateur commence un drag et relâche hors d'une zone de drop valide
- **THEN** l'ordre reste inchangé et aucune requête serveur n'est envoyée

#### Scenario: Erreur de persistance serveur
- **WHEN** `PATCH /api/workspaces/order` retourne une erreur
- **THEN** la liste revient à l'ordre serveur (rollback via refetch React Query)

### Requirement: Persistance de l'ordre des workspaces côté serveur
Le backend SHALL exposer un endpoint `PATCH /api/workspaces/order` qui réordonne les workspaces dans `config.yaml` selon la liste d'IDs fournie.

#### Scenario: Payload valide
- **WHEN** `PATCH /api/workspaces/order` reçoit `{ "order": ["id1", "id2"] }` avec des IDs correspondant exactement aux workspaces existants
- **THEN** le serveur réordonne `config.yaml`, sauvegarde, et retourne `204 No Content`

#### Scenario: Payload avec IDs inconnus ou incomplets
- **WHEN** `PATCH /api/workspaces/order` reçoit une liste d'IDs ne correspondant pas exactement aux workspaces connus
- **THEN** le serveur retourne `400 Bad Request` et `config.yaml` reste inchangé
