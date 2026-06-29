## MODIFIED Requirements

### Requirement: Timeout de session inactive
Une session d'exploration SHALL être automatiquement terminée après 30 minutes d'inactivité (aucun message envoyé ni reçu).

#### Scenario: Session expirée par inactivité
- **WHEN** aucun message n'a été échangé pendant 30 minutes
- **THEN** le subprocess est terminé et le panneau affiche un message "Session expirée — cliquez pour reprendre"

#### Scenario: Reprise après expiration (named session)
- **WHEN** l'utilisateur clique pour reprendre après expiration d'une named session
- **THEN** le backend relance le subprocess avec `--resume <claudeSessionId>` et Claude reprend la conversation là où elle s'était arrêtée

#### Scenario: Reprise après expiration (session sans historique)
- **WHEN** l'utilisateur clique pour reprendre après expiration ET aucun `claudeSessionId` n'est présent dans preferences
- **THEN** le backend lance un nouveau subprocess sans `--resume`, comme actuellement
