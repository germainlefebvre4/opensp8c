# Spec Delta

## ADDED Requirements

### Requirement: Événement SSE de mise à jour de l'état du pool d'agents
Le backend SHALL émettre, sur le même stream SSE `/api/workspaces/{id}/events`, un événement `pool_updated` chaque fois que l'état du pool d'agents change : démarrage du pool, arrêt du pool, ou changement de statut d'un worker (`idle`, `working`, `testing`, `healing`, `paused`) ou de son change assigné. Cet événement est déclenché par les transitions d'état en mémoire du gestionnaire de pool, indépendamment de toute modification de fichier sur le filesystem, et ne SHALL PAS être soumis au debounce de 150ms utilisé pour les événements liés aux fichiers.

#### Scenario: Démarrage du pool
- **WHEN** le pool d'agents démarre suite à un appel `POST /workspaces/{id}/pool/start`
- **THEN** le serveur envoie `event: pool_updated\ndata: {}\n\n` à tous les clients abonnés au stream de ce workspace

#### Scenario: Arrêt du pool
- **WHEN** le pool d'agents s'arrête suite à un appel `POST /workspaces/{id}/pool/stop`
- **THEN** le serveur envoie `event: pool_updated\ndata: {}\n\n` à tous les clients abonnés au stream de ce workspace

#### Scenario: Changement de statut d'un worker
- **WHEN** un worker du pool change de statut (par exemple de `working` à `testing`, ou se voit assigner un nouveau change)
- **THEN** le serveur envoie `event: pool_updated\ndata: {}\n\n` à tous les clients abonnés au stream de ce workspace

#### Scenario: Réception côté client
- **WHEN** le frontend reçoit un événement `pool_updated` sur le stream SSE
- **THEN** il invalide et recharge l'état du pool (`GET /workspaces/{id}/pool/status`) ainsi que la liste des changes, pour refléter à jour le bouton d'en-tête, le panneau d'état, et les badges `worker_active` sur les cartes
