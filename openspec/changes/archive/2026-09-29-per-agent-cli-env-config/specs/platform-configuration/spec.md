# Spec Delta

## MODIFIED Requirements

### Requirement: Affichage adaptatif des variables recommandées
Le système SHALL afficher, dans la vue dédiée à l'agent Gemini (Configuration > CLI > Gemini), les variables d'environnement recommandées pour cet agent (`GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, `GEMINI_SANDBOX`) et distinguer visuellement une valeur système héritée d'une surcharge utilisateur. Ces variables ne SHALL PAS être affichées dans le formulaire global de Configuration > CLI.

#### Scenario: Affichage par défaut avec valeur système présente
- **WHEN** la vue Configuration > CLI > Gemini est affichée, qu'un champ recommandé est vide de toute surcharge utilisateur, mais possède une valeur définie au niveau du système hôte (présente dans `systemEnv`)
- **THEN** l'input affiche cette valeur système comme placeholder (ex: "Système : mon-projet")
- **THEN** un message discret de statut s'affiche sous le champ indiquant qu'elle sera héritée de l'environnement système

#### Scenario: Affichage avec surcharge utilisateur
- **WHEN** l'utilisateur saisit sa propre valeur dans un champ recommandé de la vue Gemini alors qu'une valeur système existe
- **THEN** l'input affiche la valeur saisie par l'utilisateur
- **THEN** un message discret de statut indique qu'elle surcharge la valeur système

## ADDED Requirements

### Requirement: Vue de configuration dédiée par agent CLI
Le système SHALL permettre, depuis le registre affiché dans Configuration > CLI, d'ouvrir une vue dédiée à un agent CLI précis en cliquant sur sa ligne. Cette vue SHALL afficher la liste librement éditable des variables d'environnement propres à cet agent (ajout et suppression de paires clé/valeur), persistées via `PATCH /api/preferences` dans `agentEnv.<agentID>`, ainsi qu'un lien vers la documentation officielle de cet agent CLI. Cette vue SHALL rester accessible pour un agent non installé sur le système.

#### Scenario: Ouverture de la vue d'un agent depuis le registre
- **WHEN** l'utilisateur clique sur la ligne de l'agent `codex` dans Configuration > CLI
- **THEN** l'application affiche la vue de configuration dédiée à `codex`, sans quitter Configuration > CLI

#### Scenario: Ajout d'une variable spécifique à un agent
- **WHEN** l'utilisateur ajoute une paire clé/valeur dans la vue dédiée à l'agent `antigravity` et enregistre
- **THEN** `preferences.json` est mis à jour avec cette variable dans `agentEnv.antigravity`, sans affecter le dictionnaire global `env` ni celui des autres agents

#### Scenario: Suppression d'une variable spécifique à un agent
- **WHEN** l'utilisateur retire une variable existante dans la vue dédiée à un agent et enregistre
- **THEN** cette variable n'apparaît plus dans `agentEnv.<agentID>` après enregistrement

#### Scenario: Lien vers la documentation officielle
- **WHEN** l'utilisateur ouvre la vue dédiée à un agent CLI
- **THEN** un lien vers la documentation officielle de cet agent est affiché

#### Scenario: Vue accessible pour un agent non installé
- **WHEN** l'utilisateur clique sur la ligne d'un agent dont le CLI n'est pas installé sur le système
- **THEN** la vue de configuration dédiée à cet agent s'affiche normalement, permettant d'y renseigner des variables d'environnement par anticipation
