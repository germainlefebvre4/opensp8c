# Spec Delta

## MODIFIED Requirements

### Requirement: Affichage des métadonnées sémantiques sur chaque entrée timeline
Chaque entrée de la timeline SHALL afficher : le nom du change, sa date de création, son statut Kanban, et — si disponibles — un badge par valeur du tableau `tags.type` et son niveau de complexité. Les specs touchées par le change (delta specs) SHALL être affichées sous le nom du change sous forme de chips cliquables distincts. Les composants sémantiques issus des tags qui n'ont pas de spec formelle correspondante SHALL être affichés comme chips secondaires non-cliquables.

#### Scenario: Entrée avec delta specs
- **WHEN** un change a des delta specs (fichiers dans son dossier `specs/`)
- **THEN** chaque spec est affichée sous forme de chip cliquable ; un clic navigue vers `/specs?workspace=<id>&selected=<spec-name>`

#### Scenario: Entrée avec tags et delta specs
- **WHEN** un change a à la fois des delta specs et des `tags.components`
- **THEN** les delta specs sont affichées comme chips primaires ; seuls les composants de tags non couverts par une spec formelle sont affichés en chips secondaires (style atténué)

#### Scenario: Entrée avec tags uniquement (pas de delta specs)
- **WHEN** un change a des `tags.components` mais aucune delta spec
- **THEN** les composants de tags sont affichés comme chips secondaires

#### Scenario: Entrée avec plusieurs valeurs de type
- **WHEN** un change a `tags.type: [frontend, backend]`
- **THEN** l'entrée affiche deux badges type distincts, chacun cliquable individuellement pour filtrer

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
