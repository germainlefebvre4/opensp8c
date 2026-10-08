# Spec Delta

## MODIFIED Requirements

### Requirement: Pastille Done dans le menu latéral
Le menu latéral SHALL afficher, pour chaque workspace, le nombre de changes au statut "done" (terminés, non archivés) comme segment `done` de la barre segmentée du projet (couleur emerald, `bg-emerald-500`) lorsque ce compteur est supérieur à zéro, ainsi que dans le total de la barre.

#### Scenario: Workspace avec des changes done
- **WHEN** un workspace a un ou plusieurs changes au statut "done"
- **THEN** la barre segmentée du projet contient un segment emerald proportionnel à ce compteur, avec une infobulle indiquant le nombre

#### Scenario: Workspace sans change done
- **WHEN** un workspace n'a aucun change au statut "done"
- **THEN** aucun segment done n'est affiché

#### Scenario: Pastilles visibles au survol
- **WHEN** l'utilisateur survole un item workspace dans le menu latéral
- **THEN** tous les segments de la barre (to-explore, ready, todo, in-progress, verifying, to-review, done) restent visibles

#### Scenario: Cohérence couleur avec le Kanban
- **WHEN** le segment done est affiché dans le menu latéral
- **THEN** sa couleur est `bg-emerald-500`, identique au dot de la colonne Done dans le Kanban
