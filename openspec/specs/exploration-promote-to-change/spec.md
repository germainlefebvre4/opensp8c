## Purpose

Gérer la promotion d'un ghost card vers un change réel : dialog de confirmation, endpoint `/promote`, déclenchement FF (session active ou avec contexte injecté), et transition visuelle ghost→change.
## Requirements
### Requirement: Dialog de confirmation avant promotion
Quand l'utilisateur déclenche la promotion d'un ghost card vers la colonne "todo" (par drag), le frontend SHALL afficher une dialog de confirmation avant de lancer FF.

#### Scenario: Dialog affichée au drag ghost card vers "todo"
- **WHEN** l'utilisateur dépose un ghost card (nommé) sur la colonne "todo"
- **THEN** une dialog s'affiche avec le texte "Créer un change à partir de cette exploration ?" et deux boutons : [Annuler] et [Créer le change]

#### Scenario: Annulation — ghost card reste en "to-explore"
- **WHEN** l'utilisateur clique sur [Annuler] dans la dialog de confirmation
- **THEN** la dialog se ferme, le ghost card reste dans la colonne "to-explore" sans modification, aucun appel API n'est effectué

#### Scenario: Confirmation — promotion lancée
- **WHEN** l'utilisateur clique sur [Créer le change] dans la dialog de confirmation
- **THEN** la dialog se ferme, le frontend envoie une requête `POST /api/workspaces/{id}/explorations/{ghostId}/promote` avec le contexte localStorage dans le body

### Requirement: Promotion via FF dans la session existante ou avec contexte injecté
L'endpoint `/promote` SHALL déclencher FF en réutilisant la session existante si elle est active, ou en démarrant un nouveau subprocess avec le contexte conversationnel injecté si la session a expiré.

#### Scenario: Session exploration encore active — FF dans la même session
- **WHEN** `POST /promote` est reçu ET la session du ghost card est encore vivante dans `session.Manager`
- **THEN** le backend écrit `/opsx:ff\n` sur stdin du subprocess existant, émet un event SSE `ff_started` avec le nom du ghost card, et surveille le stream pour `change_created`

#### Scenario: Session exploration expirée — FF avec contexte injecté
- **WHEN** `POST /promote` est reçu ET la session du ghost card a expiré ET le body contient le contexte conversationnel
- **THEN** le backend démarre un nouveau subprocess avec le contexte injecté comme premier message système, puis envoie `/opsx:ff`, et surveille le stream pour `change_created`

#### Scenario: FF produit le change_created marker — change créé dans "todo"
- **WHEN** le subprocess FF produit une ligne contenant `{"event":"change_created","name":"<name>"}` sur stdout
- **THEN** le backend crée le dossier `openspec/changes/<name>/` (via `openspec new change`), émet `ff_done` via SSE, et le ghost record est supprimé de `preferences.json`

#### Scenario: FF échoue — ghost card reste en "to-explore"
- **WHEN** le subprocess FF se termine avec une erreur
- **THEN** le backend émet `ff_failed` via SSE avec le ghostId, le ghost card reste en "to-explore", et l'utilisateur peut réessayer

### Requirement: Transition visuelle ghost card → change réel
Pendant que FF est en cours, le ghost card SHALL afficher un indicateur de progression. Quand FF se termine, la carte doit transitionner vers un change normal dans la colonne "todo".

#### Scenario: Ghost card en cours de promotion affiche un spinner
- **WHEN** le frontend reçoit l'event SSE `ff_started` pour un ghostId
- **THEN** le ghost card affiche un spinner ou indicateur de progression, le drag est désactivé

#### Scenario: Ghost card disparaît après ff_done
- **WHEN** le frontend reçoit l'event SSE `ff_done`
- **THEN** le ghost card disparaît de "to-explore" et un change normal apparaît dans "todo" avec les artefacts créés par FF

### Requirement: Déplacement des logs de l'exploration vers le change créé

Quand la promotion d'un ghost aboutit à la création d'un change réel, le backend SHALL déplacer les logs de chat de l'exploration vers le dossier de logs du change créé, avant d'émettre `ff_done`.

#### Scenario: Logs déplacés avant ff_done
- **WHEN** le subprocess FF se termine sans erreur (`proc.Wait()` ne retourne pas d'erreur)
- **THEN** le backend déplace `conversations/<workspaceId>/_explore/<ghostId>/` vers `conversations/<workspaceId>/<name>/` avant de broadcaster `ff_done`

**Note d'implémentation** : `runPromoteFF` ne parse pas le marker `change_created` sur le stdout du subprocess FF (il est actuellement consommé sans être analysé) — le succès est déterminé uniquement par le code de sortie du subprocess. Le nom du change créé est supposé égal à `ghostName` (le nom du ghost au moment de la promotion).

#### Scenario: FF échoue — logs de l'exploration conservés en place
- **WHEN** le subprocess FF échoue et le ghost card reste en "to-explore"
- **THEN** les logs de l'exploration restent sous `conversations/<workspaceId>/_explore/<ghostId>/`, aucun déplacement n'est effectué

