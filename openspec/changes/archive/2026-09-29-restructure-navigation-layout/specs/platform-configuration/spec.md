# Spec Delta

## MODIFIED Requirements

### Requirement: Accès à Configuration indépendant du workspace
Le système SHALL exposer une entrée de navigation "Configuration" dans la barre principale globale, distincte du sous-menu des onglets liés au workspace actif (Kanban, Specs, Timeline, Agents, Réglages), et accessible même lorsqu'aucun workspace n'est configuré.

#### Scenario: Accès sans workspace configuré
- **WHEN** aucun workspace n'est configuré dans l'application
- **THEN** l'entrée de navigation Configuration est visible
- **THEN** son activation affiche la page Configuration

#### Scenario: Séparation visuelle des onglets liés au workspace
- **WHEN** l'utilisateur navigue vers Configuration
- **THEN** l'URL ne porte pas de paramètre de workspace actif, contrairement aux onglets Kanban/Specs/Timeline/Agents/Réglages

#### Scenario: Emplacement dans le panneau de gauche
- **WHEN** l'utilisateur consulte l'application
- **THEN** l'entrée de navigation Configuration est affichée dans la barre principale en haut de la fenêtre
- **THEN** elle n'est affichée ni dans le panneau de gauche ni dans le sous-menu du workspace

## REMOVED Requirements

### Requirement: Sidebar conservée sur Configuration
**Reason**: La page Configuration s'affiche désormais en pleine largeur, sans sidebar des projets ni sous-menu du workspace (voir `app-navigation-layout`).
**Migration**: Utiliser l'entrée « Projects » de la barre principale pour revenir à la liste des projets.

### Requirement: Accessibilité de Configuration en panneau réduit
**Reason**: L'entrée Configuration ne se trouve plus dans le panneau de gauche mais dans la barre principale, toujours visible ; il n'y a plus de cas où elle est masquée par le repli du panneau.
**Migration**: Aucune ; l'entrée Configuration de la barre principale est accessible en permanence.
