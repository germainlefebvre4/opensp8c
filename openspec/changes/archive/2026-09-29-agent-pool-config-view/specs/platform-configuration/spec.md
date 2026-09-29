# Spec Delta

## MODIFIED Requirements

### Requirement: Registre des agents en lecture seule
Le système SHALL afficher, dans la section "CLI" de Configuration, au-dessus de la configuration des variables d'environnement, la liste des agents supportés par la plateforme avec leur statut d'installation et leur version détectée. Cette vue SHALL être en lecture seule.

#### Scenario: Affichage du registre
- **WHEN** l'utilisateur ouvre Configuration > CLI
- **THEN** la liste des agents supportés (Claude, Codex, Gemini, Antigravity, Copilot) s'affiche
- **THEN** chaque agent affiche son statut installé/non installé, et sa version lorsqu'il est installé

#### Scenario: Aucune action de modification
- **WHEN** l'utilisateur consulte Configuration > CLI
- **THEN** aucun contrôle d'édition (sélection de l'agent par défaut, ajout d'un agent) n'est proposé sur cette vue

## ADDED Requirements

### Requirement: Onglet Agent Pool dans Configuration
Le système SHALL exposer, dans Configuration, un onglet nommé "Agent Pool" à la place de l'ancien onglet "Agents" (dont le contenu de registre est déplacé dans l'onglet "CLI"). Cet onglet SHALL afficher la vue de visibilité globale des agent pools décrite par la capacité `agent-pool-visibility` : tous les pools et workers actuellement actifs, tous workspaces confondus.

#### Scenario: Onglet renommé
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** l'onglet précédemment nommé "Agents" est désormais nommé "Agent Pool"
- **THEN** le registre des CLI installés ne s'affiche plus sous cet onglet

#### Scenario: Contenu de l'onglet Agent Pool
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool
- **THEN** la liste de tous les agent pools actifs, tous workspaces confondus, s'affiche selon les règles de la capacité `agent-pool-visibility`
