### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher les changements OpenSpec répartis en quatre slots horizontaux d'égale largeur : **To Explore**, **To Do**, **In Progress**, et **Done/Archived**. Le slot **Done/Archived** contient verticalement la colonne Done (en haut, prenant l'espace restant) et la colonne Archived (en bas, hauteur auto limitée à ses cartes visibles). Les colonnes actives lisent depuis `openspec/changes/` (hors `archive/`). La colonne Archived lit depuis `openspec/changes/archive/` via un endpoint dédié. La règle de calcul du statut est inchangée pour les colonnes actives.

#### Scenario: Chargement du Kanban avec changements archivés
- **WHEN** l'utilisateur ouvre le Kanban Board ou change de workspace actif
- **THEN** l'application charge les changements actifs depuis `/workspaces/{id}/changes` et les changements archivés depuis `/workspaces/{id}/archived-changes`, et les affiche dans leurs colonnes respectives — Done en haut du slot, Archived en bas

#### Scenario: Chargement du Kanban sans changements archivés
- **WHEN** le répertoire `openspec/changes/archive/` est vide ou absent
- **THEN** la colonne Archived est affichée vide en bas du slot Done, sans erreur

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
Un séparateur visuel horizontal SHALL être rendu entre les colonnes **Done** et **Archived** à l'intérieur du slot partagé, pour signaler la frontière entre changements actifs et archivés.

#### Scenario: Rendu du séparateur horizontal
- **WHEN** le Kanban est affiché
- **THEN** une ligne horizontale tenue sépare visuellement la section Done de la section Archived dans le même slot de colonne

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

### Requirement: Ouvrir l'ExplorePanel au clic sur une carte To Explore
L'utilisateur SHALL pouvoir cliquer sur une carte dans la colonne **To Explore** pour ouvrir le bottom panel de conversation. Le panel SHALL s'afficher sous les colonnes Kanban (layout flex-col), sans masquer ni comprimer les colonnes. La carte entière est la zone cliquable.

#### Scenario: Clic sur carte en colonne To Explore
- **WHEN** l'utilisateur clique sur une carte dans la colonne **To Explore**
- **THEN** le bottom panel de chat s'ouvre sous les colonnes Kanban, les colonnes restant visibles et interactibles au-dessus

#### Scenario: Bottom panel ne masque pas les colonnes
- **WHEN** le bottom panel est ouvert
- **THEN** les colonnes Kanban restent visibles et interactibles dans la partie supérieure de l'écran

### Requirement: Colonnes Kanban pleine hauteur
Les colonnes Kanban SHALL occuper toute la hauteur disponible de la zone de contenu, indépendamment du nombre de cartes qu'elles contiennent.

#### Scenario: Colonne vide
- **WHEN** une colonne ne contient aucune carte
- **THEN** la colonne s'étend sur toute la hauteur disponible et reste une zone de dépôt valide

#### Scenario: Colonnes de hauteurs différentes
- **WHEN** les colonnes contiennent des nombres différents de cartes
- **THEN** toutes les colonnes ont la même hauteur (celle de la colonne la plus haute ou de la zone disponible)

### Requirement: Application pleine largeur avec colonnes auto-adaptées
Le Kanban Board SHALL occuper toute la largeur disponible de la zone de contenu, que le DetailPanel soit ouvert ou non. Lorsque le DetailPanel est ouvert, les colonnes SHALL partager l'espace horizontal avec lui selon un layout flex : colonnes en `flex: 1` et DetailPanel en largeur fixe (`420px`). Le bottom panel d'exploration n'affecte pas la largeur des colonnes. Les colonnes SHALL être scrollables horizontalement si leur largeur minimale combinée dépasse l'espace disponible.

#### Scenario: Redimensionnement de la fenêtre sans panel
- **WHEN** l'utilisateur redimensionne la fenêtre du navigateur et aucun panel n'est ouvert
- **THEN** les colonnes s'adaptent automatiquement pour remplir toute la largeur disponible sans débordement horizontal

#### Scenario: DetailPanel ouvert — colonnes réduites
- **WHEN** le DetailPanel est ouvert
- **THEN** les colonnes occupent l'espace restant après le slot de 420px du DetailPanel, avec un scroll horizontal si nécessaire

#### Scenario: DetailPanel fermé — colonnes pleine largeur
- **WHEN** le DetailPanel est fermé
- **THEN** les colonnes reprennent toute la largeur disponible

#### Scenario: Bottom panel ouvert — largeur colonnes inchangée
- **WHEN** le bottom panel d'exploration est ouvert
- **THEN** les colonnes conservent leur largeur (le bottom panel n'affecte que la hauteur disponible)

### Requirement: Rafraîchissement automatique du Kanban
Le Kanban SHALL se rafraîchir automatiquement pour refléter les changements apportés aux fichiers OpenSpec par des outils externes (Claude Code, openspec CLI). Le rafraîchissement SHALL se faire via les événements SSE du stream `/api/workspaces/{id}/events` — sans polling périodique. À réception d'un événement `change_updated`, le frontend SHALL invalider la liste des changes ET le détail du change concerné. À réception d'un événement `change_created` ou `change_deleted`, le frontend SHALL invalider uniquement la liste des changes. En cas d'indisponibilité du stream SSE, les données affichées restent celles du dernier fetch réussi (pas de fallback polling).

#### Scenario: Mise à jour d'un change via Claude Code
- **WHEN** Claude Code modifie `tasks.md` d'un change et que l'événement `change_updated` est reçu via SSE
- **THEN** le frontend recharge la liste des changes et le détail du change concerné, et le kanban se met à jour immédiatement sans action utilisateur

#### Scenario: Création d'un nouveau change via la CLI
- **WHEN** la CLI crée un nouveau répertoire de change et que l'événement `change_created` est reçu via SSE
- **THEN** le frontend recharge la liste des changes et la nouvelle carte apparaît dans la colonne appropriée

#### Scenario: Archivage d'un change via la CLI
- **WHEN** la CLI archive un change (déplace le répertoire) et que l'événement `change_deleted` est reçu via SSE
- **THEN** le frontend recharge la liste des changes et la carte disparaît des colonnes actives

#### Scenario: Déconnexion SSE — données conservées
- **WHEN** le stream SSE est interrompu (réseau, redémarrage serveur)
- **THEN** les données du kanban restent affichées dans leur dernier état connu, sans message d'erreur intrusif
