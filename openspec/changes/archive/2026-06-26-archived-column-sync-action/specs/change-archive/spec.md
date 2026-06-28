## MODIFIED Requirements

### Requirement: Archiver un changement depuis l'UI
L'utilisateur SHALL pouvoir déclencher l'archivage d'un changement en colonne **Done** via le bouton **"Sync & Archive"** qui apparaît au survol de la carte. Le backend SHALL exécuter `openspec archive <name> --yes` dans le répertoire du workspace actif, sans interaction utilisateur. Cette action n'est pas disponible pour les changements déjà archivés.

#### Scenario: Archivage réussi
- **WHEN** l'utilisateur clique sur "Sync & Archive" au survol d'une carte en colonne Done et que la commande réussit
- **THEN** le backend exécute `openspec archive <name> --yes`, la carte disparaît de la colonne Done et le Kanban se rafraîchit

#### Scenario: Archivage avec tasks non finalisées
- **WHEN** l'utilisateur clique sur "Sync & Archive" et que `openspec archive <name> --yes` retourne une erreur de validation
- **THEN** le backend capture la sortie de la commande et l'affiche sur la carte sous forme de message d'erreur, la carte reste dans la colonne Done

#### Scenario: Archivage avec sync de specs requis
- **WHEN** `openspec archive <name> --yes` doit synchroniser les specs avant d'archiver
- **THEN** la synchronisation est effectuée automatiquement (le flag `--yes` skip toutes les confirmations) et l'archivage se complète sans intervention utilisateur

#### Scenario: Changement déjà archivé
- **WHEN** un changement est dans la colonne Archived
- **THEN** aucun bouton "Sync & Archive" n'est affiché, ni sur la carte ni dans le DetailPanel

### Requirement: Feedback de progression de l'archivage
L'UI SHALL afficher un indicateur de chargement sur la carte pendant l'exécution de `openspec archive` et informer l'utilisateur du résultat (succès ou erreur) directement sur la carte.

#### Scenario: Indicateur pendant l'archivage
- **WHEN** l'archivage est en cours
- **THEN** un spinner est affiché sur la carte et le bouton "Sync & Archive" est désactivé ; la carte n'est pas cliquable

#### Scenario: Succès de l'archivage
- **WHEN** la commande `openspec archive` se termine avec succès
- **THEN** la carte disparaît de la colonne Done ; le Kanban se rafraîchit et la colonne Archived se met à jour

#### Scenario: Échec de l'archivage
- **WHEN** la commande `openspec archive` se termine avec une erreur
- **THEN** un message d'erreur est affiché sur la carte (sortie CLI capturée), un bouton "Réessayer" est disponible, et la carte reste dans la colonne Done
