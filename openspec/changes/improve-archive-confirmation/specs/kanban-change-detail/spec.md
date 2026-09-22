# Spec Delta

## MODIFIED Requirements

### Requirement: Archiver un change depuis le DetailPanel
L'utilisateur SHALL pouvoir déclencher l'archivage d'un change en statut **Done** depuis le DetailPanel, via le même dialog de confirmation partagé que celui utilisé depuis la carte Kanban.

#### Scenario: Confirmation demandée avant archivage
- **WHEN** l'utilisateur clique sur "Sync & Archive" dans le DetailPanel d'un change en statut **Done**
- **THEN** le dialog de confirmation partagé s'ouvre, affichant le nom du change et l'avertissement d'irréversibilité, sans appel au backend

#### Scenario: Annulation de la confirmation
- **WHEN** l'utilisateur clique sur "Annuler" dans le dialog ouvert depuis le DetailPanel
- **THEN** le dialog se ferme, aucun appel au backend n'est effectué, et le DetailPanel reste ouvert et inchangé

#### Scenario: Archivage depuis le panneau
- **WHEN** l'utilisateur confirme l'archivage dans le dialog ouvert depuis le DetailPanel
- **THEN** le change est archivé, le dialog se ferme, un toast confirme l'archivage, et le DetailPanel se ferme

#### Scenario: Erreur d'archivage
- **WHEN** l'archivage échoue
- **THEN** le dialog reste ouvert, le message d'erreur est affiché dans le dialog avec un bouton "Réessayer", et le DetailPanel reste ouvert
