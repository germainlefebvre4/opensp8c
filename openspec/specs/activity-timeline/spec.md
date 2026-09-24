# activity-timeline Specification

## Purpose

Fournit, pour chaque change, un flux chronologique unifié des événements agent et non-agent (cycle de vie Kanban, pool d'agents, git), affiché dans l'onglet Conversation du DetailPanel.

## Requirements

### Requirement: Persistance des entrées d'activité non-agent
Le backend SHALL maintenir un `ActivityStore` qui persiste, pour chaque change, les entrées d'activité qui ne proviennent pas de l'agent, sous forme d'un fichier JSONL append-only à l'emplacement `<config-dir>/activity/<workspaceID>/<changeName>/activity.jsonl`. Chaque ligne SHALL contenir au minimum un horodatage (`ts`), un type d'événement, une catégorie utilisée pour la coloration, et un résumé textuel. Une entrée liée à un événement instantané (toggle de tâche, déclenchement d'un run, reset de tâches, transition de statut worker, commit git détecté) ne porte pas de durée.

#### Scenario: Append d'une nouvelle entrée
- **WHEN** un événement non-agent éligible se produit pour un change (voir exigences dédiées ci-dessous)
- **THEN** une nouvelle ligne est ajoutée à `activity/<workspaceID>/<changeName>/activity.jsonl` sans altérer les lignes précédentes

#### Scenario: Échec d'écriture non bloquant
- **WHEN** l'écriture d'une entrée dans `activity.jsonl` échoue (ex. disque plein)
- **THEN** la requête HTTP ou l'opération à l'origine de l'événement n'échoue pas pour autant ; l'échec d'écriture de l'entrée d'activité est journalisé côté serveur sans impacter le flux principal

### Requirement: Entrée d'activité au toggle d'une tâche
Le backend SHALL enregistrer une entrée d'activité chaque fois qu'une tâche de `tasks.md` change d'état (cochée ou décochée) via l'endpoint de toggle existant, incluant le texte de la tâche et son nouvel état.

#### Scenario: Tâche cochée
- **WHEN** une tâche non complétée est basculée à l'état complété via l'endpoint de toggle
- **THEN** une entrée d'activité de type toggle de tâche est ajoutée pour ce change, avec le texte de la tâche et l'état "complétée"

### Requirement: Entrée d'activité au déclenchement et au reset des tâches d'un run
Le backend SHALL enregistrer une entrée d'activité lorsqu'un run est déclenché pour un change, et une entrée distincte lorsque les tâches d'un change sont réinitialisées.

#### Scenario: Déclenchement d'un run
- **WHEN** un run est déclenché pour un change
- **THEN** une entrée d'activité de type déclenchement de run est ajoutée pour ce change

#### Scenario: Reset des tâches
- **WHEN** les tâches d'un change sont réinitialisées
- **THEN** une entrée d'activité de type reset de tâches est ajoutée pour ce change

### Requirement: Entrée d'activité à la transition de statut d'un worker du pool
Le backend SHALL enregistrer une entrée d'activité pour le change concerné chaque fois qu'un worker du pool d'agents change de statut (`idle`, `working`, `testing`, `healing`, `paused`) ou se voit assigner un nouveau change, en plus de l'événement `pool_updated` déjà émis.

#### Scenario: Worker prend en charge un change
- **WHEN** un worker du pool passe à l'état `working` sur un change donné
- **THEN** une entrée d'activité de type transition de statut worker est ajoutée pour ce change, incluant le nouveau statut

### Requirement: Détection best-effort des commits git
Le backend SHALL détecter périodiquement, pour chaque change disposant d'un worktree/branche actif, les nouveaux commits apparus sur sa branche depuis la dernière détection, et enregistrer une entrée d'activité par commit détecté (SHA court, message de commit). Cette détection est best-effort et ne garantit pas de distinguer un commit produit par l'agent d'un commit produit manuellement par un humain.

#### Scenario: Nouveau commit détecté sur la branche d'un change
- **WHEN** le job périodique de détection s'exécute et trouve un ou plusieurs commits nouveaux sur la branche d'un change depuis la dernière exécution
- **THEN** une entrée d'activité de type commit git est ajoutée par nouveau commit, dans l'ordre chronologique

#### Scenario: Aucun nouveau commit
- **WHEN** le job périodique de détection s'exécute et qu'aucun commit nouveau n'est trouvé sur la branche d'un change
- **THEN** aucune entrée d'activité n'est ajoutée pour ce change

### Requirement: Endpoint de lecture fusionnée
Le backend SHALL exposer `GET /api/workspaces/{id}/changes/{name}/activity` retournant, triées par ordre chronologique croissant, la fusion des entrées persistées dans `ActivityStore` et des entrées dérivées des runs de conversation existants du change (narration agent et appels d'outils). Chaque entrée d'appel d'outil dérivée SHALL inclure le nom de l'outil et une durée calculée à partir des horodatages d'appel et de résultat de cet outil ; les autres entrées n'ont pas de durée.

#### Scenario: Change avec activité agent et non-agent
- **WHEN** le frontend appelle `GET /changes/{name}/activity` pour un change ayant à la fois des runs de conversation et des entrées `ActivityStore`
- **THEN** la réponse contient une liste fusionnée triée chronologiquement, mêlant entrées agent (avec durée pour les appels d'outils) et entrées non-agent (sans durée)

#### Scenario: Change sans aucune activité
- **WHEN** le frontend appelle `GET /changes/{name}/activity` pour un change n'ayant encore ni run de conversation ni entrée `ActivityStore`
- **THEN** la réponse retourne une liste vide sans erreur

### Requirement: Onglet Conversation du DetailPanel
Le `DetailPanel` SHALL exposer un onglet **"Conversation"** (remplaçant l'ancien onglet "Log") affichant le flux fusionné du change en lecture seule, sous forme d'une liste chronologique d'entrées. Chaque entrée SHALL afficher un badge identifiant son type, un horodatage, et sa durée lorsqu'elle en a une. Aucune zone de saisie n'est présente dans cet onglet.

#### Scenario: Onglet Conversation avec activité
- **WHEN** l'utilisateur ouvre le DetailPanel d'un change ayant au moins une entrée dans le flux fusionné
- **THEN** l'onglet "Conversation" est visible et affiche la liste chronologique complète, la plus récente en bas ou en haut selon l'ordre choisi par défaut

#### Scenario: Onglet Conversation sans activité
- **WHEN** le change n'a encore aucune entrée dans le flux fusionné
- **THEN** l'onglet "Conversation" est visible mais affiche un message vide, sans erreur

### Requirement: Frise chronologique monoligne colorée par type
L'onglet Conversation SHALL afficher, au-dessus de la liste, une frise chronologique sur une seule ligne représentant l'ensemble des entrées du flux fusionné. Chaque entrée SHALL être rendue avec une couleur déterminée uniquement par son type d'action, identique à la couleur du badge correspondant dans la liste, et associée à une légende. Une entrée d'appel d'outil agent SHALL être rendue comme un segment dont la largeur est proportionnelle à sa durée réelle. Une entrée sans durée (toggle de tâche, déclenchement de run, reset de tâches, transition de statut worker, commit git, narration agent) SHALL être rendue comme un marqueur ponctuel positionné à son horodatage.

#### Scenario: Deux appels d'outils de durées différentes
- **WHEN** la frise affiche un appel d'outil de 4 secondes et un appel d'outil de 400 millisecondes
- **THEN** le segment du premier est visuellement environ 10 fois plus large que celui du second

#### Scenario: Survol d'un segment ou marqueur
- **WHEN** l'utilisateur survole un segment ou un marqueur de la frise
- **THEN** un tooltip affiche le type d'action, l'horodatage, et la durée si elle existe

#### Scenario: Légende des types d'action
- **WHEN** l'onglet Conversation est affiché
- **THEN** une légende associant chaque type d'action rencontré dans le flux à sa couleur est visible

### Requirement: Filtre par type d'action dans l'onglet Conversation
L'utilisateur SHALL pouvoir filtrer les entrées affichées dans la liste et sur la frise en sélectionnant ou désélectionnant un ou plusieurs types d'action depuis la légende.

#### Scenario: Désactivation d'un type
- **WHEN** l'utilisateur désélectionne le type "Bash" dans la légende
- **THEN** les entrées de type "Bash" disparaissent de la liste et de la frise, les autres types restent affichés

### Requirement: Mise à jour temps réel de l'onglet Conversation
Pendant que le DetailPanel d'un change est ouvert sur l'onglet Conversation, l'arrivée d'une nouvelle entrée d'activité pour ce change SHALL être reflétée sans que l'utilisateur ait à recharger la page, via le mécanisme SSE existant.

#### Scenario: Nouvelle entrée pendant consultation
- **WHEN** l'utilisateur consulte l'onglet Conversation d'un change et qu'une nouvelle entrée d'activité est ajoutée côté serveur pour ce change
- **THEN** la liste et la frise se mettent à jour avec la nouvelle entrée sans action manuelle de l'utilisateur
