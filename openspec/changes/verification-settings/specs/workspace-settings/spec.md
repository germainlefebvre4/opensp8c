# Spec Delta

## MODIFIED Requirements

### Requirement: Sous-onglets de Settings
L'écran Settings SHALL exposer cinq sous-onglets dans la barre de sous-onglets commune des pages du workspace (voir `workspace-page-subtabs`) : « Agent Pool », « Colonnes » (« Columns » en anglais), « Vérification » (« Verification »), « Environnement » (« Environment ») et « Spécialisations » (« Specializations »). L'écran SHALL n'afficher aucun titre de page au-dessus de cette barre, et SHALL afficher le nom du workspace concerné à l'extrémité droite de la barre. Le sous-onglet actif SHALL être reflété dans le paramètre d'URL `tab`, « Agent Pool » étant le sous-onglet par défaut lorsque le paramètre est absent ou inconnu. Le sous-onglet Agent Pool SHALL proposer les mêmes réglages que celui de Configuration, avec la même vocation, mais valables pour ce workspace seulement.

#### Scenario: Affichage des sous-onglets
- **WHEN** l'utilisateur ouvre Settings avec un workspace sélectionné
- **THEN** les sous-onglets Agent Pool, Colonnes, Vérification, Environnement et Spécialisations sont visibles

#### Scenario: Libellés traduits
- **WHEN** la langue de l'application est l'anglais
- **THEN** le sous-onglet « Colonnes » est libellé « Columns », « Vérification » est libellé « Verification » et « Environnement » est libellé « Environment »

#### Scenario: Pas de titre de page, nom du workspace en fin de barre
- **WHEN** l'utilisateur ouvre Settings sur le workspace nommé `A`
- **THEN** aucun titre « Réglages » n'est affiché et le nom du workspace `A` est affiché à l'extrémité droite de la barre de sous-onglets

#### Scenario: Sous-onglet actif dans l'URL
- **WHEN** l'utilisateur clique sur le sous-onglet « Vérification »
- **THEN** l'URL porte `tab=verification` et le rechargement de la page rouvre Settings sur « Vérification »

#### Scenario: Sous-onglet par défaut
- **WHEN** l'utilisateur ouvre Settings sans paramètre `tab`, ou avec une valeur de `tab` inconnue
- **THEN** le sous-onglet « Agent Pool » est actif

### Requirement: Réglage des rôles par workspace dans Colonnes
Le sous-onglet Colonnes de Settings SHALL afficher un réglage global du workspace et un réglage pour chacun des six rôles (`explorer`, `ff`, `implementer`, `fixer`, `verifier`, `documenter`), chacun avec ses champs agent, modèle et effort, selon les règles de `agent-role-settings`. Chaque ligne SHALL indiquer la colonne ou l'étape du Kanban à laquelle son rôle correspond.

#### Scenario: Affichage des lignes de rôle
- **WHEN** l'utilisateur ouvre Settings > Colonnes
- **THEN** une ligne globale et six lignes de rôle sont affichées, chacune avec un sélecteur d'agent, un sélecteur de modèle avec saisie libre et, lorsque l'agent en propose, un sélecteur d'effort

#### Scenario: Enregistrement d'une surcharge de rôle
- **WHEN** l'utilisateur choisit le modèle `haiku` pour le rôle `documenter` et enregistre
- **THEN** la surcharge est persistée dans la section du workspace et s'applique aux prochains lancements de ce workspace
