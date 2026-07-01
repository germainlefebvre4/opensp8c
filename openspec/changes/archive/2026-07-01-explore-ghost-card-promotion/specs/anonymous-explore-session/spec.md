## MODIFIED Requirements

### Requirement: Promotion de session anonyme vers session nommée
La promotion automatique via `change_created` est remplacée par un mécanisme en deux temps : nommage du ghost card via `ghost_named` sur premier message, puis promotion explicite vers un change réel uniquement quand l'utilisateur déclenche FF.

#### Scenario: LLM émet ghost_named — ghost card renommé, session reste anonyme
- **WHEN** le subprocess de la session anonyme produit une ligne contenant `{"event":"ghost_named","name":"<name>"}` sur stdout
- **THEN** le backend met à jour le ghost record dans `preferences.json` avec le nouveau nom, émet un event SSE `ghost_named`, et la session reste une session anonyme — elle n'est PAS rekeyed vers une session nommée à ce stade

#### Scenario: change_created ignoré en mode exploration anonyme
- **WHEN** le subprocess d'une session anonyme produit une ligne contenant `{"event":"change_created","name":"<name>"}` sur stdout
- **THEN** le backend ignore cet event — aucune promotion automatique, aucun change créé ; le ghost card reste inchangé dans "to-explore"

#### Scenario: Promotion vers change réel uniquement via /promote
- **WHEN** l'endpoint `POST /api/workspaces/{id}/explorations/{ghostId}/promote` est appelé
- **THEN** FF est déclenché (session existante ou contexte injecté) et le change est créé — c'est le seul chemin vers la création d'un change depuis une exploration anonyme

## REMOVED Requirements

### Requirement: Promotion de session anonyme vers session nommée (ancienne mécanique)
**Reason**: Remplacé par le flux ghost card — la promotion automatique (LLM décide) supprime le contrôle utilisateur. Le nouveau flux `ghost_named` + promote explicite reprend ce rôle avec contrôle humain.
**Migration**: Le marker `change_created` n'est plus détecté dans les sessions anonymes. Le `anonSystemPrompt` est mis à jour pour utiliser `ghost_named` à la place. Les sessions anonymes existantes (déjà promues avant cette migration) ne sont pas affectées.
