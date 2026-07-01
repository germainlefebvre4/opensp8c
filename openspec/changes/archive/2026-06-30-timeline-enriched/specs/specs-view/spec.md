## REMOVED Requirements

### Requirement: Toggle de mode Contenu/Historique
**Reason**: Le mode Historique migre vers la TimelinePage (mode Matrice). La SpecsPage redevient une vue de contenu uniquement. Le lien vers l'historique est remplacé par une navigation explicite vers la Timeline.
**Migration**: Naviguer vers `/timeline?workspace=<id>&spec=<name>` pour accéder à l'historique des changes d'une spec.

## ADDED Requirements

### Requirement: Lien de navigation vers l'historique d'une spec dans la Timeline
La vue Specs SHALL afficher un lien "Voir l'historique →" dans le panneau de détail d'une spec sélectionnée, permettant à l'utilisateur de naviguer directement vers la Timeline en mode Matrice avec cette spec pré-sélectionnée.

#### Scenario: Clic sur "Voir l'historique →"
- **WHEN** l'utilisateur a sélectionné une spec et clique sur le lien "Voir l'historique →"
- **THEN** l'application navigue vers `/timeline?workspace=<id>&spec=<spec-name>`, ouvrant la Timeline en mode Matrice avec le panel de cette spec affiché

#### Scenario: Lien absent si aucune spec sélectionnée
- **WHEN** aucune spec n'est sélectionnée dans la liste
- **THEN** le lien "Voir l'historique →" n'est pas affiché
