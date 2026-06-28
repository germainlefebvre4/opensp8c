## MODIFIED Requirements

### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher tous les changements OpenSpec du workspace actif (`openspec/changes/`) répartis en quatre colonnes : **To Explore**, **To Do**, **In Progress**, **Done**. La colonne d'un changement est déterminée automatiquement par sa progression en tasks, calculée depuis `tasks.md`. Le champ `kanban_status` dans `.openspec.yaml` est ignoré.

La règle de calcul est :
- `tasks_total == 0` → **To Explore**
- `tasks_done == 0` et `tasks_total > 0` → **To Do**
- `0 < tasks_done < tasks_total` → **In Progress**
- `tasks_done == tasks_total > 0` → **Done**

#### Scenario: Chargement du Kanban
- **WHEN** l'utilisateur ouvre le Kanban Board ou change de workspace actif
- **THEN** l'application lit tous les répertoires dans `openspec/changes/` (hors `archive/`), calcule la colonne de chaque changement depuis sa progression en tasks, et place chaque changement dans la colonne correspondante

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

## REMOVED Requirements

### Requirement: Déplacer une carte par drag & drop
**Reason**: L'app est read-only. La colonne est dérivée automatiquement depuis les tasks — le drag & drop n'a plus de sémantique utile et crée de la confusion.
**Migration**: Les transitions de colonne se font naturellement en cochant des tasks via le terminal/CLI OpenSpec.

### Requirement: Changer le statut depuis la carte
**Reason**: Le bouton "→ To Do" appelait `PUT /status` pour écrire `kanban_status`. Ce champ n'est plus la source de vérité.
**Migration**: Aucune action manuelle requise — la colonne se met à jour automatiquement au rafraîchissement suivant.
