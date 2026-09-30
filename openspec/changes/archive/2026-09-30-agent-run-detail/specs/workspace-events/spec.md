# Spec Delta

## ADDED Requirements

### Requirement: Événement pool_run_appended
Le stream SSE d'un workspace SHALL émettre un événement `pool_run_appended` portant le nom du change (`event: pool_run_appended\ndata: {"name":"<change-name>"}\n\n`) lorsque le run pool de ce change reçoit de nouvelles lignes ou change d'issue. Le backend SHALL limiter cet événement à au plus un par seconde et par change pendant une rafale de lignes, tout en garantissant qu'un événement est émis après la dernière ligne d'une rafale et à la fin du run.

#### Scenario: Rafale de lignes de sortie
- **WHEN** l'agent produit 200 lignes de sortie en une seconde
- **THEN** le client reçoit au plus un événement `pool_run_appended` pour ce change pendant cette seconde, puis un événement supplémentaire après la dernière ligne

#### Scenario: Fin de run
- **WHEN** le worker se termine
- **THEN** un événement `pool_run_appended` est émis pour le change afin que les clients relisent l'issue finale du run
