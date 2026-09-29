# Spec Delta

## ADDED Requirements

### Requirement: Langue de la documentation générée
Le run de génération SHALL produire les pages de documentation dans la langue `documentation` résolue au moment du lancement, telle que définie par `agent-language-settings`. Les noms de fichiers des pages (`overview.md`, `architecture.md`, `domain-model.md`, `workflows.md`) et le chemin `docs/opensp8c/` SHALL rester inchangés quelle que soit la langue.

#### Scenario: Documentation dans la langue de l'application
- **WHEN** la langue `documentation` vaut `auto`, la langue de l'application est `fr`, et une génération aboutit
- **THEN** les pages sous `docs/opensp8c/` sont rédigées en français

#### Scenario: Documentation dans une langue explicite
- **WHEN** la langue `documentation` vaut `en` alors que la langue de l'application est `fr`, et une génération aboutit
- **THEN** les pages sous `docs/opensp8c/` sont rédigées en anglais

#### Scenario: Noms de fichiers indépendants de la langue
- **WHEN** une génération aboutit avec une langue `documentation` autre que l'anglais
- **THEN** les pages portent exactement les noms de fichiers fixes définis pour la génération
