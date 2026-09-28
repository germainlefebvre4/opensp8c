# Spec Delta

## Purpose

Page de configuration globale de la plateforme, indépendante de tout workspace, permettant de visualiser le registre des agents supportés et de configurer les binaires CLI (variables d'environnement, mode question native).

## ADDED Requirements

### Requirement: Accès à Configuration indépendant du workspace
Le système SHALL exposer une entrée de navigation "Configuration", distincte de la liste d'onglets liés au workspace actif (Kanban, Specs, Timeline, Agents, Réglages), et accessible même lorsqu'aucun workspace n'est configuré.

#### Scenario: Accès sans workspace configuré
- **WHEN** aucun workspace n'est configuré dans l'application
- **THEN** l'entrée de navigation Configuration est visible
- **THEN** son activation affiche la page Configuration

#### Scenario: Séparation visuelle des onglets liés au workspace
- **WHEN** l'utilisateur navigue vers Configuration
- **THEN** l'URL ne porte pas de paramètre de workspace actif, contrairement aux onglets Kanban/Specs/Timeline/Agents/Réglages

### Requirement: Sidebar conservée sur Configuration
Le système SHALL conserver l'affichage de la sidebar workspace (sélecteur d'agent par défaut, liste des projets) lorsque la page Configuration est affichée.

#### Scenario: Sidebar visible sur Configuration
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** la sidebar (sélecteur d'agent par défaut, liste des projets) reste affichée à gauche de l'écran

### Requirement: Registre des agents en lecture seule
Le système SHALL afficher, dans la section "Agents" de Configuration, la liste des agents supportés par la plateforme avec leur statut d'installation et leur version détectée. Cette vue SHALL être en lecture seule.

#### Scenario: Affichage du registre
- **WHEN** l'utilisateur ouvre Configuration > Agents
- **THEN** la liste des agents supportés (Claude, Codex, Gemini, Antigravity, Copilot) s'affiche
- **THEN** chaque agent affiche son statut installé/non installé, et sa version lorsqu'il est installé

#### Scenario: Aucune action de modification
- **WHEN** l'utilisateur consulte Configuration > Agents
- **THEN** aucun contrôle d'édition (sélection de l'agent par défaut, ajout d'un agent) n'est proposé sur cette vue

### Requirement: Affichage adaptatif des variables recommandées
Le système SHALL afficher, dans la section "CLI" de Configuration, les variables d'environnement recommandées (`GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, `GEMINI_SANDBOX`) et distinguer visuellement une valeur système héritée d'une surcharge utilisateur.

#### Scenario: Affichage par défaut avec valeur système présente
- **WHEN** Configuration > CLI est affichée, qu'un champ recommandé est vide de toute surcharge utilisateur, mais possède une valeur définie au niveau du système hôte (présente dans `systemEnv`)
- **THEN** l'input affiche cette valeur système comme placeholder (ex: "Système : mon-projet")
- **THEN** un message discret de statut s'affiche sous le champ indiquant qu'elle sera héritée de l'environnement système

#### Scenario: Affichage avec surcharge utilisateur
- **WHEN** l'utilisateur saisit sa propre valeur dans un champ recommandé alors qu'une valeur système existe
- **THEN** l'input affiche la valeur saisie par l'utilisateur
- **THEN** un message discret de statut indique qu'elle surcharge la valeur système

### Requirement: Gestion des variables d'environnement personnalisées
Le système SHALL permettre, dans Configuration > CLI, d'ajouter et de retirer des variables d'environnement personnalisées (clé/valeur libres), persistées dans `preferences.json` (champ `env`) via les endpoints existants `GET`/`PATCH /api/preferences`.

#### Scenario: Ajout d'une variable personnalisée
- **WHEN** l'utilisateur ajoute une paire clé/valeur dans Configuration > CLI et enregistre
- **THEN** `preferences.json` est mis à jour avec cette variable dans `env`

#### Scenario: Suppression d'une variable personnalisée
- **WHEN** l'utilisateur retire une variable personnalisée existante et enregistre
- **THEN** cette variable n'apparaît plus dans `env` après enregistrement

### Requirement: Réglage du mode question native
Le système SHALL exposer, dans Configuration > CLI, une bascule "mode question native (Claude)" permettant d'activer ou de désactiver le mode question native décrit par `explore-native-question-mode`. Cette préférence SHALL être persistée dans `preferences.json` sous un champ booléen dédié, désactivée par défaut. L'interface SHALL indiquer que ce réglage n'a d'effet que pour l'agent Claude.

#### Scenario: Affichage de la bascule
- **WHEN** l'utilisateur ouvre Configuration > CLI
- **THEN** une bascule "mode question native (Claude)" est visible, accompagnée d'une mention indiquant qu'elle est sans effet pour les autres agents

#### Scenario: Activation de la bascule
- **WHEN** l'utilisateur active la bascule et enregistre
- **THEN** `preferences.json` est mis à jour avec le champ booléen à `true`, et `GET /api/preferences` reflète ce changement

#### Scenario: Valeur par défaut
- **WHEN** `preferences.json` est créé pour la première fois ou que le champ n'a jamais été défini
- **THEN** le mode question native est considéré comme désactivé
