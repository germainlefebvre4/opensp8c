## ADDED Requirements

### Requirement: Lister les spécifications actées
L'application SHALL afficher la liste de toutes les spécifications actées du workspace actif, lues depuis `openspec/specs/`. Chaque spec est représentée par son nom (nom du répertoire) et le chemin vers son fichier `spec.md`.

#### Scenario: Workspace avec des specs
- **WHEN** l'utilisateur ouvre la vue Specs d'un workspace contenant des entrées dans `openspec/specs/`
- **THEN** la liste affiche toutes les specs trouvées, triées alphabétiquement par nom

#### Scenario: Workspace sans specs
- **WHEN** le dossier `openspec/specs/` est vide ou absent
- **THEN** la vue affiche un message "Aucune spécification actée pour ce workspace"

### Requirement: Consulter le contenu d'une spécification
L'utilisateur SHALL pouvoir sélectionner une spec dans la liste pour en afficher le contenu Markdown rendu.

#### Scenario: Affichage du contenu d'une spec
- **WHEN** l'utilisateur clique sur une spec dans la liste
- **THEN** le contenu de son fichier `spec.md` est affiché en Markdown rendu dans un panneau de détail
