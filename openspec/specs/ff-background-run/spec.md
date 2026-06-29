## Purpose

Spec du lancement du ff en arrière-plan : endpoint POST /ff, namespace de session isolé, auto-nettoyage, et événements SSE du cycle de vie.

## Requirements

### Requirement: Déclenchement du ff en arrière-plan
Le backend SHALL exposer un endpoint `POST /api/workspaces/{id}/changes/{name}/ff` qui spawn un subprocess ff fire-and-forget dans le répertoire du workspace. Le subprocess SHALL recevoir `/opsx:ff` comme premier message stdin. La réponse HTTP SHALL être retournée immédiatement (202 Accepted) sans attendre la fin du subprocess.

#### Scenario: Déclenchement réussi
- **WHEN** le frontend envoie `POST /changes/{name}/ff` pour un changement sans ff actif
- **THEN** le backend spawn le subprocess, retourne 202, et émet l'événement SSE `ff_started`

#### Scenario: ff déjà en cours (guard)
- **WHEN** le frontend envoie `POST /changes/{name}/ff` pour un changement dont un subprocess ff est déjà actif
- **THEN** le backend retourne 409 Conflict sans spawner de nouveau subprocess

#### Scenario: Changement introuvable
- **WHEN** le frontend envoie `POST /changes/{name}/ff` pour un nom de changement inexistant
- **THEN** le backend retourne 404 Not Found

### Requirement: Namespace de session ff isolé
Les sessions ff SHALL utiliser la clé `workspaceID + "/__ff__/" + changeName` dans le session manager, distincte des clés des sessions explore (`workspaceID + "/" + changeName`). Un ff actif n'interfère pas avec une session explore existante pour le même changement.

#### Scenario: Clé ff distincte de la clé explore
- **WHEN** un ff est déclenché pour un changement dont une session explore est active
- **THEN** les deux sessions coexistent dans le manager sans collision de clé

### Requirement: Auto-nettoyage de la session ff
Quand le subprocess ff se termine (stdout fermé), la session ff SHALL être retirée du manager automatiquement par la goroutine fanOut, sans attendre le timeout d'inactivité.

#### Scenario: Nettoyage automatique après complétion
- **WHEN** le subprocess ff termine normalement
- **THEN** la session est supprimée du manager et un nouveau `POST /ff` pour le même changement est accepté

### Requirement: Événements SSE du cycle de vie ff
Le backend SHALL émettre trois types d'événements SSE dans le stream `/api/workspaces/{id}/events` pour chaque run ff :
- `ff_started` : émis au démarrage du subprocess
- `ff_done` : émis quand le subprocess se termine avec succès (exit code 0)
- `ff_failed` : émis quand le subprocess se termine en erreur (exit code non-0 ou erreur de spawn)

#### Scenario: Événement ff_started
- **WHEN** le subprocess ff démarre avec succès
- **THEN** le stream SSE du workspace émet `{"type":"ff_started","name":"<changeName>"}`

#### Scenario: Événement ff_done
- **WHEN** le subprocess ff se termine normalement
- **THEN** le stream SSE émet `{"type":"ff_done","name":"<changeName>"}`

#### Scenario: Événement ff_failed
- **WHEN** le subprocess ff se termine avec une erreur ou ne peut pas être spawné
- **THEN** le stream SSE émet `{"type":"ff_failed","name":"<changeName>","error":"<message>"}`

### Requirement: Spinner sur la carte pendant ff
La carte d'un changement SHALL afficher un indicateur de chargement (spinner) entre la réception de l'événement `ff_started` et la réception de `ff_done` ou `ff_failed`. Le spinner indique que la carte est non-draggable.

#### Scenario: Spinner affiché après ff_started
- **WHEN** l'événement SSE `ff_started` est reçu pour un changement
- **THEN** la carte de ce changement affiche un spinner et le drag est désactivé

#### Scenario: Spinner retiré après ff_done
- **WHEN** l'événement SSE `ff_done` est reçu
- **THEN** le spinner disparaît, la carte affiche son état normal (la colonne a déjà changé via change_updated)

#### Scenario: Erreur affichée après ff_failed
- **WHEN** l'événement SSE `ff_failed` est reçu
- **THEN** le spinner est remplacé par un indicateur d'erreur sur la carte, le drag est réactivé pour permettre une nouvelle tentative
