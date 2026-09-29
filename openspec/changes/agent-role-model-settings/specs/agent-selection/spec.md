# Spec Delta

## ADDED Requirements

### Requirement: Superposition des variables d'environnement du workspace
Lors du démarrage de tout subprocess d'agent CLI pour un workspace, le backend SHALL superposer aux variables décrites par « Injection dynamique de variables d'environnement au démarrage des agents » les variables surchargées par ce workspace, dans l'ordre : variables globales, variables de l'agent, variables globales du workspace, variables de l'agent pour le workspace ; la dernière valeur définie l'emporte. Les variables d'un workspace SHALL n'affecter que les subprocess lancés pour ce workspace.

#### Scenario: Variable du workspace prioritaire
- **WHEN** `env` définit `FOO=global` et que la section du workspace `A` définit `FOO=workspace`
- **THEN** un subprocess lancé pour `A` reçoit `FOO=workspace`

#### Scenario: Autre workspace non affecté
- **WHEN** la section du workspace `A` définit `FOO=workspace` et qu'un subprocess est lancé pour le workspace `B`
- **THEN** le subprocess de `B` ne reçoit pas `FOO=workspace`

### Requirement: Le verrou de session reste prioritaire sur les réglages de rôle
Le verrouillage de l'agent d'une conversation à sa création SHALL prévaloir sur l'agent défini par un réglage de rôle ou par le réglage global des rôles. Pour une session verrouillée, seuls le modèle et l'effort sont résolus par la cascade de `agent-role-settings`, pour l'agent verrouillé.

#### Scenario: Session verrouillée et réglage de rôle différent
- **WHEN** une session nommée est verrouillée sur `claude` et que le rôle `explorer` est ensuite réglé sur `codex`
- **THEN** la session existante continue avec `claude`

#### Scenario: Agent d'une nouvelle session issu du rôle
- **WHEN** une nouvelle session d'exploration est créée et que le rôle `explorer` définit l'agent `gemini`
- **THEN** l'agent de la session est `gemini` et est verrouillé pour sa durée
