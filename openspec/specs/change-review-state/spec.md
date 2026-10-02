# change-review-state Specification

## Purpose

Définit l'état « en revue » d'un change : un marqueur persistant, attaché à la branche du change, dont dérive le statut To Review, afin que ce statut survive aux redémarrages et reste cohérent pour tous les consommateurs (Kanban, détail, compteurs, dispatcher).

## Requirements

### Requirement: Marqueur de revue posé à la fin du worker en hitl-review
Lorsqu'un worker en mode `hitl-review` termine un changement avec succès (travail committé dans `feature/<change>`, issue `awaiting-review`), le backend SHALL enregistrer un marqueur de revue pour ce changement. Le marqueur SHALL être stocké dans la configuration git du dépôt, attaché à la branche `feature/<change>`, de sorte qu'il disparaisse avec la branche. Poser le marqueur SHALL NOT modifier un fichier du workspace (l'arbre de travail du dépôt principal reste inchangé). Si le marqueur ne peut pas être enregistré, le worker SHALL NOT terminer en `awaiting-review` : il SHALL passer à `paused` avec une raison lisible indiquant que l'état de revue n'a pas pu être enregistré.

#### Scenario: Marqueur posé à la fin du worker
- **WHEN** le worker du changement `add-user-auth` termine en `hitl-review` avec l'issue `awaiting-review`
- **THEN** un marqueur de revue est enregistré pour `add-user-auth` et le travail reste committé dans `feature/add-user-auth`

#### Scenario: Aucun fichier du workspace modifié
- **WHEN** le marqueur de revue est posé
- **THEN** `git status` du dépôt principal ne signale aucune modification supplémentaire

#### Scenario: Marqueur supprimé avec la branche
- **WHEN** la branche `feature/add-user-auth` est supprimée après fusion
- **THEN** le marqueur de revue de `add-user-auth` n'existe plus

#### Scenario: Échec d'enregistrement du marqueur
- **WHEN** l'enregistrement du marqueur échoue à la fin du worker
- **THEN** le worker passe à `paused` avec une raison indiquant que l'état de revue n'a pas pu être enregistré, et le changement n'est pas présenté comme en revue

### Requirement: Statut To Review dérivé du marqueur
Tout changement actif portant un marqueur de revue SHALL avoir le statut Kanban `to-review`, dans la liste des changes, dans le détail d'un change et dans les compteurs par workspace. Ce statut SHALL primer sur celui déduit de l'avancement des tasks et de l'état « lancé » du change : l'avancement du `tasks.md` du workspace principal SHALL NOT influer sur lui. Un marqueur dont la branche `feature/<change>` n'existe plus, ou dont le change n'est plus actif, SHALL être ignoré. Si les marqueurs ne peuvent pas être lus (dépôt non git, commande git indisponible), le statut SHALL être dérivé comme sans marqueur, sans erreur.

#### Scenario: Change en revue avec tasks non cochées dans le workspace
- **WHEN** un changement porte un marqueur de revue et que le `tasks.md` de son workspace principal est à `0/N`
- **THEN** son statut est `to-review`

#### Scenario: Change en revue marqué lancé
- **WHEN** un changement lancé porte un marqueur de revue
- **THEN** son statut est `to-review` et non `todo`

#### Scenario: Change sans marqueur
- **WHEN** un changement ne porte aucun marqueur de revue
- **THEN** son statut est dérivé de l'avancement de ses tasks et de son état « lancé », comme avant

#### Scenario: Marqueur orphelin ignoré
- **WHEN** un marqueur de revue existe pour un changement dont la branche `feature/<change>` n'existe plus
- **THEN** le marqueur est ignoré et le statut est dérivé sans lui

#### Scenario: Dépôt non git
- **WHEN** le workspace n'est pas un dépôt git
- **THEN** la liste des changes est retournée sans erreur, avec des statuts dérivés sans marqueur

### Requirement: Persistance de l'état de revue
L'état de revue d'un change SHALL survivre à l'arrêt et au redémarrage du pool d'agents ainsi qu'au redémarrage du backend. Il SHALL être indépendant de l'exécution du pool : un changement en revue SHALL rester en To Review lorsque le pool est arrêté ou lorsque le mode de délégation du workspace est modifié. Seule une action explicite de l'utilisateur (voir `change-review-actions`) ou la suppression de la branche SHALL lever le marqueur.

#### Scenario: Redémarrage du pool
- **WHEN** le pool est arrêté puis redémarré alors que `add-user-auth` est en revue
- **THEN** `add-user-auth` reste en To Review et aucun worker ne lui est assigné

#### Scenario: Redémarrage du backend
- **WHEN** le backend est redémarré alors que `add-user-auth` est en revue
- **THEN** `add-user-auth` est toujours affiché en To Review

#### Scenario: Pool arrêté
- **WHEN** le pool est arrêté alors qu'un changement est en revue
- **THEN** le changement reste en To Review

#### Scenario: Changement de mode de délégation
- **WHEN** le mode de délégation du workspace passe de `hitl-review` à `full-autonomy` alors qu'un changement est en revue
- **THEN** le changement reste en To Review jusqu'à une action explicite de l'utilisateur

### Requirement: Rafraîchissement du Kanban à chaque changement d'état de revue
Poser ou lever un marqueur de revue ne modifiant aucun fichier OpenSpec, le backend SHALL publier explicitement l'événement SSE `change_updated` du change concerné sur le flux d'événements du workspace, afin que le Kanban et le détail du change se rafraîchissent sans action de l'utilisateur.

#### Scenario: Carte déplacée en To Review à la fin du worker
- **WHEN** un worker termine en `awaiting-review` et que le marqueur est posé
- **THEN** un événement `change_updated` est publié pour ce change et la carte apparaît en To Review sans rechargement manuel

#### Scenario: Carte retirée de To Review à la levée du marqueur
- **WHEN** le marqueur de revue d'un change est levé
- **THEN** un événement `change_updated` est publié pour ce change et la carte quitte la colonne To Review
