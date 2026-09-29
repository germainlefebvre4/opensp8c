# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: Promotion via FF dans la session existante ou avec contexte injecté`
- TO: `### Requirement: Promotion via FF dans une nouvelle session indépendante`

## MODIFIED Requirements

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
