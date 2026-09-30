# Spec Delta

## MODIFIED Requirements

### Requirement: Configuration du pool d'agents par workspace
Le backend SHALL exposer un mécanisme de configuration du pool d'agents propre à chaque workspace. Cette configuration comprend le nombre maximum d'agents parallèles (`size`, de 1 à 5), le mode de délégation (`delegation_mode`, `full-autonomy` ou `hitl-review`) et le nombre maximal de tentatives d'auto-correction (`max_attempts`), appliqués uniquement aux workers lancés pour ce workspace. Cette configuration SHALL être persistée : des défauts globaux sont définis dans Configuration et chaque workspace peut les surcharger champ par champ dans Settings. Au démarrage d'un pool, la configuration effective SHALL être résolue en prenant, pour chaque champ, la valeur du workspace, sinon le défaut global, sinon la valeur par défaut de la plateforme ; une configuration passée explicitement dans la requête de démarrage SHALL l'emporter sur cette résolution pour ce lancement.

#### Scenario: Configuration valide du pool d'un workspace
- **WHEN** le client demande à configurer le pool du workspace `A` avec une taille de 3 et le mode `hitl-review`
- **THEN** le backend stocke et applique cette configuration pour tous les futurs workers lancés par le pool de ce workspace, sans affecter la configuration ou l'exécution du pool d'un autre workspace

#### Scenario: Résolution sans requête explicite
- **WHEN** un pool est démarré pour le workspace `A` sans configuration explicite, que le défaut global fixe la taille à 2 et que `A` surcharge le mode à `full-autonomy`
- **THEN** le pool de `A` démarre avec une taille de 2 et le mode `full-autonomy`

#### Scenario: Configuration explicite dans la requête
- **WHEN** un pool est démarré avec une taille de 5 dans la requête alors que la configuration résolue donnerait 2
- **THEN** le pool démarre avec une taille de 5 pour ce lancement sans modifier les valeurs persistées

## ADDED Requirements

### Requirement: Application des réglages de rôle aux workers
Le pool SHALL lancer chaque worker avec l'agent, le modèle et l'effort résolus pour le rôle `implementer` du workspace, et SHALL utiliser le rôle `fixer` pour le subprocess démarré lors d'une relance après « Demander des corrections ». Les tours d'auto-guérison d'un worker partageant son subprocess d'implémentation, ils SHALL conserver les réglages de l'`implementer`.

#### Scenario: Worker avec réglages de rôle
- **WHEN** le rôle `implementer` du workspace `A` définit l'agent `claude`, le modèle `sonnet` et l'effort `medium`
- **THEN** le subprocess d'un worker de `A` est lancé avec ces valeurs

#### Scenario: Relance après demande de corrections
- **WHEN** un worker est relancé après « Demander des corrections »
- **THEN** le nouveau subprocess utilise les réglages du rôle `fixer`

#### Scenario: Tours d'auto-guérison
- **WHEN** la validation échoue et que le worker réinjecte l'erreur dans son subprocess
- **THEN** aucun nouveau subprocess n'est démarré et le modèle reste celui de l'`implementer`
