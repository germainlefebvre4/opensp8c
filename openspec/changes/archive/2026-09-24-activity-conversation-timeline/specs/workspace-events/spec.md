# Spec Delta

## ADDED Requirements

### Requirement: Événement SSE d'ajout d'une entrée d'activité
Le backend SHALL émettre, sur le même stream SSE `/api/workspaces/{id}/events`, un événement `activity_appended` chaque fois qu'une nouvelle entrée est ajoutée à l'`ActivityStore` d'un change (toggle de tâche, déclenchement de run, reset de tâches, transition de statut worker, commit git détecté). Cet événement inclut le nom du change concerné et ne SHALL PAS être soumis au debounce de 150ms utilisé pour les événements liés aux fichiers.

#### Scenario: Ajout d'une entrée d'activité
- **WHEN** une nouvelle entrée est ajoutée à l'`ActivityStore` du change `<changeName>`
- **THEN** le serveur envoie `event: activity_appended\ndata: {"name":"<changeName>"}\n\n` à tous les clients abonnés au stream de ce workspace

#### Scenario: Réception côté client
- **WHEN** le frontend reçoit un événement `activity_appended` pour le change actuellement affiché dans le DetailPanel
- **THEN** il invalide et recharge le flux fusionné (`GET /changes/{name}/activity`) pour refléter la nouvelle entrée dans l'onglet Conversation
