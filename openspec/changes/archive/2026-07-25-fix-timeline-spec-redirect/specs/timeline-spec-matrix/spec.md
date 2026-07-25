## MODIFIED Requirements

### Requirement: Navigation depuis le panel de spec vers la SpecsPage
Le panel de spec du mode Matrice SHALL proposer un lien "Voir la spec →" permettant de naviguer vers le contenu de la spec dans la SpecsPage.

#### Scenario: Clic sur le lien "Voir la spec →"
- **WHEN** l'utilisateur clique sur "Voir la spec →" dans le panel de spec
- **THEN** l'application navigue vers `/specs?workspace=<id>&selected=<name>` affichant le contenu de la spec
