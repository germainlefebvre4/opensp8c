# Spec Delta

## MODIFIED Requirements

### Requirement: Sélectionner le workspace actif
L'application SHALL afficher la liste des workspaces configurés avec leurs compteurs Kanban et permettre à l'utilisateur de basculer entre eux. Le workspace actif détermine les changements et specs affichés dans le Kanban et dans la vue Specs.

#### Scenario: Changement de workspace actif
- **WHEN** l'utilisateur sélectionne un workspace différent dans la liste
- **THEN** le Kanban et la vue Specs sont rechargés avec les données du nouveau workspace actif

#### Scenario: Aucun workspace configuré
- **WHEN** aucun workspace n'est présent dans `config.yaml` au démarrage
- **THEN** la structure globale de l'application (barre de navigation, sidebar workspace) reste affichée
- **THEN** la zone de contenu principale affiche une invitation à ajouter un premier projet, à la place du Kanban
- **THEN** les pages non liées à un workspace (Agents, Réglages, Configuration) restent pleinement accessibles depuis la navigation
