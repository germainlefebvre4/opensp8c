# Spec Delta

## MODIFIED Requirements

### Requirement: Rétrogradation manuelle To Do → Ready

Le backend SHALL exposer une action symétrique permettant de remarquer un change `launched: false`, le retirant de l'éligibilité au ramassage par l'Agent Pool. Cette action SHALL supporter un paramètre optionnel `force=true`. Si un worker de l'Agent Pool est actuellement actif sur ce change : sans `force=true`, l'action SHALL être refusée avec une erreur 409 Conflict ; avec `force=true`, le backend SHALL interrompre le worker actif (`cancel()`), conserver le code et les commits dans le git worktree (ou réinitialiser les modifications non validées si l'exécution n'est pas isolée dans un worktree), puis ATTENDRE la fin réelle du worker, dans un délai maximum, avant de répondre. Selon l'issue du worker : si le worker s'est arrêté sans fusionner (`stopped`), le backend SHALL écrire `launched: false` dans le `.openspec.yaml` et renvoyer un statut 204 ; si le worker a fusionné le change avant d'honorer l'annulation (`completed`), le backend SHALL refuser la rétrogradation, laisser `launched` inchangé et renvoyer une erreur 409 Conflict indiquant que le change est déjà fusionné dans la branche cible ; si le délai est dépassé alors que le worker est toujours en cours, le backend SHALL ne pas rétrograder et renvoyer une erreur explicite invitant à réessayer. Dans l'interface Kanban, glisser une carte ayant un worker actif de `To Do` vers `Ready` SHALL ouvrir un dialogue de confirmation demandant à l'utilisateur de valider l'interruption du worker avant d'appeler l'endpoint avec `force=true`, et l'interface SHALL afficher un message distinct lorsque la rétrogradation est refusée parce que le change est déjà fusionné.

#### Scenario: Rétrogradation réussie
- **WHEN** l'utilisateur déclenche la rétrogradation d'un change en colonne `To Do` sans worker actif (drag vers `Ready` ou bouton dédié)
- **THEN** le backend écrit `launched: false` dans le `.openspec.yaml` du change, le watcher SSE émet `change_updated`, et la carte apparaît en colonne `Ready`

#### Scenario: Rétrogradation refusée pendant exécution
- **WHEN** l'action de rétrogradation est appelée sans `force=true` pour un change dont `worker_active` vaut `true`
- **THEN** le backend refuse l'action avec une erreur 409 Conflict, et le change reste `launched: true`

#### Scenario: Rétrogradation forcée avec worker actif
- **WHEN** l'action de rétrogradation est appelée avec `force=true` pour un change dont un worker est actif et que ce worker s'arrête sans avoir fusionné
- **THEN** le backend interrompt le subprocess de l'agent associé, attend la fin du worker, retire le worker de la liste active, écrit `launched: false` dans `.openspec.yaml`, répond 204 et émet les événements SSE `pool_updated` et `change_updated`

#### Scenario: Rétrogradation forcée alors que le change est déjà fusionné
- **WHEN** l'action de rétrogradation est appelée avec `force=true` et que le worker a déjà démarré la fusion du change avant d'honorer l'annulation, et que cette fusion aboutit
- **THEN** le backend n'écrit pas `launched: false`, le change reste `launched: true`, et la réponse est une erreur 409 Conflict indiquant que le change est déjà fusionné dans la branche cible

#### Scenario: Rétrogradation forcée dont l'attente expire
- **WHEN** l'action de rétrogradation est appelée avec `force=true` et que le worker n'a pas terminé dans le délai d'attente maximum
- **THEN** le backend n'écrit pas `launched: false` et renvoie une erreur explicite indiquant que le worker est toujours en cours et qu'il faut réessayer

#### Scenario: Rétrogradation conserve le code du git worktree
- **WHEN** un worker actif est interrompu via rétrogradation forcée et qu'il opérait dans un git worktree dédié (`~/.opensp8c/worktrees/wt-<name>`)
- **THEN** le dossier worktree et la branche `feature/<name>` avec ses modifications et commits restent intacts sur le disque

#### Scenario: Confirmation UI lors du drag d'une carte avec worker actif
- **WHEN** l'utilisateur glisse-dépose une carte ayant `worker_active = true` de `To Do` vers `Ready`
- **THEN** l'interface ouvre une modale de confirmation d'interruption du worker ; la confirmation déclenche l'appel avec `force=true` et l'annulation laisse la carte en `To Do`

#### Scenario: Message distinct pour un change déjà fusionné
- **WHEN** la rétrogradation forcée est refusée parce que le change est déjà fusionné
- **THEN** l'interface affiche un message indiquant que le change a été fusionné avant l'interruption, et non le message générique d'échec
