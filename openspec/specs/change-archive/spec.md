## Purpose

Permettre à l'utilisateur d'archiver un changement terminé (colonne Done) directement depuis l'UI Kanban, en délégant la commande `openspec archive` au backend.

## Requirements

### Requirement: Archiver un changement depuis l'UI
L'utilisateur SHALL pouvoir déclencher l'archivage d'un changement en colonne **Done** via le bouton **"Sync & Archive"** qui apparaît au survol de la carte, ou via le point d'entrée équivalent du DetailPanel. Cliquer sur ce bouton SHALL d'abord ouvrir un dialog de confirmation partagé (nom du change, rappel que l'action synchronise les specs et archive le change de façon irréversible, boutons "Annuler" / "Archiver"). L'appel au backend n'est déclenché qu'après confirmation explicite dans le dialog. Le backend SHALL exécuter `openspec archive <name> --yes` dans le répertoire du workspace actif, sans interaction utilisateur supplémentaire. Cette action n'est pas disponible pour les changements déjà archivés.

#### Scenario: Confirmation demandée avant archivage
- **WHEN** l'utilisateur clique sur "Sync & Archive" au survol d'une carte en colonne Done
- **THEN** un dialog de confirmation s'ouvre et affiche le nom du change et un avertissement sur le caractère irréversible de l'action, sans qu'aucun appel au backend n'ait encore été déclenché

#### Scenario: Annulation de la confirmation
- **WHEN** l'utilisateur clique sur "Annuler" dans le dialog de confirmation
- **THEN** le dialog se ferme, aucun appel au backend n'est effectué, et la carte reste inchangée dans la colonne Done

#### Scenario: Archivage réussi
- **WHEN** l'utilisateur confirme l'archivage dans le dialog et que la commande réussit
- **THEN** le backend exécute `openspec archive <name> --yes`, le dialog se ferme, la carte disparaît de la colonne Done et le Kanban se rafraîchit

#### Scenario: Archivage avec tasks non finalisées
- **WHEN** l'utilisateur confirme l'archivage dans le dialog et que `openspec archive <name> --yes` retourne une erreur de validation
- **THEN** le backend capture la sortie de la commande et l'affiche dans le dialog sous forme de message d'erreur, le dialog reste ouvert et la carte reste dans la colonne Done

#### Scenario: Changement déjà archivé
- **WHEN** un changement est dans la colonne Archived
- **THEN** aucun bouton "Sync & Archive" n'est affiché, ni sur la carte ni dans le DetailPanel

#### Scenario: Archivage avec sync de specs requis
- **WHEN** `openspec archive <name> --yes` doit synchroniser les specs avant d'archiver
- **THEN** la synchronisation est effectuée automatiquement par la commande (le flag `--yes` skip toutes les confirmations) et l'archivage se complète sans intervention utilisateur

### Requirement: Feedback de progression et de succès de l'archivage
Le dialog de confirmation SHALL afficher un indicateur de chargement pendant l'exécution de `openspec archive` et informer l'utilisateur du résultat (succès ou erreur). En cas de succès, un toast SHALL confirmer l'archivage après la fermeture du dialog.

#### Scenario: Indicateur pendant l'archivage
- **WHEN** l'archivage est en cours après confirmation
- **THEN** le dialog affiche un spinner avec le texte "Synchronisation et archivage en cours...", les boutons "Annuler" et "Archiver" sont désactivés, et le dialog ne peut pas être fermé

#### Scenario: Succès de l'archivage
- **WHEN** la commande `openspec archive` se termine avec succès
- **THEN** le dialog se ferme, un toast affiche "Change « <nom> » archivé", la carte disparaît de la colonne Done, et le Kanban se met à jour (colonne Archived incluse)

#### Scenario: Échec de l'archivage
- **WHEN** la commande `openspec archive` se termine avec une erreur
- **THEN** le dialog reste ouvert, un message d'erreur est affiché dans le dialog (sortie CLI capturée), un bouton "Réessayer" est disponible, et la carte reste dans la colonne Done
