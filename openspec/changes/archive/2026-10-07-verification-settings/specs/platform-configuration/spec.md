# Spec Delta

## MODIFIED Requirements

### Requirement: Sous-onglet Colonnes dans Configuration
Le système SHALL exposer, dans Configuration, un sous-onglet « Colonnes » (« Columns » en anglais) hébergeant le réglage global de l'agent, du modèle et de l'effort, ainsi que le réglage de chacun des six rôles décrits par la capacité `agent-role-settings`. Les valeurs préréglées SHALL être affichées comme valeurs par défaut lorsque rien n'est enregistré, et l'interface SHALL indiquer, pour chaque rôle, la colonne ou l'étape du Kanban correspondante.

#### Scenario: Affichage du sous-onglet
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** un sous-onglet « Colonnes » est visible aux côtés des autres sous-onglets

#### Scenario: Contenu du sous-onglet
- **WHEN** l'utilisateur active le sous-onglet « Colonnes »
- **THEN** une ligne de réglage global et six lignes de rôle (explorer, ff, implementer, fixer, verifier, documenter) sont affichées, avec les champs agent, modèle avec saisie libre et effort

#### Scenario: Préréglages affichés
- **WHEN** aucun réglage de rôle n'est enregistré et que l'agent par défaut est Claude
- **THEN** le rôle `explorer` affiche `opus` et l'effort `high` comme valeurs par défaut
- **THEN** le rôle `verifier` affiche `sonnet` et l'effort `medium` comme valeurs par défaut
- **THEN** le rôle `documenter` affiche `haiku` et l'effort `low` comme valeurs par défaut

#### Scenario: Effort masqué
- **WHEN** l'utilisateur sélectionne pour un rôle un agent qui ne déclare aucun niveau d'effort
- **THEN** le champ effort de ce rôle n'est pas proposé
