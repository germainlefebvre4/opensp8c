# Spec Delta

## MODIFIED Requirements

### Requirement: Sélectionner le workspace actif
L'application SHALL afficher la liste des workspaces configurés avec leurs compteurs Kanban et permettre à l'utilisateur de basculer entre eux, à la souris comme au clavier. Le workspace actif détermine les changements et specs affichés dans le Kanban et dans la vue Specs, ainsi que les surcharges affichées dans Settings.

#### Scenario: Changement de workspace actif
- **WHEN** l'utilisateur sélectionne un workspace différent dans la liste
- **THEN** le Kanban et la vue Specs sont rechargés avec les données du nouveau workspace actif
- **THEN** Settings affiche les surcharges du nouveau workspace actif

#### Scenario: Sélection au clavier
- **WHEN** l'utilisateur place le focus sur la ligne d'un workspace et active la touche Entrée ou Espace
- **THEN** ce workspace devient le workspace actif

#### Scenario: Workspace actif signalé
- **WHEN** un workspace est actif
- **THEN** sa ligne est exposée aux technologies d'assistance comme l'élément courant de la liste

#### Scenario: Aucun workspace configuré
- **WHEN** aucun workspace n'est présent dans `config.yaml` au démarrage
- **THEN** la structure globale de l'application (barre de navigation, sidebar workspace) reste affichée
- **THEN** la zone de contenu principale affiche une invitation à ajouter un premier projet, à la place du Kanban
- **THEN** Configuration, qui n'est pas liée à un workspace, reste pleinement accessible depuis la navigation
- **THEN** Settings affiche l'état vide « aucun workspace », comme les onglets Kanban, Specs, Timeline et Agents

### Requirement: Supprimer un workspace
L'utilisateur SHALL pouvoir supprimer un workspace de la liste depuis un menu d'actions (`…`) de sa ligne, toujours visible et utilisable au clavier. La suppression SHALL exiger une confirmation qui nomme le projet et précise que le dossier du projet n'est pas supprimé. La suppression retire uniquement l'entrée dans `config.yaml` ; elle ne modifie pas le répertoire du projet.

#### Scenario: Suppression d'un workspace
- **WHEN** l'utilisateur ouvre le menu d'actions d'un workspace, choisit « Retirer du suivi » et confirme
- **THEN** le workspace est retiré de `config.yaml` et de l'interface ; si c'était le workspace actif, l'application sélectionne le premier workspace restant ou affiche l'écran d'accueil si la liste est vide

#### Scenario: Suppression annulée
- **WHEN** l'utilisateur ouvre la confirmation puis l'annule
- **THEN** le workspace reste dans la liste et dans `config.yaml`

#### Scenario: Menu accessible sans survol
- **WHEN** l'utilisateur navigue au clavier dans la liste des workspaces
- **THEN** le menu d'actions de chaque ligne est atteignable sans survol à la souris

#### Scenario: Confirmation explicite
- **WHEN** la confirmation de suppression est affichée
- **THEN** elle nomme le workspace et indique que son dossier n'est pas supprimé

## ADDED Requirements

### Requirement: Erreurs d'ajout affichées dans la sidebar
Lorsque l'ajout d'un workspace échoue (répertoire sans dossier `openspec/`, workspace déjà configuré, chemin invalide), la sidebar SHALL afficher le message d'erreur à proximité du formulaire d'ajout, laisser le formulaire ouvert avec la valeur saisie, et effacer l'erreur dès que l'utilisateur modifie le champ ou annule.

#### Scenario: Répertoire sans openspec
- **WHEN** l'utilisateur saisit un chemin sans dossier `openspec/` et valide
- **THEN** un message d'erreur s'affiche dans la sidebar et le formulaire reste ouvert avec le chemin saisi

#### Scenario: Doublon
- **WHEN** l'utilisateur saisit le chemin d'un workspace déjà configuré
- **THEN** un message indique que le workspace existe déjà et aucun doublon n'est créé

#### Scenario: Erreur effacée
- **WHEN** une erreur est affichée et l'utilisateur modifie le champ
- **THEN** le message d'erreur disparaît
