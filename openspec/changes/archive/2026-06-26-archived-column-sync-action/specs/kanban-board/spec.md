## MODIFIED Requirements

### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher tous les changements OpenSpec du workspace actif répartis en cinq colonnes : **To Explore**, **To Do**, **In Progress**, **Done**, et **Archived**. Les quatre premières colonnes lisent depuis `openspec/changes/` (hors `archive/`). La colonne **Archived** lit depuis `openspec/changes/archive/` via un endpoint dédié. Un séparateur visuel (ligne verticale ou espace accentué) sépare les colonnes Done et Archived pour marquer la frontière entre changements actifs et archivés. La règle de calcul du statut est inchangée pour les colonnes actives.

#### Scenario: Chargement du Kanban avec changements archivés
- **WHEN** l'utilisateur ouvre le Kanban Board ou change de workspace actif
- **THEN** l'application charge les changements actifs depuis `/workspaces/{id}/changes` et les changements archivés depuis `/workspaces/{id}/archived-changes`, et les affiche dans leurs colonnes respectives

#### Scenario: Chargement du Kanban sans changements archivés
- **WHEN** le répertoire `openspec/changes/archive/` est vide ou absent
- **THEN** la colonne Archived est affichée vide, sans erreur

#### Scenario: Changement sans tasks.md
- **WHEN** un changement n'a pas de fichier `tasks.md` (ou tasks_total == 0)
- **THEN** il est affiché dans la colonne **To Explore**

#### Scenario: Changement avec tasks non démarrées
- **WHEN** un changement a un `tasks.md` avec des tasks mais aucune cochée (`tasks_done == 0`)
- **THEN** il est affiché dans la colonne **To Do**

#### Scenario: Changement partiellement complété
- **WHEN** un changement a au moins une task cochée mais pas toutes
- **THEN** il est affiché dans la colonne **In Progress**

#### Scenario: Changement entièrement complété
- **WHEN** toutes les tasks d'un changement sont cochées (`tasks_done == tasks_total > 0`)
- **THEN** il est affiché dans la colonne **Done**

### Requirement: Colonne Archived — affichage paginé et lecture seule
La colonne **Archived** SHALL afficher les changements par ordre antéchronologique (les plus récents en premier), limités à 5 par défaut. Un bouton "Afficher plus" SHALL permettre d'en charger 5 supplémentaires à chaque clic. Les cartes de la colonne Archived SHALL avoir un traitement visuel atténué (teintes slate/grises) pour les distinguer visuellement des changements actifs. Aucune action n'est disponible sur les cartes archivées — elles sont en lecture seule.

#### Scenario: Colonne Archived avec plus de 5 changements
- **WHEN** le répertoire archive contient plus de 5 changements
- **THEN** les 5 plus récents sont affichés et un bouton "Afficher plus" est visible en bas de la colonne

#### Scenario: Clic sur "Afficher plus"
- **WHEN** l'utilisateur clique sur "Afficher plus"
- **THEN** 5 changements supplémentaires s'ajoutent à l'affichage, et le bouton disparaît si tous les changements sont désormais visibles

#### Scenario: Colonne Archived avec 5 changements ou moins
- **WHEN** le répertoire archive contient 5 changements ou moins
- **THEN** tous sont affichés, sans bouton "Afficher plus"

#### Scenario: Clic sur une carte archivée
- **WHEN** l'utilisateur clique sur une carte dans la colonne Archived
- **THEN** le DetailPanel s'ouvre en lecture seule, sans bouton d'action d'archivage

### Requirement: Séparateur visuel entre Done et Archived
Un séparateur visuel SHALL être rendu entre les colonnes **Done** et **Archived** pour signaler la frontière entre changements actifs (cycle de vie en cours) et changements archivés (cycle de vie clos).

#### Scenario: Rendu du séparateur
- **WHEN** le Kanban est affiché
- **THEN** un élément visuel distinct (ex. ligne verticale tenue, espace accentué, ou bordure) sépare visuellement la colonne Done de la colonne Archived

### Requirement: Afficher la carte d'un changement
Chaque changement SHALL être représenté par une carte épurée affichant : le nom du changement et la progression des tasks (barre de progression + compteur). Les cartes en colonne **Done** SHALL afficher une action rapide **"Sync & Archive"** au survol. Les cartes en colonne **Archived** n'affichent aucune action.

#### Scenario: Carte sans tasks.md
- **WHEN** le changement n'a pas encore de fichier `tasks.md`
- **THEN** la progression est affichée comme "0 / 0 tasks" sans erreur

#### Scenario: Carte avec tasks.md
- **WHEN** le changement a un fichier `tasks.md` contenant des items `- [ ]` et `- [x]`
- **THEN** la carte affiche "N / M tasks" où N est le nombre de `[x]` et M le total

#### Scenario: Survol d'une carte Done
- **WHEN** l'utilisateur survole une carte dans la colonne Done
- **THEN** un bouton "Sync & Archive" apparaît sur la carte

#### Scenario: Carte en colonne Archived
- **WHEN** la carte est dans la colonne Archived
- **THEN** aucun bouton d'action n'est visible, au survol ou autrement
