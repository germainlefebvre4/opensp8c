# Spec Delta

## MODIFIED Requirements

### Requirement: Solidification du change brouillon
La solidification (ou "figer") d'un change brouillon par l'utilisateur (soit explicitement, soit par action implicite telle que la modification d'une tâche) SHALL détruire définitivement le ghost d'exploration associé et le fichier de brouillon pour finaliser le change. Le drag-and-drop d'une carte ne SHALL PAS solidifier un change : aucune transition de drag n'est autorisée vers la colonne "in-progress" (voir `kanban-drag-drop`).

#### Scenario: Clic sur "Figer" dans la carte ou le DetailPanel
- **WHEN** l'utilisateur clique sur le bouton "Figer" du change brouillon dans la colonne "ready" ou "todo", ou son DetailPanel
- **THEN** le frontend appelle `DELETE /api/workspaces/{id}/explorations/{ghostId}`, le backend supprime le ghost de `preferences.json`, détruit le fichier `drafts/<ghostId>.json`, et émet l'event SSE `exploration_deleted` pour faire disparaître le ghost card de "to-explore"

#### Scenario: Modification implicite d'une tâche fige le change
- **WHEN** l'utilisateur coche ou modifie une tâche d'un change brouillon dans le DetailPanel
- **THEN** le frontend déclenche silencieusement la suppression du ghost associé pour nettoyer l'espace d'exploration, rendant le change solide de manière transparente

#### Scenario: Passage à l'état In Progress fige le change
- **WHEN** l'utilisateur tente de drag-and-drop le change brouillon de la colonne "todo" vers "in-progress"
- **THEN** le drop est refusé (voir `kanban-drag-drop`), le ghost associé n'est pas supprimé et le change reste brouillon dans "todo"
