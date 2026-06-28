## ADDED Requirements

### Requirement: Archiver un changement depuis l'UI
L'utilisateur SHALL pouvoir déclencher l'archivage d'un changement en colonne **Done** via un bouton dans la carte. Le backend SHALL exécuter `openspec archive <name> --yes` dans le répertoire du workspace actif, sans interaction utilisateur.

#### Scenario: Archivage réussi
- **WHEN** l'utilisateur clique sur "Archiver" dans une carte en colonne Done et que toutes les tasks sont complètes
- **THEN** le backend exécute `openspec archive <name> --yes`, la commande réussit, et la carte disparaît du Kanban

#### Scenario: Archivage avec tasks non finalisées
- **WHEN** l'utilisateur clique sur "Archiver" et que `openspec archive <name> --yes` retourne une erreur de validation (tasks non complètes)
- **THEN** le backend capture la sortie de la commande et l'affiche dans la carte sous forme de message d'erreur en rouge, listant les tasks manquantes. La carte reste dans la colonne Done.

#### Scenario: Archivage avec sync de specs requis
- **WHEN** `openspec archive <name> --yes` doit synchroniser les specs avant d'archiver
- **THEN** la synchronisation est effectuée automatiquement par la commande (le flag `--yes` skip toutes les confirmations) et l'archivage se complète sans intervention utilisateur

### Requirement: Feedback de progression de l'archivage
L'UI SHALL afficher un indicateur de chargement pendant l'exécution de `openspec archive` et informer l'utilisateur du résultat (succès ou erreur).

#### Scenario: Indicateur pendant l'archivage
- **WHEN** l'archivage est en cours
- **THEN** le bouton "Archiver" est désactivé et un spinner est affiché sur la carte

#### Scenario: Succès de l'archivage
- **WHEN** la commande `openspec archive` se termine avec succès
- **THEN** un message de succès est brièvement affiché avant que la carte disparaisse du Kanban
