## MODIFIED Requirements

### Requirement: Persistance des ghost records au redémarrage serveur
Le backend SHALL charger les ghost records depuis `preferences.json` au démarrage. L'endpoint de listing des changes SHALL inclure les ghost records dans la réponse.

#### Scenario: Ghost records chargés au démarrage
- **WHEN** le serveur démarre et des ghost records existent dans `preferences.json`
- **THEN** l'API `/api/workspaces/{id}/changes` retourne les ghost records parmi les changes, avec un champ `kanban_status: "to-explore"` et `is_ghost: true`

#### Scenario: Ghost record absent du workspace actif
- **WHEN** un ghost record référence un workspaceId différent du workspace courant
- **THEN** ce ghost record n'apparaît pas dans la liste des changes du workspace courant

#### Scenario: Restauration du nom du ghost card au chargement/reprise
- **WHEN** l'utilisateur reprend une exploration existante via `resumeGhostId`
- **THEN** le frontend SHALL restaurer réactivement le nom réel du ghost card (`ghostName`) depuis la liste des changes existants (`useChanges`), ce qui permet au bouton de création de change de s'afficher correctement
