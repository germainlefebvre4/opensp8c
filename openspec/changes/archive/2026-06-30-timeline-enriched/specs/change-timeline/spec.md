## MODIFIED Requirements

### Requirement: Affichage des métadonnées sémantiques sur chaque entrée timeline
Chaque entrée de la timeline SHALL afficher : le nom du change, sa date de création, son statut Kanban, et — si disponibles — son type applicatif et son niveau de complexité. Les specs touchées par le change (delta specs) SHALL être affichées sous le nom du change sous forme de chips cliquables distincts. Les composants sémantiques issus des tags qui n'ont pas de spec formelle correspondante SHALL être affichés comme chips secondaires non-cliquables.

#### Scenario: Entrée avec delta specs
- **WHEN** un change a des delta specs (fichiers dans son dossier `specs/`)
- **THEN** chaque spec est affichée sous forme de chip cliquable ; un clic navigue vers `/specs?workspace=<id>&selected=<spec-name>`

#### Scenario: Entrée avec tags et delta specs
- **WHEN** un change a à la fois des delta specs et des `tags.components`
- **THEN** les delta specs sont affichées comme chips primaires ; seuls les composants de tags non couverts par une spec formelle sont affichés en chips secondaires (style atténué)

#### Scenario: Entrée avec tags uniquement (pas de delta specs)
- **WHEN** un change a des `tags.components` mais aucune delta spec
- **THEN** les composants de tags sont affichés comme chips secondaires

#### Scenario: Entrée sans tags ni delta specs
- **WHEN** un change ne possède ni `tags` ni delta specs
- **THEN** l'entrée s'affiche sans chips, sans erreur

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

### Requirement: Filtrage de la timeline par spec et par tags
L'utilisateur SHALL pouvoir filtrer les entrées de la timeline par spec (depuis la heatmap ou en saisissant un nom) et par tag sémantique (type applicatif, composant LLM). Les deux types de filtres sont cumulables. Les filtres actifs sont affichés comme chips supprimables.

#### Scenario: Filtre par spec
- **WHEN** l'utilisateur sélectionne une spec comme filtre
- **THEN** seuls les changes ayant cette spec dans leurs delta specs sont affichés

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

## ADDED Requirements

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
