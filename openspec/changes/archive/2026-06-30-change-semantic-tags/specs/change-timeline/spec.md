## ADDED Requirements

### Requirement: Vue Timeline accessible depuis la navigation
L'application SHALL proposer une vue `/timeline` accessible depuis la navigation principale. Cette vue affiche l'ensemble des changes du workspace (actifs et archivés) en ordre antéchronologique, groupés par mois de création.

#### Scenario: Accès à la timeline
- **WHEN** l'utilisateur clique sur l'entrée "Timeline" dans la navigation
- **THEN** la vue `/timeline` s'affiche avec la liste de tous les changes du workspace, actifs et archivés, triés du plus récent au plus ancien

#### Scenario: Groupement par mois
- **WHEN** la timeline est affichée
- **THEN** les changes sont groupés sous des en-têtes de mois ("Juin 2026", "Mai 2026", etc.)

### Requirement: Affichage des métadonnées sémantiques sur chaque entrée timeline
Chaque entrée de la timeline SHALL afficher : le nom du change, sa date de création, son statut Kanban, et — si disponibles — son type applicatif et son niveau de complexité. Les composants touchés SHALL être affichés sous le nom du change sous forme de chips discrets.

#### Scenario: Entrée avec tags complets
- **WHEN** un change possède une section `tags` avec `type`, `complexity` et `components`
- **THEN** l'entrée affiche le badge de type, l'indicateur de complexité, et les chips de composants

#### Scenario: Entrée sans tags
- **WHEN** un change ne possède pas encore de section `tags`
- **THEN** l'entrée s'affiche sans badge ni chips, sans erreur

### Requirement: Filtrage de la timeline par tags
L'utilisateur SHALL pouvoir filtrer les entrées de la timeline en sélectionnant un ou plusieurs tags (type applicatif et/ou composants). Les filtres actifs sont affichés comme chips supprimables. La timeline se met à jour instantanément à chaque modification des filtres.

#### Scenario: Filtre par type applicatif
- **WHEN** l'utilisateur sélectionne le filtre "frontend"
- **THEN** seules les entrées dont le tag `type` est `frontend` sont affichées

#### Scenario: Filtre par composant
- **WHEN** l'utilisateur sélectionne un composant (ex. "explore-panel")
- **THEN** seules les entrées dont le tableau `components` contient "explore-panel" sont affichées

#### Scenario: Combinaison de filtres
- **WHEN** l'utilisateur sélectionne type "frontend" et composant "search-bar"
- **THEN** seules les entrées satisfaisant les deux critères simultanément sont affichées

#### Scenario: Suppression d'un filtre
- **WHEN** l'utilisateur clique sur le × d'un chip de filtre actif
- **THEN** ce filtre est retiré et la timeline se met à jour

#### Scenario: Aucun résultat
- **WHEN** la combinaison de filtres actifs ne correspond à aucun change
- **THEN** la timeline affiche un message "Aucun changement ne correspond aux filtres sélectionnés"

### Requirement: Heatmap des composants les plus modifiés
La vue Timeline SHALL afficher une section "Composants fréquents" présentant les composants les plus présents parmi les changes de la période filtrée, avec un indicateur de fréquence (nombre de changes). Un clic sur un composant dans la heatmap l'ajoute aux filtres actifs.

#### Scenario: Heatmap avec filtres actifs
- **WHEN** des filtres sont actifs
- **THEN** la heatmap reflète uniquement les composants présents dans les changes filtrés

#### Scenario: Clic sur un composant dans la heatmap
- **WHEN** l'utilisateur clique sur un composant dans la heatmap
- **THEN** ce composant est ajouté aux filtres actifs de la timeline
