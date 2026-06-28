## ADDED Requirements

### Requirement: Ajouter un workspace
L'utilisateur SHALL pouvoir ajouter un workspace en fournissant le chemin absolu d'un répertoire contenant un sous-dossier `openspec/`. L'application SHALL proposer un explorateur de fichiers système pour sélectionner le répertoire. Le workspace est persisté dans `config.yaml` à la racine de l'application.

#### Scenario: Ajout d'un workspace valide
- **WHEN** l'utilisateur sélectionne un répertoire contenant un dossier `openspec/`
- **THEN** le workspace est ajouté à la liste, persisté dans `config.yaml`, et sélectionné comme workspace actif

#### Scenario: Ajout d'un répertoire sans openspec/
- **WHEN** l'utilisateur sélectionne un répertoire ne contenant pas de dossier `openspec/`
- **THEN** l'application affiche un message d'erreur et n'ajoute pas le workspace

#### Scenario: Ajout d'un workspace déjà existant
- **WHEN** l'utilisateur tente d'ajouter un répertoire déjà présent dans `config.yaml`
- **THEN** l'application affiche un message indiquant que le workspace existe déjà et ne crée pas de doublon

### Requirement: Sélectionner le workspace actif
L'application SHALL afficher la liste des workspaces configurés et permettre à l'utilisateur de basculer entre eux. Le workspace actif détermine les changements et specs affichés dans le Kanban et dans la vue Specs.

#### Scenario: Changement de workspace actif
- **WHEN** l'utilisateur sélectionne un workspace différent dans la liste
- **THEN** le Kanban et la vue Specs sont rechargés avec les données du nouveau workspace actif

#### Scenario: Aucun workspace configuré
- **WHEN** aucun workspace n'est présent dans `config.yaml` au démarrage
- **THEN** l'application affiche un écran d'accueil invitant l'utilisateur à ajouter son premier workspace

### Requirement: Supprimer un workspace
L'utilisateur SHALL pouvoir supprimer un workspace de la liste. La suppression retire uniquement l'entrée dans `config.yaml` ; elle ne modifie pas le répertoire du projet.

#### Scenario: Suppression d'un workspace
- **WHEN** l'utilisateur supprime un workspace de la liste
- **THEN** le workspace est retiré de `config.yaml` et de l'interface ; si c'était le workspace actif, l'application sélectionne le premier workspace restant ou affiche l'écran d'accueil si la liste est vide
