## MODIFIED Requirements

### Requirement: Affichage distinct du ghost card dans le kanban
Le ghost card SHALL être affiché dans la colonne "to-explore" avec un traitement visuel différencié des changes normaux : bordure pointillée, badge "exploring", sans tags d'implémentation. Si un brouillon de tâches existe pour cette exploration, la carte SHALL afficher un indicateur de progression de brouillon distinct (ex: barre en pointillés de couleur violette) ou un indicateur textuel du nombre de tâches du brouillon (ex: "3 draft tasks").

#### Scenario: Ghost card visible dans "to-explore" sans brouillon
- **WHEN** un ghost record existe dans `preferences.json` pour le workspace courant ET qu'aucun fichier de brouillon de tâche n'existe
- **THEN** la colonne "to-explore" affiche une carte avec bordure pointillée, le nom du ghost card, un badge "exploring" et aucun indicateur de progression

#### Scenario: Ghost card visible dans "to-explore" avec brouillon
- **WHEN** un ghost record existe dans `preferences.json` pour le workspace courant ET qu'un fichier de brouillon contenant des tâches de brouillon existe
- **THEN** la colonne "to-explore" affiche une carte avec bordure pointillée, le nom du ghost card, un badge "exploring", et un indicateur visuel de progression de brouillon ou un badge textuel "N draft tasks"

#### Scenario: Ghost card non draggable avant nommage
- **WHEN** le ghost card is encore en phase de nommage (label "Exploring...")
- **THEN** le drag-and-drop est désactivé sur cette carte

#### Scenario: Ghost card draggable après nommage
- **WHEN** le ghost card a reçu son nom via ghost_named
- **THEN** la carte est draggable vers la colonne "todo"
