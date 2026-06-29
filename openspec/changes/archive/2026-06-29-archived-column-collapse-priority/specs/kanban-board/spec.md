## MODIFIED Requirements

### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher les changements OpenSpec répartis en quatre slots horizontaux d'égale largeur : **To Explore**, **To Do**, **In Progress**, et **Done/Archived**. Le slot **Done/Archived** contient verticalement la colonne Done (en haut, `flex-1 min-h-0`, prioritaire sur l'espace vertical) et la colonne Archived (en bas, hauteur plafonnée à 40 % du slot via `max-h`, avec scroll interne). Les colonnes actives lisent depuis `openspec/changes/` (hors `archive/`). La colonne Archived lit depuis `openspec/changes/archive/` via un endpoint dédié. La règle de calcul du statut est inchangée pour les colonnes actives.

#### Scenario: Chargement du Kanban avec changements archivés
- **WHEN** l'utilisateur ouvre le Kanban Board ou change de workspace actif
- **THEN** l'application charge les changements actifs depuis `/workspaces/{id}/changes` et les changements archivés depuis `/workspaces/{id}/archived-changes`, et les affiche dans leurs colonnes respectives — Done en haut du slot (priorité flex), Archived en bas (hauteur plafonnée)

#### Scenario: Rétrécissement vertical — Done conserve son espace
- **WHEN** la hauteur disponible du Kanban diminue (bottom panel ouvert, fenêtre réduite)
- **THEN** la colonne Archived se compresse en premier (dans la limite de son plafond), Done conserve l'espace résiduel disponible

#### Scenario: Chargement du Kanban sans changements archivés
- **WHEN** le répertoire `openspec/changes/archive/` est vide ou absent
- **THEN** la colonne Archived est affichée vide en bas du slot Done, sans erreur

### Requirement: Colonne Archived — affichage paginé, lecture seule et collapsible
La colonne **Archived** SHALL afficher les changements par ordre antéchronologique (les plus récents en premier), limités à **3** par défaut (au lieu de 5). Un bouton "Afficher plus" SHALL permettre d'en charger 3 supplémentaires à chaque clic. La colonne Archived SHALL exposer un bouton **collapse/expand** (chevron) dans son header, permettant de masquer entièrement la liste de cartes tout en conservant le header visible. L'état collapsed est local et non persisté. Les cartes de la colonne Archived SHALL avoir un traitement visuel atténué (teintes slate/grises). Aucune action n'est disponible sur les cartes archivées — elles sont en lecture seule.

#### Scenario: Colonne Archived avec plus de 3 changements
- **WHEN** le répertoire archive contient plus de 3 changements
- **THEN** les 3 plus récents sont affichés et un bouton "Afficher plus" est visible en bas de la colonne

#### Scenario: Clic sur "Afficher plus"
- **WHEN** l'utilisateur clique sur "Afficher plus"
- **THEN** 3 changements supplémentaires s'ajoutent à l'affichage, et le bouton disparaît si tous les changements sont désormais visibles

#### Scenario: Colonne Archived avec 3 changements ou moins
- **WHEN** le répertoire archive contient 3 changements ou moins
- **THEN** tous sont affichés, sans bouton "Afficher plus"

#### Scenario: Collapse de la colonne Archived
- **WHEN** l'utilisateur clique sur le chevron collapse dans le header de la colonne Archived
- **THEN** la liste de cartes et le bouton "Afficher plus" sont masqués, seul le header reste visible, le chevron indique l'état collapsed

#### Scenario: Expand de la colonne Archived
- **WHEN** l'utilisateur clique sur le chevron expand dans le header de la colonne Archived (état collapsed)
- **THEN** la liste de cartes réapparaît avec le nombre de cartes visibles précédent

#### Scenario: Clic sur une carte archivée
- **WHEN** l'utilisateur clique sur une carte dans la colonne Archived
- **THEN** le DetailPanel s'ouvre en lecture seule, sans bouton d'action d'archivage
