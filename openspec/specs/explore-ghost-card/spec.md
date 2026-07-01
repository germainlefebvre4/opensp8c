## Purpose

Gérer le cycle de vie du ghost card dans le kanban : création au premier message, nommage par le LLM via `ghost_named`, affichage distinct (bordure pointillée, badge "exploring"), et persistance dans `preferences.json`.

## Requirements

### Requirement: Création du ghost card au premier message
Au premier message envoyé dans une session anonyme, le backend SHALL créer un ghost record dans `preferences.json` et émettre un event SSE `ghost_card_created` au frontend. Aucun dossier workspace n'est créé à ce stade.

#### Scenario: Premier message déclenche la création du ghost record
- **WHEN** l'utilisateur envoie son premier message dans un panel d'exploration anonyme
- **THEN** le backend crée un ghost record `{id, workspaceId, name: "explore-<6chars>", sessionId, createdAt}` dans `preferences.json`, émet un event SSE `ghost_card_created` avec le ghost record, et la carte apparaît dans la colonne "to-explore" avec le label "Exploring..."

#### Scenario: Ouverture du panel sans message — aucun ghost créé
- **WHEN** l'utilisateur ouvre le panel d'exploration anonyme mais ne envoie aucun message avant de le fermer
- **THEN** aucun ghost record n'est créé dans `preferences.json` et aucune carte n'apparaît dans le kanban

#### Scenario: Messages suivants n'en créent pas d'autres
- **WHEN** l'utilisateur envoie un deuxième message ou plus dans la même session anonyme
- **THEN** aucun nouveau ghost record n'est créé

### Requirement: Nommage LLM du ghost card via event ghost_named
Le LLM SHALL émettre un marker `ghost_named` dans sa première réponse. Le backend SHALL détecter ce marker dans le stream stdout, mettre à jour le ghost record et broadcaster l'event SSE au frontend.

#### Scenario: LLM émet le marker ghost_named
- **WHEN** le subprocess de la session anonyme produit une ligne contenant `{"event":"ghost_named","name":"<name>"}` sur stdout
- **THEN** le backend met à jour le ghost record (champ `name`), émet un event SSE `ghost_named` avec le nom, et la carte kanban se renomme avec le nom reçu

#### Scenario: Collision de nom — suffixe automatique
- **WHEN** le nom proposé par ghost_named correspond à un change existant dans le workspace
- **THEN** le backend ajoute un suffixe `-2` (puis `-3`, etc.) jusqu'à trouver un nom disponible, et utilise ce nom suffixé dans le ghost record et l'event SSE

#### Scenario: Ghost_named non reçu après la première réponse
- **WHEN** la première réponse de l'IA se termine sans avoir émis de marker ghost_named
- **THEN** le ghost card conserve son id temporaire `explore-<6chars>` comme nom

### Requirement: Affichage distinct du ghost card dans le kanban
Le ghost card SHALL être affiché dans la colonne "to-explore" avec un traitement visuel différencié des changes normaux : bordure pointillée, badge "exploring", sans tags ni barre de progression.

#### Scenario: Ghost card visible dans "to-explore"
- **WHEN** un ghost record existe dans `preferences.json` pour le workspace courant
- **THEN** la colonne "to-explore" affiche une carte avec bordure pointillée, le nom du ghost card, un badge "exploring" et aucun autre badge ni barre de progression

#### Scenario: Ghost card non draggable avant nommage
- **WHEN** le ghost card est encore en phase de nommage (label "Exploring...")
- **THEN** le drag-and-drop est désactivé sur cette carte

#### Scenario: Ghost card draggable après nommage
- **WHEN** le ghost card a reçu son nom via ghost_named
- **THEN** la carte est draggable vers la colonne "todo"

### Requirement: Persistance des ghost records au redémarrage serveur
Le backend SHALL charger les ghost records depuis `preferences.json` au démarrage. L'endpoint de listing des changes SHALL inclure les ghost records dans la réponse.

#### Scenario: Ghost records chargés au démarrage
- **WHEN** le serveur démarre et des ghost records existent dans `preferences.json`
- **THEN** l'API `/api/workspaces/{id}/changes` retourne les ghost records parmi les changes, avec un champ `kanban_status: "to-explore"` et `is_ghost: true`

#### Scenario: Ghost record absent du workspace actif
- **WHEN** un ghost record référence un workspaceId différent du workspace courant
- **THEN** ce ghost record n'apparaît pas dans la liste des changes du workspace courant
