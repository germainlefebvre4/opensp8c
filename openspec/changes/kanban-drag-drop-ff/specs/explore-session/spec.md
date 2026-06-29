## ADDED Requirements

### Requirement: Fermeture de session explore lors du déclenchement ff
Quand un drag `to-explore → todo` est confirmé pour un changement dont une session explore est active (ExplorePanel ouvert), le frontend SHALL fermer l'ExplorePanel et terminer la session explore (`DELETE /changes/{name}/explore`) avant de déclencher le ff (`POST /changes/{name}/ff`). Ces deux actions SHALL être séquentielles.

#### Scenario: Drag vers todo avec ExplorePanel actif
- **WHEN** l'utilisateur drop une carte **To Explore** vers **To Do** et qu'un ExplorePanel est ouvert pour ce changement
- **THEN** le frontend appelle `DELETE /changes/{name}/explore`, attend la réponse, puis appelle `POST /changes/{name}/ff`

#### Scenario: Drag vers todo sans ExplorePanel actif
- **WHEN** l'utilisateur drop une carte **To Explore** vers **To Do** et qu'aucun ExplorePanel n'est ouvert
- **THEN** le frontend appelle directement `POST /changes/{name}/ff` sans DELETE préalable
