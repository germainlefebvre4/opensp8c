# Spec Delta

## ADDED Requirements

### Requirement: Défauts de l'Agent Pool dans Configuration
Le système SHALL exposer, dans Configuration > Agent Pool, au-dessus de la vue de visibilité des pools actifs, des réglages par défaut du pool d'agents : taille (de 1 à 5), mode de délégation (`full-autonomy` ou `hitl-review`) et nombre maximal de tentatives d'auto-correction. Ces valeurs SHALL être persistées dans `preferences.json`, retournées par `GET /api/preferences` et servir de base à tous les workspaces, qui peuvent les surcharger dans Settings > Agent Pool. À défaut de réglage enregistré, la taille vaut 3, le mode `hitl-review` et le nombre de tentatives 3.

#### Scenario: Valeurs par défaut initiales
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool sans réglage de pool enregistré
- **THEN** les défauts affichés sont une taille de 3, le mode `hitl-review` et 3 tentatives

#### Scenario: Enregistrement des défauts
- **WHEN** l'utilisateur fixe la taille à 2 et enregistre
- **THEN** `preferences.json` reflète ce défaut et `GET /api/preferences` le retourne
- **THEN** le pool démarré pour un workspace sans surcharge a une taille de 2

#### Scenario: Vue de visibilité conservée
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool
- **THEN** la vue de visibilité de tous les pools actifs, tous workspaces confondus, reste affichée sous les défauts

#### Scenario: Valeur invalide
- **WHEN** l'utilisateur tente d'enregistrer une taille de 0
- **THEN** l'enregistrement est refusé avec une erreur de validation et les défauts enregistrés ne sont pas modifiés

### Requirement: Sous-onglet Colonnes dans Configuration
Le système SHALL exposer, dans Configuration, un sous-onglet « Colonnes » (« Columns » en anglais) hébergeant le réglage global de l'agent, du modèle et de l'effort, ainsi que le réglage de chacun des cinq rôles décrits par la capacité `agent-role-settings`. Les valeurs préréglées SHALL être affichées comme valeurs par défaut lorsque rien n'est enregistré, et l'interface SHALL indiquer, pour chaque rôle, la colonne ou l'étape du Kanban correspondante.

#### Scenario: Affichage du sous-onglet
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** un sous-onglet « Colonnes » est visible aux côtés des autres sous-onglets

#### Scenario: Contenu du sous-onglet
- **WHEN** l'utilisateur active le sous-onglet « Colonnes »
- **THEN** une ligne de réglage global et cinq lignes de rôle (explorer, ff, implementer, fixer, documenter) sont affichées, avec les champs agent, modèle avec saisie libre et effort

#### Scenario: Préréglages affichés
- **WHEN** aucun réglage de rôle n'est enregistré et que l'agent par défaut est Claude
- **THEN** le rôle `explorer` affiche `opus` et l'effort `high` comme valeurs par défaut
- **THEN** le rôle `documenter` affiche `haiku` et l'effort `low` comme valeurs par défaut

#### Scenario: Effort masqué
- **WHEN** l'utilisateur sélectionne pour un rôle un agent qui ne déclare aucun niveau d'effort
- **THEN** le champ effort de ce rôle n'est pas proposé
