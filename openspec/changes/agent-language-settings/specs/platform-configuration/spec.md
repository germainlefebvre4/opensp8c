# Spec Delta

## ADDED Requirements

### Requirement: Réglage des langues d'agents dans Configuration > Langue
Le système SHALL exposer, dans l'onglet "Langue" de Configuration, en plus du sélecteur de langue de l'interface, trois réglages : langue du chat, langue de la documentation (documentation générée et artefacts OpenSpec) et langue du code. Les réglages du chat et de la documentation SHALL proposer, en plus des langues supportées, une option "suivre la langue de l'application" (`auto`) ; le réglage du code SHALL proposer uniquement les langues supportées. Les langues proposées SHALL provenir d'une liste unique partagée par les trois réglages. Les valeurs SHALL être persistées via `PATCH /api/preferences`, comme décrit par `agent-language-settings`.

#### Scenario: Affichage des valeurs par défaut
- **WHEN** l'utilisateur ouvre Configuration > Langue sans réglage de langue d'agent enregistré
- **THEN** le chat et la documentation affichent l'option "suivre la langue de l'application"
- **THEN** le code affiche l'anglais

#### Scenario: Option auto indiquant la langue résolue
- **WHEN** le chat ou la documentation vaut `auto`
- **THEN** l'interface indique la langue actuellement résolue (celle de l'application)

#### Scenario: Le code n'offre pas d'option auto
- **WHEN** l'utilisateur ouvre la liste du réglage de la langue du code
- **THEN** seules les langues supportées sont proposées, sans option "suivre la langue de l'application"

#### Scenario: Enregistrement d'un réglage
- **WHEN** l'utilisateur change la langue de la documentation en français et enregistre
- **THEN** `preferences.json` reflète ce réglage et `GET /api/preferences` le retourne
- **THEN** les réglages du chat et du code restent inchangés

#### Scenario: Ajout d'une langue supportée
- **WHEN** une nouvelle langue est ajoutée à la liste des langues supportées
- **THEN** elle apparaît dans les trois listes de Configuration > Langue sans autre modification de l'écran
