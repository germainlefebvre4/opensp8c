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
L'endpoint `/promote` SHALL déclencher FF en réutilisant la session existante si elle est active, ou en démarrant un nouveau subprocess avec le contexte conversationnel injecté si la session a expiré. Si un fichier de brouillon `drafts/<ghostId>.json` existe pour cette exploration, le backend SHALL lire son contenu et l'associer au contexte ou l'injecter au subprocess pour que le change créé contienne les tâches du brouillon. Sur succès de la promotion, le fichier de brouillon de tâche SHALL être supprimé.

#### Scenario: Session exploration encore active — FF dans la même session
- **WHEN** `POST /promote` est reçu ET la session du ghost card est encore vivante dans `session.Manager`
- **THEN** le backend écrit `/opsx:ff\n` sur stdin du subprocess existant, émet un event SSE `ff_started` avec le nom du ghost card, et surveille le stream pour `change_created`

#### Scenario: Session exploration expirée — FF avec contexte injecté
- **WHEN** `POST /promote` est reçu ET la session du ghost card a expiré ET le body contient le contexte conversationnel
- **THEN** le backend démarre un nouveau subprocess avec le contexte injecté comme premier message système (incluant les tâches de brouillon éventuelles), puis envoie `/opsx:ff`, et surveille le stream pour `change_created`

#### Scenario: FF produit le change_created marker — change créé dans "todo"
- **WHEN** le subprocess FF produit une ligne contenant `{"event":"change_created","name":"<name>"}` sur stdout
- **THEN** le backend crée le dossier `openspec/changes/<name>/` (via `openspec new change`), émet `ff_done` via SSE, le ghost record est supprimé de `preferences.json`, et le fichier de brouillon `drafts/<ghostId>.json` est supprimé du disque

#### Scenario: FF échoue — ghost card reste en "to-explore"
- **WHEN** le subprocess FF se termine avec une erreur
- **THEN** le backend émet `ff_failed` via SSE avec le ghostId, le ghost card reste en "to-explore", et le fichier de brouillon `drafts/<ghostId>.json` est conservé pour permettre une nouvelle tentative

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

### Requirement: Bouton de promotion dans le header d'exploration anonyme
Le frontend SHALL afficher un bouton d'action discret "Créer le change" dans le header du volet d'exploration anonyme (`ExploreAnonymousPanel`) dès que l'exploration a démarré (présence de `ghostId` et `ghostName`).

#### Scenario: Bouton affiché dès le démarrage de l'exploration anonyme
- **WHEN** le volet d'exploration anonyme est ouvert et dispose de `ghostId` et `ghostName`
- **THEN** un bouton d'action "Créer le change" s'affiche dans le header à côté des actions système (agrandir/fermer)

#### Scenario: Bouton absent si l'exploration n'est pas initialisée ou déjà associée à un change
- **WHEN** le volet d'exploration s'ouvre pour une exploration déjà associée à un change normal (non-ghost)
- **THEN** le bouton de promotion n'est pas affiché dans le header

### Requirement: Comportement responsive du bouton de promotion
Le bouton de promotion dans le header SHALL adapter sa présentation selon la largeur du volet d'exploration.

#### Scenario: Largeur suffisante — texte et icône
- **WHEN** le volet d'exploration a une largeur supérieure ou égale à 350px
- **THEN** le bouton de promotion affiche une icône d'action (`✨`) suivie du texte descriptif (ex: "Créer le change")

#### Scenario: Largeur étroite — icône seule avec tooltip
- **WHEN** le volet d'exploration a une largeur inférieure à 350px
- **THEN** le bouton de promotion n'affiche que l'icône `✨` et affiche un tooltip descriptif lors du survol (ex: "Créer le change à partir de cette exploration")

### Requirement: Dialogue de confirmation de promotion depuis le volet
Le clic sur le bouton de promotion du volet d'exploration SHALL ouvrir la même boîte de dialogue de confirmation que le drag-and-drop, permettant de modifier le nom et de confirmer ou annuler la promotion.

#### Scenario: Validation de la promotion depuis le dialogue
- **WHEN** l'utilisateur clique sur le bouton de promotion, confirme/modifie le nom du change dans le dialogue, et clique sur [Créer le change]
- **THEN** le dialogue et le volet d'exploration se ferment, la carte correspondante passe en état "FF running", et l'appel API `POST /api/workspaces/{id}/explorations/{ghostId}/promote` est envoyé

