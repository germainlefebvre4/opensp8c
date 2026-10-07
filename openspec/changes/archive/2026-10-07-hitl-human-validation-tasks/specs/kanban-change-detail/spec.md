# Spec Delta

## ADDED Requirements

### Requirement: Signalement des tâches de validation humaine dans le DetailPanel
Dans la liste des tâches du `DetailPanel`, une tâche dont `human_review` est vrai SHALL porter un badge « Validation humaine » (« Human validation » en anglais), visible qu'elle soit cochée ou non, et son texte SHALL être affiché sans le commentaire du marqueur. Le checkbox d'une tâche marquée SHALL rester interactif dans les conditions où celui des autres tâches l'est. Pour un change en To Review, l'onglet Tâches SHALL aussi afficher, au-dessus de la liste, le nombre de tâches restant à valider lorsqu'il est supérieur à zéro. Les libellés SHALL être disponibles en français et en anglais.

#### Scenario: Tâche marquée non cochée
- **WHEN** le DetailPanel affiche une tâche `4.2 Parcours manuel` avec `human_review` à vrai et `done` à faux
- **THEN** la tâche porte le badge « Validation humaine », son texte ne montre pas le commentaire du marqueur et son checkbox est décoché et actif

#### Scenario: Tâche marquée cochée
- **WHEN** l'utilisateur coche cette tâche
- **THEN** le badge reste affiché et la tâche apparaît comme faite

#### Scenario: Compteur de tâches à valider en revue
- **WHEN** l'utilisateur ouvre l'onglet Tâches d'un change en To Review qui a 2 tâches non cochées
- **THEN** un message indique « 2 tâches à valider » au-dessus de la liste

#### Scenario: Aucune tâche restante
- **WHEN** toutes les tâches d'un change en To Review sont cochées
- **THEN** aucun message de tâches à valider n'est affiché
