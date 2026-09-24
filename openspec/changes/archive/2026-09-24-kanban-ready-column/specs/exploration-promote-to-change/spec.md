# Spec Delta

## MODIFIED Requirements

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

### Requirement: Promotion via FF dans la session existante ou avec contexte injecté
L'endpoint `/promote` SHALL déclencher FF en réutilisant la session existante si elle est active, ou en démarrant un nouveau subprocess avec le contexte conversationnel injecté si la session a expiré. Si un fichier de brouillon `drafts/<ghostId>.json` existe pour cette exploration, le backend SHALL lire son contenu et l'associer au contexte ou l'injecter au subprocess pour que le change créé contienne les tâches du brouillon. Le change créé SHALL avoir `launched: false` dans son `.openspec.yaml`, comme tout change produit par un Fast-Forward vers `Ready` (voir `kanban-ready-column`). Sur succès de la promotion, le fichier de brouillon de tâche et le ghost record SHALL être conservés pour permettre la coexistence et l'affinage ultérieur.

#### Scenario: Session exploration encore active — FF dans la même session
- **WHEN** `POST /promote` est reçu ET la session du ghost card est encore vivante dans `session.Manager`
- **THEN** le backend écrit `/opsx:ff\n` sur stdin du subprocess existant, émet un event SSE `ff_started` avec le nom du ghost card, et surveille le stream pour `change_created`

#### Scenario: Session exploration expirée — FF avec contexte injecté
- **WHEN** `POST /promote` est reçu ET la session du ghost card a expiré ET le body contient le contexte conversationnel
- **THEN** le backend démarre un nouveau subprocess avec le contexte injecté comme premier message système (incluant les tâches de brouillon éventuelles), puis envoie `/opsx:ff`, et surveille le stream pour `change_created`

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
