# Spec Delta

## MODIFIED Requirements

### Requirement: Reprise du change après une demande de correction
Une fois le marqueur levé par une demande de correction, le change SHALL redevenir éligible au dispatcher (statut dérivé de l'avancement et de l'état « lancé »). Le worker qui le reprend SHALL réutiliser la branche et le worktree existants, appliquer les tâches restantes, dont la correction, avec le même flux que pour toute reprise (`/opsx:apply`, triage, validation, contrôle de complétion exigeant que toutes les tâches à faire par l'agent, corrections comprises, soient cochées : les tâches de validation humaine restées décochées ne comptent pas, voir `agent-pool-orchestrator`), puis repasser le change en revue en mode `hitl-review`. Si le pool est arrêté au moment de la demande, le change SHALL être repris au prochain démarrage du pool.

#### Scenario: Worker relancé sur la correction
- **WHEN** la correction est demandée et que le pool tourne
- **THEN** le dispatcher assigne un worker au change, qui réutilise `feature/<change>` et son worktree et traite la tâche de correction

#### Scenario: Retour en revue après correction
- **WHEN** le worker termine les tâches, corrections incluses, et que la validation réussit
- **THEN** le change repasse en To Review avec un nouveau marqueur de revue

#### Scenario: Correction non appliquée
- **WHEN** le worker termine sans avoir coché la tâche de correction
- **THEN** il ne finalise pas le changement et passe à `paused` (contrôle de complétion existant)

#### Scenario: Pool arrêté à la demande de correction
- **WHEN** la correction est demandée alors que le pool est arrêté
- **THEN** le change n'est plus en To Review et sera repris par un worker au prochain démarrage du pool

#### Scenario: Tâches de validation humaine conservées après correction
- **WHEN** une correction est demandée pour un change en revue dont des tâches de validation humaine sont décochées, et que le worker applique la correction
- **THEN** ces tâches restent décochées, le contrôle de complétion les ignore et le change repasse en To Review

## ADDED Requirements

### Requirement: Approbation refusée tant que des tâches restent à valider
`POST /api/workspaces/{id}/changes/{name}/review/approve` SHALL refuser l'approbation avec `409` et le code `tasks_pending` tant que le `tasks.md` de `feature/<change>` contient au moins une tâche non cochée, qu'elle porte ou non le marqueur de validation humaine. La réponse SHALL porter le nombre de tâches non cochées (`remaining`) et un message lisible. Le refus SHALL être décidé avant toute intégration, validation ou fusion et SHALL conserver la branche, le worktree, le travail committé et le marqueur de revue tels qu'avant l'appel. Le contrôle SHALL être fait sous le verrou des actions de revue du change, de sorte qu'une coche concurrente soit prise en compte. Il SHALL NE PAS s'appliquer à une nouvelle approbation d'un change dont la branche est déjà fusionnée (reprise du nettoyage), ni à un change dont la branche n'a pas de `tasks.md` exploitable.

#### Scenario: Tâches restantes à valider
- **WHEN** l'approbation est demandée pour `add-user-auth`, en revue, dont la branche a 2 tâches décochées (dont une marquée)
- **THEN** la réponse est `409` avec le code `tasks_pending` et `remaining` valant 2, et rien n'est fusionné ni modifié

#### Scenario: Tâches toutes cochées
- **WHEN** l'utilisateur a coché toutes les tâches de la branche en revue puis demande l'approbation
- **THEN** l'approbation suit son déroulement habituel (intégration, validation, fusion)

#### Scenario: Tâche décochée par l'utilisateur
- **WHEN** l'utilisateur décoche une tâche faite par l'agent d'un change en revue, puis demande l'approbation
- **THEN** la réponse est `409` avec le code `tasks_pending` et `remaining` valant 1

#### Scenario: Branche déjà fusionnée
- **WHEN** l'approbation est rejouée pour un change dont la branche est déjà fusionnée dans la branche cible
- **THEN** le contrôle des tâches n'est pas appliqué et le nettoyage se termine

### Requirement: Approbation désactivée dans l'interface tant que des tâches restent à valider
Lorsque la liste des tâches d'un change en To Review contient au moins une tâche non cochée, le bouton « Approuver & Fusionner » de l'onglet Actions du DetailPanel SHALL rester visible mais désactivé, avec une info-bulle indiquant le nombre de tâches à valider (« N tâche(s) à valider » en français, « N task(s) to validate » en anglais). Le bouton SHALL redevenir actif dès que toutes les tâches sont cochées, sans rechargement de la page. Si le backend répond `409` avec le code `tasks_pending`, l'interface SHALL afficher un message lisible indiquant que des tâches restent à valider, sans fermer le DetailPanel ni déplacer la carte. Les libellés SHALL être disponibles en français et en anglais.

#### Scenario: Bouton désactivé avec tâches restantes
- **WHEN** l'utilisateur ouvre l'onglet Actions d'un change en revue qui a 2 tâches non cochées
- **THEN** « Approuver & Fusionner » est visible et désactivé, et son info-bulle indique « 2 tâches à valider »

#### Scenario: Bouton actif après avoir tout coché
- **WHEN** l'utilisateur coche la dernière tâche depuis l'onglet Tâches
- **THEN** « Approuver & Fusionner » devient actif sans rechargement

#### Scenario: Refus du backend
- **WHEN** le backend répond `409` avec le code `tasks_pending`
- **THEN** un message indique que des tâches restent à valider, la carte reste dans To Review et le DetailPanel reste ouvert
