## ADDED Requirements

### Requirement: Bouton delete dans le panel d'exploration d'un ghost card
Le panel d'exploration d'un ghost card SHALL afficher un bouton delete dans son header. Ce bouton déclenche une dialog de confirmation avant suppression.

#### Scenario: Bouton delete visible dans le header du panel ghost
- **WHEN** le panel d'exploration est ouvert pour un ghost card (`is_ghost: true`)
- **THEN** le header affiche un bouton delete (icône 🗑) à côté du bouton de fermeture

#### Scenario: Bouton delete absent pour les sessions named non-ghost
- **WHEN** le panel d'exploration est ouvert pour un change normal en "to-explore" (non-ghost)
- **THEN** aucun bouton delete n'est affiché dans le header

#### Scenario: Clic sur delete — dialog de confirmation
- **WHEN** l'utilisateur clique sur le bouton delete dans le header du panel ghost
- **THEN** une dialog s'affiche avec le texte "Abandonner cette exploration ?" et "La conversation sera perdue." et deux boutons : [Annuler] et [Abandonner]

#### Scenario: Confirmation de suppression — panel fermé + ghost supprimé
- **WHEN** l'utilisateur clique sur [Abandonner] dans la dialog de confirmation
- **THEN** le panel se ferme, la session backend est stoppée, le ghost record est supprimé de `preferences.json` via `DELETE /api/workspaces/{id}/explorations/{ghostId}`, et localStorage est nettoyé

#### Scenario: Annulation de suppression — panel reste ouvert
- **WHEN** l'utilisateur clique sur [Annuler] dans la dialog de confirmation
- **THEN** la dialog se ferme et le panel d'exploration reste ouvert sans modification
