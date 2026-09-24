# Spec Delta

## MODIFIED Requirements

### Requirement: Boucle d'auto-correction (Self-Healing Loop) des workers
Chaque worker exécutant un changement SHALL invoquer un agent CLI réel (et non simulé) au sein d'une session non-interactive unique par tentative, chargée d'exécuter les tâches restantes du `tasks.md`. Après cette invocation, le worker SHALL exécuter la commande de validation de test ou de compilation. En cas d'échec, le worker SHALL ré-injecter les logs d'erreurs dans le contexte du modèle en relançant une nouvelle session d'agent, jusqu'à un maximum configurable de tentatives.

#### Scenario: Invocation réelle de l'agent sur un changement Todo
- **WHEN** un worker prend en charge le changement `add-user-auth`
- **THEN** le backend lance une session non-interactive de l'agent configuré pour ce workspace/changement, dans le worktree isolé du changement, avec pour instruction d'implémenter les tâches restantes de `tasks.md`

#### Scenario: Auto-correction réussie sur erreur de compilation
- **WHEN** l'agent modifie un fichier provoquant une erreur de compilation Go, et que la commande de build échoue
- **THEN** le worker extrait l'erreur de compilation, la transmet à l'agent dans un message système, et l'agent génère un correctif qui réussit la compilation au deuxième essai

#### Scenario: Échec de correction après tentatives maximales
- **WHEN** l'agent n'arrive pas à résoudre l'erreur de build après 3 tentatives consécutives
- **THEN** le worker marque l'état comme bloqué, arrête l'exécution de ce changement pour demander l'aide de l'utilisateur, et libère le worker pour d'autres tâches indépendantes

## ADDED Requirements

### Requirement: Vérification de complétion avant finalisation
Après une validation réussie (build/tests OK), le worker SHALL vérifier que `tasks.md` ne contient plus aucune tâche non cochée avant de finaliser le changement (fusion automatique en mode `full-autonomy`, ou transition vers l'état de revue en mode `hitl-review`). Si des tâches restent non cochées malgré une validation réussie, le worker SHALL NOT finaliser le changement et SHALL passer son propre statut à `paused` pour signaler qu'une intervention est nécessaire.

#### Scenario: Finalisation refusée si des tâches restent ouvertes
- **WHEN** la validation (tests/build) réussit pour le changement `add-user-auth` mais que `tasks.md` contient encore des tâches non cochées
- **THEN** le worker ne fusionne pas la branche et ne fait pas transitionner le changement vers l'état suivant ; il passe son propre statut à `paused`

#### Scenario: Finalisation autorisée quand toutes les tâches sont cochées
- **WHEN** la validation réussit et que toutes les tâches de `tasks.md` sont cochées
- **THEN** le worker finalise le changement selon le mode de délégation configuré (fusion en `full-autonomy`, transition vers revue en `hitl-review`)
