# Spec Delta

## MODIFIED Requirements

### Requirement: Accès à Configuration indépendant du workspace
Le système SHALL exposer une entrée de navigation "Configuration" dans le panneau de gauche, au-dessus du bloc "Projets" (sélecteur d'agent par défaut, liste des projets), distincte de la liste d'onglets liés au workspace actif (Kanban, Specs, Timeline, Agents, Réglages) affichée dans la barre de navigation du haut, et accessible même lorsqu'aucun workspace n'est configuré.

#### Scenario: Accès sans workspace configuré
- **WHEN** aucun workspace n'est configuré dans l'application
- **THEN** l'entrée de navigation Configuration est visible
- **THEN** son activation affiche la page Configuration

#### Scenario: Séparation visuelle des onglets liés au workspace
- **WHEN** l'utilisateur navigue vers Configuration
- **THEN** l'URL ne porte pas de paramètre de workspace actif, contrairement aux onglets Kanban/Specs/Timeline/Agents/Réglages

#### Scenario: Emplacement dans le panneau de gauche
- **WHEN** l'utilisateur consulte l'application
- **THEN** l'entrée de navigation Configuration est affichée en haut du panneau de gauche, au-dessus du bloc "Projets"
- **THEN** elle n'est plus affichée dans la barre de navigation du haut

### Requirement: Sidebar conservée sur Configuration
Le système SHALL conserver l'affichage du panneau de gauche (entrée Configuration, sélecteur d'agent par défaut, liste des projets) lorsque la page Configuration est affichée, et SHALL mettre en évidence l'entrée Configuration comme active.

#### Scenario: Sidebar visible sur Configuration
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** le panneau de gauche (entrée Configuration, sélecteur d'agent par défaut, liste des projets) reste affiché à gauche de l'écran

#### Scenario: Entrée Configuration mise en évidence
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** l'entrée de navigation Configuration dans le panneau de gauche est visuellement marquée comme active

## ADDED Requirements

### Requirement: Accessibilité de Configuration en panneau réduit
Le système SHALL permettre d'accéder à Configuration même lorsque le panneau de gauche est réduit, via une icône dédiée qui reste visible et cliquable indépendamment du reste du contenu du panneau (sélecteur d'agent par défaut, liste des projets), qui lui reste masqué en mode réduit.

#### Scenario: Icône Configuration visible en mode réduit
- **WHEN** le panneau de gauche est réduit
- **THEN** une icône Configuration reste visible et cliquable dans la bande réduite
- **THEN** le sélecteur d'agent par défaut et la liste des projets restent masqués

#### Scenario: Activation depuis le mode réduit
- **WHEN** l'utilisateur clique sur l'icône Configuration alors que le panneau de gauche est réduit
- **THEN** la page Configuration s'affiche
- **THEN** l'état réduit du panneau de gauche est conservé (pas de réouverture automatique)

### Requirement: Sous-onglet Langue dans Configuration
Le système SHALL exposer, dans Configuration, un sous-onglet "Langue" aux côtés des sous-onglets "Agents" et "CLI", hébergeant le sélecteur de langue décrit par la capacité `i18n-core`.

#### Scenario: Affichage du sous-onglet Langue
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** un sous-onglet "Langue" est visible aux côtés des sous-onglets "Agents" et "CLI"

#### Scenario: Contenu du sous-onglet Langue
- **WHEN** l'utilisateur active le sous-onglet "Langue"
- **THEN** le sélecteur de langue (EN | FR) s'affiche
