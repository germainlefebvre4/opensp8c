# Spec: change-timeline

## Purpose

Vue chronologique de tous les changes du workspace. Permet de visualiser l'historique complet des changes (actifs et archivés) regroupés par mois, avec filtrage par spec et tags sémantiques, heatmap des specs fréquentes, et toggle vers le mode Matrice spec × temps.

## Requirements

### Requirement: Vue Timeline accessible depuis la navigation
L'application SHALL proposer une vue `/timeline` accessible depuis la navigation principale. Cette vue affiche l'ensemble des changes du workspace (actifs et archivés) en ordre antéchronologique, groupés par mois de création.

#### Scenario: Accès à la timeline
- **WHEN** l'utilisateur clique sur l'entrée "Timeline" dans la navigation
- **THEN** la vue `/timeline` s'affiche avec la liste de tous les changes du workspace, actifs et archivés, triés du plus récent au plus ancien

#### Scenario: Groupement par mois
- **WHEN** la timeline est affichée
- **THEN** les changes sont groupés sous des en-têtes de mois ("Juin 2026", "Mai 2026", etc.)

### Requirement: Affichage des métadonnées sémantiques sur chaque entrée timeline
Chaque entrée de la timeline SHALL afficher : le nom du change, sa date de création, son statut Kanban, et — si disponibles — un badge par valeur du tableau `tags.type` et son niveau de complexité. Les specs touchées par le change (delta specs) SHALL être affichées sous le nom du change sous forme de chips cliquables distincts. Les composants sémantiques issus des tags qui n'ont pas de spec formelle correspondante SHALL être affichés comme chips secondaires non-cliquables.

#### Scenario: Entrée avec delta specs
- **WHEN** un change a des delta specs (fichiers dans son dossier `specs/`)
- **THEN** chaque spec est affichée sous forme de chip cliquable ; un clic navigue vers `/specs?workspace=<id>&selected=<spec-name>`

#### Scenario: Entrée avec tags et delta specs
- **WHEN** un change a à la fois des delta specs et des `tags.components`
- **THEN** les delta specs sont affichées comme chips primaires ; seuls les composants de tags non couverts par une spec formelle sont affichés en chips secondaires (style atténué)

#### Scenario: Entrée avec plusieurs valeurs de type
- **WHEN** un change a `tags.type: [frontend, backend]`
- **THEN** l'entrée affiche deux badges type distincts, chacun cliquable individuellement pour filtrer

#### Scenario: Entrée avec tags uniquement (pas de delta specs)
- **WHEN** un change a des `tags.components` mais aucune delta spec
- **THEN** les composants de tags sont affichés comme chips secondaires

#### Scenario: Entrée sans tags ni delta specs
- **WHEN** un change ne possède ni `tags` ni delta specs
- **THEN** l'entrée s'affiche sans chips, sans erreur

### Requirement: Filtrage de la timeline par spec et par tags
L'utilisateur SHALL pouvoir filtrer les entrées de la timeline par spec (depuis la heatmap ou en saisissant un nom) et par tag sémantique (type applicatif, composant LLM). Un filtre de type applicatif retient une entrée si son tableau `tags.type` contient la valeur du filtre. Les deux types de filtres (spec et tag) sont cumulables. Les filtres actifs sont affichés comme chips supprimables.

#### Scenario: Filtre par spec
- **WHEN** l'utilisateur sélectionne une spec comme filtre
- **THEN** seuls les changes ayant cette spec dans leurs delta specs sont affichés

#### Scenario: Filtre par type applicatif
- **WHEN** l'utilisateur sélectionne le filtre "frontend"
- **THEN** seules les entrées dont le tableau `tags.type` contient `frontend` sont affichées

#### Scenario: Filtre par type applicatif sur une entrée à plusieurs types
- **WHEN** l'utilisateur sélectionne le filtre "backend" et qu'une entrée a `tags.type: [frontend, backend]`
- **THEN** cette entrée est affichée, car `backend` figure dans son tableau `type`

#### Scenario: Filtre par composant sémantique
- **WHEN** l'utilisateur sélectionne un composant de tag comme filtre
- **THEN** seules les entrées dont le tableau `tags.components` contient ce composant sont affichées

#### Scenario: Combinaison filtre spec + filtre tag
- **WHEN** l'utilisateur active un filtre spec et un filtre tag simultanément
- **THEN** seules les entrées satisfaisant les deux critères sont affichées

#### Scenario: Suppression d'un filtre
- **WHEN** l'utilisateur clique sur le × d'un chip de filtre actif
- **THEN** ce filtre est retiré et la timeline se met à jour

#### Scenario: Aucun résultat
- **WHEN** la combinaison de filtres actifs ne correspond à aucun change
- **THEN** la timeline affiche un message "Aucun changement ne correspond aux filtres sélectionnés"

### Requirement: Heatmap des specs les plus modifiées
La vue Timeline SHALL afficher une section "Specs fréquentes" présentant les specs les plus touchées parmi les changes de la période filtrée, avec un indicateur de fréquence (nombre de changes). La heatmap est calculée depuis les delta specs (endpoint `/specs/overview`) et non depuis `tags.components`. Un clic sur une spec dans la heatmap l'ajoute aux filtres actifs.

#### Scenario: Heatmap affichée avec données delta specs
- **WHEN** la timeline est affichée et des changes ont des delta specs
- **THEN** la heatmap "Specs fréquentes" affiche les specs les plus touchées, triées par fréquence décroissante

#### Scenario: Clic sur une spec dans la heatmap
- **WHEN** l'utilisateur clique sur une spec dans la heatmap
- **THEN** cette spec est ajoutée aux filtres actifs et seuls les changes ayant une delta spec correspondante sont affichés

#### Scenario: Heatmap avec filtres actifs
- **WHEN** des filtres sont actifs
- **THEN** la heatmap reflète uniquement les specs présentes dans les changes filtrés

### Requirement: Toggle Changes / Matrice dans la Timeline
La TimelinePage SHALL afficher un toggle [Changes | Matrice] en haut de page permettant de basculer entre la liste chronologique des changes et la grille spec × temps.

#### Scenario: Toggle visible en permanence
- **WHEN** l'utilisateur est sur la page Timeline
- **THEN** le toggle [Changes | Matrice] est visible en haut de page

#### Scenario: Mode Changes actif par défaut
- **WHEN** l'utilisateur navigue vers `/timeline` sans paramètre `?spec=`
- **THEN** le mode Changes est actif par défaut

#### Scenario: Mode Matrice activé par deep-link
- **WHEN** l'URL contient `?spec=<name>`
- **THEN** le mode Matrice s'active automatiquement

### Requirement: Robustesse de l'entrée timeline face à des tags incomplets
Une entrée de la timeline SHALL s'afficher sans erreur lorsque les tags du change n'ont pas de `type` exploitable (champ absent, `null` ou tableau vide). Dans ce cas, aucun badge de type n'est rendu, et le reste de l'entrée (nom, statut, date, complexité, chips de specs et de composants) reste affiché. Un tag incomplet sur un change ne SHALL jamais empêcher l'affichage de la page Timeline ni des autres entrées.

#### Scenario: Tags sans `type`
- **WHEN** un change a une section `tags` dont le champ `type` est absent ou `null`
- **THEN** l'entrée s'affiche sans badge de type et sans erreur, avec ses autres métadonnées

#### Scenario: Tags avec `type` vide
- **WHEN** un change a `tags.type: []`
- **THEN** l'entrée s'affiche sans badge de type et sans erreur

#### Scenario: Un change à tags incomplets parmi d'autres
- **WHEN** la timeline contient un change à tags incomplets et d'autres changes correctement tagués
- **THEN** tous les changes sont affichés et la page reste utilisable
