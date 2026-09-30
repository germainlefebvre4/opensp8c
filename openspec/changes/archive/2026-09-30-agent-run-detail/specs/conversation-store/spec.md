# Spec Delta

## ADDED Requirements

### Requirement: Runs de kind pool
Le `ConversationStore` SHALL accepter le kind `pool` en plus de `chat` et `ff`, et persister les runs des workers du pool à l'emplacement `conversations/<workspaceID>/<changeName>/pool/<timestamp>.jsonl`. Il SHALL pouvoir lister les runs d'un kind pour l'ensemble des changes d'un workspace, en ignorant les dossiers d'exploration anonyme (`_explore`). Les lignes de marqueur de début et de fin d'un run pool SHALL être ignorées par les consommateurs qui dérivent des entrées d'activité d'un run.

#### Scenario: Ouverture d'un run pool
- **WHEN** un worker du pool démarre le subprocess d'un agent pour le change `add-user-auth`
- **THEN** le fichier du run est créé sous `conversations/<wsID>/add-user-auth/pool/`

#### Scenario: Liste des runs pool d'un workspace
- **WHEN** plusieurs changes du workspace ont des runs de kind `pool`
- **THEN** le store retourne les runs de tous ces changes, sans inclure les explorations anonymes

#### Scenario: Runs pool présents dans l'onglet Conversation d'un change
- **WHEN** un change a un run pool et que le client demande son flux d'activité fusionné
- **THEN** la narration et les appels d'outils de ce run apparaissent dans le flux, et les marqueurs de début et de fin n'y produisent aucune entrée
