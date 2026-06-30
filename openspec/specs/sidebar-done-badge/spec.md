## Purpose

Afficher une pastille de comptage visuelle dans le menu latéral pour les changes au statut "done" (terminés, non archivés) par workspace.

## Requirements

### Requirement: Pastille Done dans le menu latéral
Le menu latéral SHALL afficher une pastille emerald (`bg-emerald-500`) indiquant le nombre de changes au statut "done" (terminés, non archivés) pour chaque workspace, lorsque ce compteur est supérieur à zéro.

#### Scenario: Workspace avec des changes done
- **WHEN** un workspace a un ou plusieurs changes au statut "done"
- **THEN** une pastille emerald affiche le compteur correspondant dans le menu latéral

#### Scenario: Workspace sans change done
- **WHEN** un workspace n'a aucun change au statut "done"
- **THEN** aucune pastille done n'est affichée

#### Scenario: Pastilles visibles au survol
- **WHEN** l'utilisateur survole un item workspace dans le menu latéral
- **THEN** toutes les pastilles de comptage (to-explore, todo, in-progress, done) restent visibles

#### Scenario: Cohérence couleur avec le Kanban
- **WHEN** la pastille done est affichée dans le menu latéral
- **THEN** sa couleur est `bg-emerald-500`, identique au dot de la colonne Done dans le Kanban
