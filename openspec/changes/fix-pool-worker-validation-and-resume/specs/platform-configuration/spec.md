# Spec Delta

## ADDED Requirements

### Requirement: Commande de validation par défaut dans Configuration
Configuration > Agent Pool SHALL permettre de définir une commande de validation par défaut (`validationCommand`, texte libre) exécutée par les workers du pool après chaque invocation de l'agent. Cette valeur SHALL être persistée dans `preferences.json` avec les autres défauts du pool, retournée par `GET /api/preferences`, et servir de base à tous les workspaces qui ne la surchargent pas dans Settings. À défaut de valeur enregistrée, le champ SHALL rester vide et les workers SHALL utiliser l'auto-détection de la commande de validation.

#### Scenario: Aucun défaut enregistré
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool sans commande de validation enregistrée
- **THEN** le champ est vide et indique que la commande est auto-détectée

#### Scenario: Défaut enregistré
- **WHEN** l'utilisateur saisit `make test` comme commande de validation par défaut et enregistre
- **THEN** la valeur est persistée dans `preferences.json`, retournée par `GET /api/preferences`, et utilisée par les workers de tout workspace sans surcharge

#### Scenario: Effacement du défaut
- **WHEN** l'utilisateur vide le champ et enregistre
- **THEN** la valeur enregistrée est supprimée et les workers reviennent à l'auto-détection
