# Spec Delta

## MODIFIED Requirements

### Requirement: Rôles d'exécution des agents
Le système SHALL distinguer six rôles d'exécution pour les subprocess d'agents CLI qu'il lance : `explorer` (sessions d'exploration nommées et anonymes), `ff` (exécutions fast-forward qui génèrent les artefacts d'un change), `implementer` (workers du pool qui implémentent les tâches d'un change), `fixer` (relancement d'un worker après « Demander des corrections » en revue HITL), `verifier` (vérification automatique d'un change, conformité et UI) et `documenter` (génération de la documentation). Chaque subprocess lancé SHALL être associé à exactement un de ces rôles.

#### Scenario: Rôle d'une session d'exploration
- **WHEN** une session d'exploration nommée ou anonyme est démarrée
- **THEN** ses réglages sont ceux du rôle `explorer`

#### Scenario: Rôle d'un fast-forward
- **WHEN** un fast-forward est déclenché pour un change
- **THEN** ses réglages sont ceux du rôle `ff`

#### Scenario: Rôle d'un worker du pool
- **WHEN** un worker du pool démarre l'implémentation d'un change de la colonne To Do
- **THEN** ses réglages sont ceux du rôle `implementer`

#### Scenario: Rôle d'une génération de documentation
- **WHEN** la génération de documentation d'un workspace est lancée
- **THEN** ses réglages sont ceux du rôle `documenter`

#### Scenario: Résolution du rôle de vérification
- **WHEN** aucun champ du rôle `verifier` n'est défini et que le réglage global définit l'agent `claude`, le modèle `opus` et l'effort `high`
- **THEN** un lancement résolu avec le rôle `verifier` utilise `claude`, `opus` et `high`

### Requirement: Préréglages par défaut des rôles pour l'agent Claude
Lorsque l'agent résolu est Claude et qu'aucun niveau ne définit un champ, le système SHALL appliquer un préréglage propre à chaque rôle : `explorer` avec un modèle de raisonnement (`opus`) et l'effort `high` ; `ff`, `implementer`, `fixer` et `verifier` avec un modèle de code (`sonnet`) et l'effort `medium` ; `documenter` avec un modèle économique (`haiku`) et l'effort `low`. Pour les autres agents, il n'y a pas de préréglage.

#### Scenario: Préréglage de l'explorateur avec Claude
- **WHEN** aucun réglage n'est enregistré et qu'une exploration est lancée avec l'agent `claude`
- **THEN** le subprocess reçoit le modèle `opus` et l'effort `high`

#### Scenario: Préréglage du documenteur avec Claude
- **WHEN** aucun réglage n'est enregistré et que la documentation est générée avec l'agent `claude`
- **THEN** le subprocess reçoit le modèle `haiku` et l'effort `low`

#### Scenario: Préréglage du vérificateur avec Claude
- **WHEN** aucun réglage n'est enregistré et qu'un lancement résolu avec le rôle `verifier` utilise l'agent `claude`
- **THEN** le modèle résolu est `sonnet` et l'effort `medium`

#### Scenario: Pas de préréglage pour un autre agent
- **WHEN** aucun réglage n'est enregistré et qu'un worker est lancé avec l'agent `gemini`
- **THEN** aucun flag de modèle ni d'effort n'est passé

### Requirement: Validation des réglages
Le système SHALL rejeter avec une erreur de validation, sans modifier les réglages enregistrés, toute mise à jour dont l'agent n'appartient pas aux agents supportés, dont l'effort n'appartient pas aux niveaux déclarés par l'agent effectivement défini au même niveau, ou dont le modèle est vide après suppression des espaces, commence par un tiret, contient un espace ou un caractère de contrôle, ou dépasse une longueur maximale raisonnable. Le rôle ciblé SHALL appartenir aux six rôles définis.

#### Scenario: Agent inconnu
- **WHEN** une mise à jour fixe l'agent d'un rôle à une valeur absente des agents supportés
- **THEN** la requête est rejetée avec une erreur de validation et aucun réglage n'est modifié

#### Scenario: Modèle assimilable à un flag
- **WHEN** une mise à jour fixe un modèle à `--dangerously-skip-permissions`
- **THEN** la requête est rejetée avec une erreur de validation

#### Scenario: Rôle inconnu
- **WHEN** une mise à jour cible un rôle autre que les six rôles définis
- **THEN** la requête est rejetée avec une erreur de validation

#### Scenario: Rôle de vérification accepté
- **WHEN** une mise à jour fixe le modèle `haiku` pour le rôle `verifier`
- **THEN** la mise à jour est acceptée et enregistrée

#### Scenario: Effort hors niveaux de l'agent
- **WHEN** une mise à jour fixe l'agent `antigravity` et l'effort `xhigh`
- **THEN** la requête est rejetée avec une erreur de validation
