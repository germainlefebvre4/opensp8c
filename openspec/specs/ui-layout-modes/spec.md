## Purpose

Gérer les modes de mise en page de l'interface : mode page (avec sidebar) et mode fullpage (sidebar masquée), ainsi que la possibilité de réduire la sidebar en mode icônes uniquement.

## Requirements

### Requirement: Toggle page/fullpage
L'application SHALL proposer un bouton de toggle dans la barre de navigation permettant de basculer entre le mode "page" (sidebar visible) et le mode "fullpage" (sidebar masquée, contenu étendu sur toute la largeur).

#### Scenario: Activation du mode fullpage
- **WHEN** l'utilisateur clique sur le bouton fullpage dans la navigation
- **THEN** la sidebar workspace disparaît et le contenu principal occupe toute la largeur de la fenêtre

#### Scenario: Retour au mode page
- **WHEN** l'utilisateur est en mode fullpage et clique sur le bouton page mode
- **THEN** la sidebar workspace réapparaît et le layout revient à l'état initial

#### Scenario: Mode éphémère
- **WHEN** l'utilisateur recharge la page
- **THEN** le layout revient au mode page par défaut (le mode fullpage n'est pas persisté)

### Requirement: Sidebar collapsible (icône toggle)
La sidebar SHALL pouvoir être réduite via un bouton icône visible en haut de la sidebar, la faisant passer à une version icônes uniquement (largeur réduite) sans perdre la navigation.

#### Scenario: Collapse de la sidebar
- **WHEN** l'utilisateur clique sur l'icône toggle de la sidebar
- **THEN** la sidebar se réduit pour ne montrer que les icônes des workspaces (sans les noms)

#### Scenario: Expand de la sidebar
- **WHEN** la sidebar est en mode réduit et l'utilisateur clique sur l'icône toggle
- **THEN** la sidebar revient à sa largeur normale avec noms et bouton "Ajouter"
