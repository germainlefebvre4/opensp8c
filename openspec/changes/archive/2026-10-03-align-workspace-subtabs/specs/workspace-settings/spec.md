# Spec Delta

## MODIFIED Requirements

### Requirement: Sous-onglets de Settings
L'écran Settings SHALL exposer quatre sous-onglets dans la barre de sous-onglets commune des pages du workspace (voir `workspace-page-subtabs`) : « Agent Pool », « Colonnes » (« Columns » en anglais), « Environnement » (« Environment ») et « Spécialisations » (« Specializations »). L'écran SHALL n'afficher aucun titre de page au-dessus de cette barre, et SHALL afficher le nom du workspace concerné à l'extrémité droite de la barre. Le sous-onglet actif SHALL être reflété dans le paramètre d'URL `tab`, « Agent Pool » étant le sous-onglet par défaut lorsque le paramètre est absent ou inconnu. Le sous-onglet Agent Pool SHALL proposer les mêmes réglages que celui de Configuration, avec la même vocation, mais valables pour ce workspace seulement.

#### Scenario: Affichage des sous-onglets
- **WHEN** l'utilisateur ouvre Settings avec un workspace sélectionné
- **THEN** les sous-onglets Agent Pool, Colonnes, Environnement et Spécialisations sont visibles

#### Scenario: Libellés traduits
- **WHEN** la langue de l'application est l'anglais
- **THEN** le sous-onglet « Colonnes » est libellé « Columns » et « Environnement » est libellé « Environment »

#### Scenario: Pas de titre de page, nom du workspace en fin de barre
- **WHEN** l'utilisateur ouvre Settings sur le workspace nommé `A`
- **THEN** aucun titre « Réglages » n'est affiché et le nom du workspace `A` est affiché à l'extrémité droite de la barre de sous-onglets

#### Scenario: Sous-onglet actif dans l'URL
- **WHEN** l'utilisateur clique sur le sous-onglet « Colonnes »
- **THEN** l'URL porte `tab=columns` et le rechargement de la page rouvre Settings sur « Colonnes »

#### Scenario: Sous-onglet par défaut
- **WHEN** l'utilisateur ouvre Settings sans paramètre `tab`, ou avec une valeur de `tab` inconnue
- **THEN** le sous-onglet « Agent Pool » est actif
