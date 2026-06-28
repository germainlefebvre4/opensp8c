## MODIFIED Requirements

### Requirement: Rafraîchissement automatique du Kanban
Le Kanban SHALL se rafraîchir automatiquement pour refléter les changements apportés aux fichiers OpenSpec par des outils externes (Claude Code, openspec CLI). Le rafraîchissement SHALL se faire via les événements SSE du stream `/api/workspaces/{id}/events` — sans polling périodique. À réception d'un événement `change_updated`, le frontend SHALL invalider la liste des changes ET le détail du change concerné. À réception d'un événement `change_created` ou `change_deleted`, le frontend SHALL invalider uniquement la liste des changes. En cas d'indisponibilité du stream SSE, les données affichées restent celles du dernier fetch réussi (pas de fallback polling).

#### Scenario: Mise à jour d'un change via Claude Code
- **WHEN** Claude Code modifie `tasks.md` d'un change et que l'événement `change_updated` est reçu via SSE
- **THEN** le frontend recharge la liste des changes et le détail du change concerné, et le kanban se met à jour immédiatement sans action utilisateur

#### Scenario: Création d'un nouveau change via la CLI
- **WHEN** la CLI crée un nouveau répertoire de change et que l'événement `change_created` est reçu via SSE
- **THEN** le frontend recharge la liste des changes et la nouvelle carte apparaît dans la colonne appropriée

#### Scenario: Archivage d'un change via la CLI
- **WHEN** la CLI archive un change (déplace le répertoire) et que l'événement `change_deleted` est reçu via SSE
- **THEN** le frontend recharge la liste des changes et la carte disparaît des colonnes actives

#### Scenario: Déconnexion SSE — données conservées
- **WHEN** le stream SSE est interrompu (réseau, redémarrage serveur)
- **THEN** les données du kanban restent affichées dans leur dernier état connu, sans message d'erreur intrusif
