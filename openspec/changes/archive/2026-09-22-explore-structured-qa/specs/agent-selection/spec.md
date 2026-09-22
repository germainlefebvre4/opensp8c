# Spec Delta

## ADDED Requirements

### Requirement: Réglage global du mode question native

Le système SHALL exposer, dans `AgentSettingsModal`, une bascule globale "mode question native (Claude)" permettant d'activer ou de désactiver le mode question native décrit par `explore-native-question-mode`. Cette préférence SHALL être persistée dans `preferences.json` sous un champ booléen dédié, désactivée par défaut, et exposée par les endpoints existants `GET`/`PATCH /api/preferences` aux côtés des champs déjà présents (`defaultAgent`, `env`). L'interface SHALL indiquer que ce réglage n'a d'effet que pour l'agent Claude.

#### Scenario: Affichage de la bascule
- **WHEN** l'utilisateur ouvre `AgentSettingsModal`
- **THEN** une bascule "mode question native (Claude)" est visible, accompagnée d'une mention indiquant qu'elle est sans effet pour les autres agents

#### Scenario: Activation de la bascule
- **WHEN** l'utilisateur active la bascule et enregistre
- **THEN** `preferences.json` est mis à jour avec le champ booléen à `true`, et `GET /api/preferences` reflète ce changement

#### Scenario: Valeur par défaut
- **WHEN** `preferences.json` est créé pour la première fois ou que le champ n'a jamais été défini
- **THEN** le mode question native est considéré comme désactivé
