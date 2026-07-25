## MODIFIED Requirements

### Requirement: Dialogue de confirmation de promotion depuis le volet
Le clic sur le bouton de promotion du volet d'exploration SHALL ouvrir la même boîte de dialogue de confirmation que le drag-and-drop, permettant de modifier le nom et de confirmer ou annuler la promotion.

#### Scenario: Validation de la promotion depuis le dialogue
- **WHEN** l'utilisateur clique sur le bouton de promotion, confirme/modifie le nom du change dans le dialogue, et clique sur [Créer le change]
- **THEN** le dialogue et le volet d'exploration se ferment, l'état maximisé du volet est réinitialisé à false, la carte correspondante passe en état "FF running", et l'appel API `POST /api/workspaces/{id}/explorations/{ghostId}/promote` est envoyé

## ADDED Requirements

### Requirement: Réinitialisation de l'état maximisé à l'abandon d'une exploration
L'action d'abandonner/supprimer une exploration fantôme SHALL réinitialiser l'état maximisé du volet d'exploration à false pour garantir que le tableau Kanban redevienne visible.

#### Scenario: Abandon d'exploration réinitialise l'état maximisé
- **WHEN** l'utilisateur supprime une exploration (via le bouton de suppression de la carte ou du volet) et confirme l'abandon
- **THEN** le volet d'exploration se ferme, la carte d'exploration est supprimée, et l'état maximisé du volet est réinitialisé à false
