# Spec Delta

## MODIFIED Requirements

### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher les changements OpenSpec répartis en six slots horizontaux d'égale largeur : **To Explore**, **Ready**, **To Do**, **In Progress**, **To Review**, et **Done/Archived**. Le slot **Done/Archived** contient verticalement la colonne Done (en haut, `flex-1 min-h-0`, prioritaire sur l'espace vertical) et la colonne Archived (en bas, hauteur plafonnée à 40 % du slot via `max-h`, avec scroll interne). Les colonnes actives lisent depuis `openspec/changes/` (hors `archive/`). La colonne Archived lit depuis `openspec/changes/archive/` via un endpoint dédié. L'endpoint `/changes` SHALL inclure les champs `days_since_activity` (int), `is_stale` (bool), et `tags` (objet optionnel `{ type, complexity, components[] }`) pour chaque change actif.

#### Scenario: Chargement du Kanban avec changements archivés
- **WHEN** l'utilisateur ouvre le Kanban Board ou change de workspace actif
- **THEN** l'application charge les changements actifs depuis `/workspaces/{id}/changes` et les changements archivés depuis `/workspaces/{id}/archived-changes`, et les affiche dans leurs colonnes respectives — Done en haut du slot (priorité flex), Archived en bas (hauteur plafonnée), et To Review entre In Progress et Done/Archived

#### Scenario: Réponse API avec champs stale et tags
- **WHEN** l'endpoint `/workspaces/{id}/changes` retourne la liste des changes
- **THEN** chaque change inclut `days_since_activity` (entier, -1 si pas de tasks.md), `is_stale` (booléen), et `tags` (objet optionnel ou `null`)

#### Scenario: Rétrécissement vertical — Done conserve son espace
- **WHEN** la hauteur disponible du Kanban diminue (bottom panel ouvert, fenêtre réduite)
- **THEN** la colonne Archived se compresse en premier (dans la limite de son plafond), Done conserve l'espace résiduel disponible

#### Scenario: Chargement du Kanban sans changements archivés
- **WHEN** le répertoire `openspec/changes/archive/` est vide ou absent
- **THEN** la colonne Archived est affichée vide en bas du slot Done, sans erreur

#### Scenario: Changement sans tasks.md
- **WHEN** un changement n'a pas de fichier `tasks.md` (ou tasks_total == 0)
- **THEN** il est affiché dans la colonne **To Explore**

#### Scenario: Changement avec tasks non démarrées et non lancé
- **WHEN** un changement a un `tasks.md` avec des tasks mais aucune cochée (`tasks_done == 0`), et n'est pas marqué "lancé" (voir `kanban-ready-column`)
- **THEN** il est affiché dans la colonne **Ready**

#### Scenario: Changement avec tasks non démarrées et lancé
- **WHEN** un changement a un `tasks.md` avec des tasks mais aucune cochée (`tasks_done == 0`), et est marqué "lancé"
- **THEN** il est affiché dans la colonne **To Do**

#### Scenario: Changement partiellement complété
- **WHEN** un changement a au moins une task cochée mais pas toutes
- **THEN** il est affiché dans la colonne **In Progress**

#### Scenario: Changement entièrement complété en mode Autonomie Totale
- **WHEN** toutes les tasks d'un changement sont cochées ET que le pool d'agents s'exécute en mode `full-autonomy` (ou hors exécution du pool)
- **THEN** il est affiché dans la colonne **Done**

#### Scenario: Changement entièrement complété en mode Review Humaine
- **WHEN** toutes les tasks d'un changement ont été complétées par un worker du pool d'agents exécuté en mode `hitl-review`
- **THEN** il est affiché dans la colonne **To Review** et n'est déplacé vers **Done** qu'après approbation de l'utilisateur

### Requirement: Afficher la carte d'un changement
Chaque changement SHALL être représenté par une carte épurée affichant : le nom du changement, la progression des tasks (barre de progression + compteur), et — lorsque les tags sont disponibles — un badge de type applicatif et un indicateur de complexité (points sur 5). Les cartes en colonne **Done** SHALL afficher une action rapide **"Sync & Archive"** au survol. Les cartes en colonne **Archived** n'affichent aucune action. Les cartes en colonnes **To Explore**, **Ready**, **To Do**, et **In Progress** SHALL être draggables selon les transitions autorisées. Les cartes en colonnes **Done** et **Archived** SHALL être non-draggables. Quand un subprocess ff est actif pour un changement, sa carte SHALL afficher un spinner à la place du contenu normal et le drag SHALL être désactivé pour cette carte. En cas d'erreur ff (`ff_failed`), la carte SHALL afficher un indicateur d'erreur et le drag est réactivé. Si un changement dans la colonne **Ready** ou **To Do** posséde une exploration fantôme active du même nom, il SHALL être affiché comme un change brouillon (draft) visuellement distinct, incluant un bouton d'action explicite "Figer". Lorsque `worker_active` vaut `true` pour un changement, sa carte SHALL afficher un badge indiquant qu'un worker du pool d'agents est actuellement assigné à ce changement, positionné sur la même ligne que le compteur de tâches, aux côtés du badge stale s'il est également présent.

#### Scenario: Carte sans tasks.md
- **WHEN** le changement n'a pas encore de fichier `tasks.md`
- **THEN** la progression est affichée comme "0 / 0 tasks" sans erreur

#### Scenario: Carte avec tasks.md
- **WHEN** le changement a un fichier `tasks.md` contenant des items `- [ ]` et `- [x]`
- **THEN** la carte affiche "N / M tasks" où N est le nombre de `[x]` et M le total

#### Scenario: Carte avec tags disponibles
- **WHEN** le changement possède une section `tags` avec `type` et `complexity`
- **THEN** la carte affiche un badge compact pour le type (ex. "🖥 frontend") et un indicateur de complexité (ex. "●●○○○") entre le nom et la barre de progression

#### Scenario: Carte sans tags
- **WHEN** le changement ne possède pas de section `tags`
- **THEN** la carte s'affiche normalement sans badge ni indicateur de complexité, sans erreur

#### Scenario: Survol d'une carte Done
- **WHEN** l'utilisateur survole une carte dans la colonne Done
- **THEN** un bouton "Sync & Archive" apparaît sur la carte

#### Scenario: Carte en colonne Archived
- **WHEN** la carte est dans la colonne Archived
- **THEN** aucun bouton d'action n'est visible, au survol ou autrement

#### Scenario: Carte avec ff actif
- **WHEN** l'événement SSE `ff_started` est reçu pour un changement
- **THEN** la carte de ce changement affiche uniquement un spinner, sans nom ni progression, et le drag est désactivé

#### Scenario: Carte avec ff échoué
- **WHEN** l'événement SSE `ff_failed` est reçu pour un changement
- **THEN** la carte affiche un indicateur d'erreur (icône ou texte "ff échoué") et le drag est réactivé

#### Scenario: Carte Done non-draggable
- **WHEN** l'utilisateur tente de drag une carte en colonne Done
- **THEN** la carte ne peut pas être saisie (drag désactivé sur cette carte)

#### Scenario: Carte Archived non-draggable
- **WHEN** l'utilisateur tente de drag une carte en colonne Archived
- **THEN** la carte ne peut pas être saisie (drag désactivé sur cette carte)

#### Scenario: Change "brouillon" (unsolidified draft) dans la colonne Ready ou To Do
- **WHEN** un changement est dans la colonne "Ready" ou "To Do" ET qu'un ghost card actif possède le même nom dans la liste des changements
- **THEN** la carte s'affiche avec une opacité légèrement atténuée, une bordure pointillée, un badge violet "projet" et un bouton explicite "Figer" qui permet de consolider le change

#### Scenario: Carte avec un worker du pool d'agents actif
- **WHEN** un changement a `worker_active = true`
- **THEN** sa carte affiche un badge signalant qu'un worker du pool d'agents y est actuellement assigné, visible que la carte affiche ou non un badge stale

#### Scenario: Carte sans worker actif
- **WHEN** un changement a `worker_active = false` ou absent
- **THEN** aucun badge de worker actif n'est affiché sur la carte
