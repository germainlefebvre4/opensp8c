# Kanban Ready Column Specification

## Purpose

Définit l'état persistant "lancé"/"rang de priorité" qui distingue la colonne Ready de la colonne To Do, les actions de promotion/rétrogradation entre les deux, et le réordonnancement par priorité des changes en attente de lancement.

## Requirements

### Requirement: Marqueur persistant "lancé" par change

Chaque change SHALL avoir un marqueur booléen persistant `launched`, stocké dans son fichier `.openspec.yaml`. Un change dont `tasks.md` existe avec au moins une tâche, et dont `launched` est absent ou `false`, SHALL avoir le statut Kanban `ready`. Un change dans les mêmes conditions de `tasks.md` mais avec `launched: true` (ou le champ absent, voir compatibilité ascendante) SHALL avoir le statut Kanban `todo` (ou son statut dérivé habituel si des tâches sont déjà cochées).

#### Scenario: Change avec tasks.md généré et non lancé
- **WHEN** un change a un fichier `tasks.md` non vide et `launched: false` (ou absent après création via le flux Ready) dans son `.openspec.yaml`
- **THEN** `GET /changes` retourne `kanban_status: "ready"` pour ce change

#### Scenario: Change lancé avec tâches non démarrées
- **WHEN** un change a un fichier `tasks.md` non vide, `launched: true`, et aucune tâche cochée
- **THEN** `GET /changes` retourne `kanban_status: "todo"` pour ce change

#### Scenario: Compatibilité ascendante — champ absent traité comme lancé
- **WHEN** un change possède un `tasks.md` non vide mais son `.openspec.yaml` ne contient aucun champ `launched` (change créé avant l'introduction de cette fonctionnalité, ou créé hors du flux applicatif Ready)
- **THEN** le champ est traité comme `launched: true` et le change conserve son statut dérivé habituel (`todo`, `in-progress` ou `done`), sans jamais apparaître en `ready`

### Requirement: Déclenchement du Fast-Forward vers Ready

Le déclenchement du Fast-Forward (drag `To Explore → Ready`, ou promotion d'un ghost card) SHALL générer les artefacts (`proposal.md`, `design.md`, `tasks.md`) exactement comme le fait aujourd'hui le déclenchement vers `To Do`, et SHALL écrire `launched: false` dans le `.openspec.yaml` du change nouvellement créé.

#### Scenario: FF direct depuis To Explore
- **WHEN** un change normal (non-ghost) est déposé de `To Explore` sur `Ready`
- **THEN** le FF est déclenché directement, et à sa complétion le `.openspec.yaml` du change contient `launched: false`

#### Scenario: FF via promotion de ghost card
- **WHEN** un ghost card nommé est déposé sur `Ready` et la promotion est confirmée
- **THEN** le change créé par la promotion a `launched: false` dans son `.openspec.yaml` et apparaît en colonne `Ready`

### Requirement: Promotion manuelle Ready → To Do

Le backend SHALL exposer une action permettant de marquer un change `launched: true`, rendant la carte éligible au ramassage par l'Agent Pool. Cette action ne SHALL modifier ni `tasks.md` ni aucun autre artefact du change.

#### Scenario: Promotion réussie
- **WHEN** l'utilisateur déclenche la promotion d'un change en colonne `Ready` (drag vers `To Do` ou bouton dédié)
- **THEN** le backend écrit `launched: true` dans le `.openspec.yaml` du change, le watcher SSE émet `change_updated`, et la carte apparaît en colonne `To Do`

### Requirement: Rétrogradation manuelle To Do → Ready

Le backend SHALL exposer une action symétrique permettant de remarquer un change `launched: false`, le retirant de l'éligibilité au ramassage par l'Agent Pool. Cette action SHALL être refusée si un worker de l'Agent Pool est actuellement actif sur ce change.

#### Scenario: Rétrogradation réussie
- **WHEN** l'utilisateur déclenche la rétrogradation d'un change en colonne `To Do` sans worker actif (drag vers `Ready` ou bouton dédié)
- **THEN** le backend écrit `launched: false` dans le `.openspec.yaml` du change, le watcher SSE émet `change_updated`, et la carte apparaît en colonne `Ready`

#### Scenario: Rétrogradation refusée pendant exécution
- **WHEN** l'utilisateur tente de rétrograder un change dont `worker_active` vaut `true`
- **THEN** le backend refuse l'action avec une erreur 409, et le change reste `launched: true`

### Requirement: Rang de priorité persistant et réordonnancement

Chaque change ayant un `tasks.md` généré SHALL avoir un rang de priorité entier persistant (`order`) dans son `.openspec.yaml`, attribué à la suite des rangs existants au moment de la génération. Les cartes de la colonne `Ready` SHALL être réordonnables par drag-and-drop ; un déplacement SHALL persister le nouveau rang du change déplacé et de tout change dont le rang relatif change en conséquence.

#### Scenario: Attribution du rang initial
- **WHEN** un Fast-Forward vers `Ready` se termine pour un change
- **THEN** son `.openspec.yaml` reçoit un `order` strictement supérieur à celui de tous les changes déjà présents en `Ready` ou `To Do`

#### Scenario: Réordonnancement par drag-and-drop dans Ready
- **WHEN** l'utilisateur glisse une carte de la colonne `Ready` vers une nouvelle position au sein de la même colonne
- **THEN** le backend persiste le nouveau rang du change déplacé, la liste `GET /changes` reflète le nouvel ordre après rafraîchissement, et l'ordre survit à un rechargement de la page

#### Scenario: Rang conservé à la promotion
- **WHEN** un change passe de `Ready` à `To Do` (promotion)
- **THEN** son rang de priorité `order` est conservé inchangé

### Requirement: Réinitialisation de l'état de lancement au reset

Le reset d'un change vers `To Explore` (vidage de `tasks.md`) SHALL également effacer les champs `launched` et `order` de son `.openspec.yaml`, afin qu'une future génération de `tasks.md` reparte d'un état de lancement propre.

#### Scenario: Reset efface l'état Ready
- **WHEN** un change en colonne `Ready` ou `To Do` est réinitialisé vers `To Explore`
- **THEN** son `.openspec.yaml` ne contient plus les champs `launched` ni `order` après le reset
