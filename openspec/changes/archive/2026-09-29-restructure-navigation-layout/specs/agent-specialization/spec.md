# Spec Delta

## MODIFIED Requirements

### Requirement: Écran de configuration pour gérer les extensions
Le système SHALL afficher un écran "Settings" comme onglet du sous-menu du workspace, distinct des écrans Kanban/Specs/Timeline/Agents, permettant de visualiser la liste de base (lecture seule) et de gérer (ajouter/retirer) les tags de spécialisation personnalisés.

#### Scenario: Affichage de l'écran Settings
- **WHEN** l'utilisateur navigue vers l'écran "Settings"
- **THEN** la liste de base est affichée en lecture seule, et les tags personnalisés existants sont affichés avec une action de suppression pour chacun, et un champ permettant d'en ajouter un nouveau

#### Scenario: Ajout invalide depuis l'écran
- **WHEN** l'utilisateur saisit une valeur qui n'est pas au format kebab-case dans le champ d'ajout de l'écran Settings
- **THEN** l'écran affiche une erreur de validation et n'envoie pas la requête au backend
