## MODIFIED Requirements

### Requirement: Afficher la carte d'un changement
Chaque changement SHALL être représenté par une carte épurée affichant : le nom du changement et la progression des tasks (barre de progression + compteur). Les cartes en colonne **Done** SHALL afficher une action rapide **"Sync & Archive"** au survol. Les cartes en colonne **Archived** n'affichent aucune action. Les cartes en colonnes **To Explore**, **To Do**, et **In Progress** SHALL être draggables selon les transitions autorisées. Les cartes en colonnes **Done** et **Archived** SHALL être non-draggables. Quand un subprocess ff est actif pour un changement, sa carte SHALL afficher un spinner à la place du contenu normal et le drag SHALL être désactivé pour cette carte. En cas d'erreur ff (`ff_failed`), la carte SHALL afficher un indicateur d'erreur et le drag est réactivé.

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
