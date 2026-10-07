# Spec Delta

## MODIFIED Requirements

### Requirement: Confirmation avant reset de tasks
Un dialog de confirmation SHALL être affiché avant tout drop sur **To Explore**. Le message SHALL être adapté à l'état des tâches du changement et à la présence d'une branche : lorsque le changement porte une branche `feature/<change>` (`has_branch`), quelle que soit sa colonne d'origine et même si aucune tâche n'est cochée, le dialog SHALL avertir que le travail réalisé dans la branche, son worktree et son état de revue seront supprimés et ne pourront pas être récupérés.

#### Scenario: Reset depuis ready (aucune tâche faite, non lancé)
- **WHEN** l'utilisateur confirme le drop d'une carte **Ready** vers **To Explore**
- **THEN** le dialog indique "Réinitialiser les tâches ?" avec un message neutre

#### Scenario: Reset depuis todo (aucune tâche faite)
- **WHEN** l'utilisateur confirme le drop d'une carte **To Do** vers **To Explore**
- **THEN** le dialog indique "Réinitialiser les tâches ?" avec un message neutre

#### Scenario: Reset depuis in-progress (tâches partiellement faites)
- **WHEN** l'utilisateur confirme le drop d'une carte **In Progress** vers **To Explore**
- **THEN** le dialog indique "X tâches complétées seront perdues. Continuer ?" avec un message d'avertissement

#### Scenario: Reset d'une carte qui porte une branche
- **WHEN** l'utilisateur dépose vers **To Explore** une carte **Ready** ou **To Do** dont `has_branch` vaut `true`
- **THEN** le dialog affiche le message d'avertissement indiquant que la branche et le travail qu'elle contient seront supprimés, avec le bouton de confirmation en style d'avertissement

#### Scenario: Refus du backend
- **WHEN** le backend répond `409` à la demande de reset (worker actif ou action de revue en cours)
- **THEN** une notification affiche le message d'erreur, la carte retourne à sa colonne d'origine et rien n'est modifié

#### Scenario: Annulation de la confirmation
- **WHEN** l'utilisateur clique "Annuler" dans le dialog de confirmation
- **THEN** la carte retourne à sa colonne d'origine sans modification
