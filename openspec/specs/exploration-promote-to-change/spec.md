## Purpose

Gérer la promotion d'un ghost card vers un change réel : dialog de confirmation, endpoint `/promote`, déclenchement FF (session active ou avec contexte injecté), et transition visuelle ghost→change.
## Requirements
### Requirement: Dialog de confirmation avant promotion
Quand l'utilisateur déclenche la promotion d'un ghost card vers la colonne "ready" (par drag), le frontend SHALL afficher une dialog de confirmation avant de lancer FF.

#### Scenario: Dialog affichée au drag ghost card vers "todo"
- **WHEN** l'utilisateur dépose un ghost card (nommé) sur la colonne "ready"
- **THEN** une dialog s'affiche avec le texte "Créer un change à partir de cette exploration ?" et deux boutons : [Annuler] et [Créer le change]

#### Scenario: Annulation — ghost card reste en "to-explore"
- **WHEN** l'utilisateur clique sur [Annuler] dans la dialog de confirmation
- **THEN** la dialog se ferme, le ghost card reste dans la colonne "to-explore" sans modification, aucun appel API n'est effectué

#### Scenario: Confirmation — promotion lancée
- **WHEN** l'utilisateur clique sur [Créer le change] dans la dialog de confirmation
- **THEN** la dialog se ferme, le frontend envoie une requête `POST /api/workspaces/{id}/explorations/{ghostId}/promote` avec le contexte localStorage dans le body


### Requirement: Promotion via FF dans une nouvelle session indépendante
L'endpoint `/promote` SHALL déclencher FF dans un nouveau subprocess, indépendant de la session d'exploration (qu'elle soit encore vivante ou expirée), en lui injectant le contexte conversationnel reçu dans le body. Le change créé SHALL démarrer avec sa propre session ; l'identifiant de session Claude de l'exploration ne SHALL PAS lui être transmis. Si un fichier de brouillon `drafts/<ghostId>.json` existe pour cette exploration, le backend SHALL lire son contenu et l'injecter au subprocess pour que le change créé contienne les tâches du brouillon. Le change créé SHALL avoir `launched: false` dans son `.openspec.yaml`, comme tout change produit par un Fast-Forward vers `Ready` (voir `kanban-ready-column`). Sur succès de la promotion, le fichier de brouillon de tâche et le ghost record SHALL être conservés pour permettre la coexistence et l'affinage ultérieur, et la session de l'exploration reste utilisable indépendamment.

#### Scenario: Session exploration encore active — FF dans un subprocess distinct
- **WHEN** `POST /promote` est reçu ET la session du ghost card est encore vivante dans `session.Manager`
- **THEN** le backend démarre un nouveau subprocess pour FF (la session d'exploration n'est ni utilisée ni interrompue), émet un event SSE `ff_started` avec le nom du ghost card, et la session d'exploration reste active

#### Scenario: Session exploration expirée — FF avec contexte injecté
- **WHEN** `POST /promote` est reçu ET la session du ghost card a expiré ET le body contient le contexte conversationnel
- **THEN** le backend démarre un nouveau subprocess avec le contexte injecté comme premier message système (incluant les tâches de brouillon éventuelles), puis envoie `/opsx:ff`

#### Scenario: Aucune transmission du claudeSessionId
- **WHEN** la promotion aboutit à la création du change
- **THEN** le subprocess de FF n'est pas lancé avec le `claudeSessionId` de l'exploration et le change n'hérite d'aucune entrée de session de celle-ci

#### Scenario: FF produit le change_created marker — change créé dans "todo"
- **WHEN** le subprocess FF produit une ligne contenant `{"event":"change_created","name":"<name>"}` sur stdout
- **THEN** le backend crée le dossier `openspec/changes/<name>/` (via `openspec new change`), écrit `launched: false` dans son `.openspec.yaml`, et émet `ff_done` via SSE, le ghost record et le fichier de brouillon restants intacts et actifs dans l'application

#### Scenario: FF échoue — ghost card reste en "to-explore"
- **WHEN** le subprocess FF se termine avec une erreur
- **THEN** le backend émet `ff_failed` via SSE avec le ghostId, le ghost card reste en "to-explore", et le fichier de brouillon `drafts/<ghostId>.json` est conservé pour permettre une nouvelle tentative

### Requirement: Transition visuelle ghost card → change réel
Pendant que FF est en cours, le ghost card SHALL afficher un indicateur de progression. Quand FF se termine, la carte d'exploration reste dans la colonne "to-explore" et un nouveau change en statut "brouillon/unsolidified" apparaît dans la colonne "ready".

#### Scenario: Ghost card en cours de promotion affiche un spinner
- **WHEN** le frontend reçoit l'event SSE `ff_started` pour un ghostId
- **THEN** le ghost card affiche un spinner ou indicateur de progression, le drag est désactivé

#### Scenario: Ghost card reste et change brouillon apparaît après ff_done
- **WHEN** le frontend reçoit l'event SSE `ff_done`
- **THEN** le ghost card reste visible dans "to-explore" pour continuer l'exploration, et un nouveau change apparaît dans "ready" avec un traitement visuel "brouillon" (bordure pointillée, opacité)


### Requirement: Déplacement des logs de l'exploration vers le change créé
Quand la promotion d'un ghost aboutit à la création d'un change réel, le backend SHALL copier les logs de chat de l'exploration vers le dossier de logs du change créé, avant d'émettre `ff_done`, permettant ainsi aux deux d'avoir accès à l'historique de discussion.

#### Scenario: Logs copiés avant ff_done
- **WHEN** le subprocess FF se termine sans erreur (`proc.Wait()` ne retourne pas d'erreur)
- **THEN** le backend copie le contenu de `conversations/<workspaceId>/_explore/<ghostId>/` vers `conversations/<workspaceId>/<name>/` avant de broadcaster `ff_done`

**Note d'implémentation** : `runPromoteFF` ne parse pas le marker `change_created` sur le stdout du subprocess FF (il est actuellement consommé sans être analysé) — le succès est déterminé uniquement par le code de sortie du subprocess. Le nom du change créé est supposé égal à `ghostName` (le nom du ghost au moment de la promotion).

#### Scenario: FF échoue — logs de l'exploration conservés en place
- **WHEN** le subprocess FF échoue et le ghost card reste en "to-explore"
- **THEN** les logs de l'exploration restent uniquement sous `conversations/<workspaceId>/_explore/<ghostId>/`, aucune copie n'est effectuée


### Requirement: Solidification du change brouillon
La solidification (ou "figer") d'un change brouillon par l'utilisateur (soit explicitement, soit par action implicite telle que la modification d'une tâche ou le passage en "In Progress") SHALL détruire définitivement le ghost d'exploration associé et le fichier de brouillon pour finaliser le change.

#### Scenario: Clic sur "Figer" dans la carte ou le DetailPanel
- **WHEN** l'utilisateur clique sur le bouton "Figer" du change brouillon dans la colonne "ready" ou "todo", ou son DetailPanel
- **THEN** le frontend appelle `DELETE /api/workspaces/{id}/explorations/{ghostId}`, le backend supprime le ghost de `preferences.json`, détruit le fichier `drafts/<ghostId>.json`, et émet l'event SSE `exploration_deleted` pour faire disparaître le ghost card de "to-explore"

#### Scenario: Modification implicite d'une tâche fige le change
- **WHEN** l'utilisateur coche ou modifie une tâche d'un change brouillon dans le DetailPanel
- **THEN** le frontend déclenche silencieusement la suppression du ghost associé pour nettoyer l'espace d'exploration, rendant le change solide de manière transparente

#### Scenario: Passage à l'état In Progress fige le change
- **WHEN** l'utilisateur drag-and-drop le change brouillon de la colonne "todo" vers "in-progress"
- **THEN** le frontend déclenche silencieusement la suppression du ghost associé avant de déplacer la carte, consolidant le change


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
- **THEN** le dialogue et le volet d'exploration se ferment, l'état maximisé du volet est réinitialisé à false, la carte correspondante passe en état "FF running", et l'appel API `POST /api/workspaces/{id}/explorations/{ghostId}/promote` est envoyé


### Requirement: Réinitialisation de l'état maximisé à l'abandon d'une exploration
L'action d'abandonner/supprimer une exploration fantôme SHALL réinitialiser l'état maximisé du volet d'exploration à false pour garantir que le tableau Kanban redevienne visible.

#### Scenario: Abandon d'exploration réinitialise l'état maximisé
- **WHEN** l'utilisateur supprime une exploration (via le bouton de suppression de la carte ou du volet) et confirme l'abandon
- **THEN** le volet d'exploration se ferme, la carte d'exploration est supprimée, et l'état maximisé du volet est réinitialisé à false
