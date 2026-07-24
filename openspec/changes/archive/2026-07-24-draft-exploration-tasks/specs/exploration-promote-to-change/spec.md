## MODIFIED Requirements

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
