## MODIFIED Requirements

### Requirement: Visibilité des erreurs subprocess
Le backend SHALL capturer le stderr du subprocess `claude` ou `gemini` et logguer chaque ligne avec un préfixe identifiable. Aucune erreur subprocess ne SHALL être silencieusement ignorée. De plus, pour les erreurs critiques empêchant le fonctionnement du service (telles que `TerminalQuotaError`, `Failed to connect to IDE companion extension`, ou `ProjectIdRequiredError`), le backend SHALL transmettre un avertissement structuré `session_warning` au frontend.

#### Scenario: Erreur de démarrage visible dans les logs
- **WHEN** le subprocess écrit sur stderr (erreur d'authentification, flag inconnu, crash)
- **THEN** chaque ligne stderr est loggée par le backend avec le préfixe `[subprocess stderr]` suivi du contenu

#### Scenario: Subprocess sain sans stderr
- **WHEN** le subprocess fonctionne normalement et ne produit rien sur stderr
- **THEN** aucun log stderr n'est émis (pas de bruit dans les logs)

#### Scenario: Alerte d'erreur critique d'authentification de projet transmise au client
- **WHEN** le subprocess écrit sur stderr un message contenant `ProjectIdRequiredError` ou `GOOGLE_CLOUD_PROJECT`
- **THEN** le backend génère et envoie un événement de type `session_warning` avec un message d'explication guidant l'utilisateur sur la définition des variables d'environnement requises
