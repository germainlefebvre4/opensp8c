# Spec Delta

## MODIFIED Requirements

### Requirement: Écran de configuration pour gérer les extensions
Le système SHALL afficher, dans le sous-onglet « Spécialisations » de l'écran "Settings" scopé au workspace actif (voir `workspace-settings`), la liste de base (lecture seule) et la gestion (ajout/retrait) des tags de spécialisation personnalisés. Le vocabulaire de ces tags SHALL rester global à la plateforme et non propre à un workspace.

#### Scenario: Affichage de l'écran Settings
- **WHEN** l'utilisateur ouvre le sous-onglet Spécialisations de l'écran Settings
- **THEN** la liste de base est affichée en lecture seule, et les tags personnalisés existants sont affichés avec une action de suppression pour chacun, et un champ permettant d'en ajouter un nouveau

#### Scenario: Ajout invalide depuis l'écran
- **WHEN** l'utilisateur saisit une valeur qui n'est pas au format kebab-case dans le champ d'ajout du sous-onglet Spécialisations
- **THEN** l'écran affiche une erreur de validation et n'envoie pas la requête au backend

#### Scenario: Vocabulaire commun aux workspaces
- **WHEN** l'utilisateur ajoute un tag personnalisé depuis Settings du workspace `A`
- **THEN** ce tag apparaît aussi dans Settings > Spécialisations du workspace `B`
