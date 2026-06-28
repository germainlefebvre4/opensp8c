## ADDED Requirements

### Requirement: Afficher les changements en colonnes Kanban
Le Kanban Board SHALL afficher tous les changements OpenSpec du workspace actif (`openspec/changes/`) répartis en quatre colonnes : **To Explore**, **To Do**, **In Progress**, **Done**. La colonne d'un changement est déterminée par le champ `kanban_status` dans son fichier `.openspec.yaml`. Si ce champ est absent, le changement est affiché dans **To Explore** par défaut.

#### Scenario: Chargement du Kanban
- **WHEN** l'utilisateur ouvre le Kanban Board ou change de workspace actif
- **THEN** l'application lit tous les répertoires dans `openspec/changes/` (hors `archive/`), lit leur `.openspec.yaml`, et place chaque changement dans la colonne correspondant à son `kanban_status`

#### Scenario: Changement sans kanban_status
- **WHEN** un changement n'a pas de champ `kanban_status` dans son `.openspec.yaml`
- **THEN** il est affiché dans la colonne **To Explore**

### Requirement: Afficher la carte d'un changement
Chaque changement SHALL être représenté par une carte affichant : le nom du changement, la progression des tasks (nombre de `[x]` sur le total dans `tasks.md`), et les actions disponibles selon la colonne.

#### Scenario: Carte sans tasks.md
- **WHEN** le changement n'a pas encore de fichier `tasks.md`
- **THEN** la progression est affichée comme "0 / 0 tasks" sans erreur

#### Scenario: Carte avec tasks.md
- **WHEN** le changement a un fichier `tasks.md` contenant des items `- [ ]` et `- [x]`
- **THEN** la carte affiche "N / M tasks" où N est le nombre de `[x]` et M le total

### Requirement: Déplacer une carte par drag & drop
L'utilisateur SHALL pouvoir déplacer une carte d'une colonne à une autre par glisser-déposer. L'application SHALL mettre à jour le champ `kanban_status` dans le `.openspec.yaml` du changement correspondant.

#### Scenario: Déplacement valide
- **WHEN** l'utilisateur dépose une carte dans une colonne différente
- **THEN** le `kanban_status` du changement est mis à jour dans `.openspec.yaml` et la carte apparaît dans la nouvelle colonne

### Requirement: Changer le statut depuis la carte
L'utilisateur SHALL pouvoir passer un changement de la colonne **To Explore** à la colonne **To Do** via un bouton dédié dans le détail de la carte, en complément du drag & drop.

#### Scenario: Bouton "Passer en To Do"
- **WHEN** l'utilisateur clique sur le bouton de transition dans une carte de la colonne **To Explore**
- **THEN** le `kanban_status` est mis à jour en `todo` et la carte est déplacée dans la colonne **To Do**

### Requirement: Rafraîchissement automatique du Kanban
Le Kanban SHALL se rafraîchir automatiquement pour refléter les changements apportés aux fichiers OpenSpec par des outils externes (Claude Code, openspec CLI).

#### Scenario: Rafraîchissement périodique
- **WHEN** 5 secondes se sont écoulées depuis le dernier chargement
- **THEN** l'application relit les changements du workspace actif et met à jour l'affichage sans perte de l'état UI (cartes ouvertes, position de scroll)
