## ADDED Requirements

### Requirement: Déplacement des logs de l'exploration vers le change créé

Quand la promotion d'un ghost aboutit à la création d'un change réel, le backend SHALL déplacer les logs de chat de l'exploration vers le dossier de logs du change créé, avant d'émettre `ff_done`.

#### Scenario: Logs déplacés avant ff_done
- **WHEN** le subprocess FF se termine sans erreur (`proc.Wait()` ne retourne pas d'erreur)
- **THEN** le backend déplace `conversations/<workspaceId>/_explore/<ghostId>/` vers `conversations/<workspaceId>/<name>/` avant de broadcaster `ff_done`

**Note d'implémentation** : `runPromoteFF` ne parse pas le marker `change_created` sur le stdout du subprocess FF (il est actuellement consommé sans être analysé) — le succès est déterminé uniquement par le code de sortie du subprocess. Le nom du change créé est supposé égal à `ghostName` (le nom du ghost au moment de la promotion).

#### Scenario: FF échoue — logs de l'exploration conservés en place
- **WHEN** le subprocess FF échoue et le ghost card reste en "to-explore"
- **THEN** les logs de l'exploration restent sous `conversations/<workspaceId>/_explore/<ghostId>/`, aucun déplacement n'est effectué
