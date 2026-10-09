# Spec: Platform Configuration

## Purpose

Page de configuration globale de la plateforme, indépendante de tout workspace, permettant de visualiser le registre des agents supportés et de configurer les binaires CLI (variables d'environnement, mode question native).

## Requirements

### Requirement: Accès à Configuration indépendant du workspace
Le système SHALL exposer une entrée de navigation "Configuration" dans la barre principale globale, distincte du sous-menu des onglets liés au workspace actif (Kanban, Specs, Timeline, Agents, Réglages), et accessible même lorsqu'aucun workspace n'est configuré.

#### Scenario: Accès sans workspace configuré
- **WHEN** aucun workspace n'est configuré dans l'application
- **THEN** l'entrée de navigation Configuration est visible
- **THEN** son activation affiche la page Configuration

#### Scenario: Séparation visuelle des onglets liés au workspace
- **WHEN** l'utilisateur navigue vers Configuration
- **THEN** l'URL ne porte pas de paramètre de workspace actif, contrairement aux onglets Kanban/Specs/Timeline/Agents/Réglages

#### Scenario: Emplacement dans le panneau de gauche
- **WHEN** l'utilisateur consulte l'application
- **THEN** l'entrée de navigation Configuration est affichée dans la barre principale en haut de la fenêtre
- **THEN** elle n'est affichée ni dans le panneau de gauche ni dans le sous-menu du workspace

### Requirement: Registre des agents en lecture seule
Le système SHALL afficher, dans la section "CLI" de Configuration, au-dessus de la configuration des variables d'environnement, la liste des agents supportés par la plateforme avec leur statut d'installation et leur version détectée. Cette vue SHALL être en lecture seule.

#### Scenario: Affichage du registre
- **WHEN** l'utilisateur ouvre Configuration > CLI
- **THEN** la liste des agents supportés (Claude, Codex, Gemini, Antigravity, Copilot) s'affiche
- **THEN** chaque agent affiche son statut installé/non installé, et sa version lorsqu'il est installé

#### Scenario: Aucune action de modification
- **WHEN** l'utilisateur consulte Configuration > CLI
- **THEN** aucun contrôle d'édition (sélection de l'agent par défaut, ajout d'un agent) n'est proposé sur cette vue

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

### Requirement: Onglet Agent Pool dans Configuration
Le système SHALL exposer, dans Configuration, un onglet nommé "Agent Pool" à la place de l'ancien onglet "Agents" (dont le contenu de registre est déplacé dans l'onglet "CLI"). Cet onglet SHALL afficher la vue de visibilité globale des agent pools décrite par la capacité `agent-pool-visibility` : tous les pools et workers actuellement actifs, tous workspaces confondus.

#### Scenario: Onglet renommé
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** l'onglet précédemment nommé "Agents" est désormais nommé "Agent Pool"
- **THEN** le registre des CLI installés ne s'affiche plus sous cet onglet

#### Scenario: Contenu de l'onglet Agent Pool
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool
- **THEN** la liste de tous les agent pools actifs, tous workspaces confondus, s'affiche selon les règles de la capacité `agent-pool-visibility`

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

### Requirement: Sous-onglet Langue dans Configuration
Le système SHALL exposer, dans Configuration, un sous-onglet "Langue" aux côtés des sous-onglets "Agents" et "CLI", hébergeant le sélecteur de langue décrit par la capacité `i18n-core`.

#### Scenario: Affichage du sous-onglet Langue
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** un sous-onglet "Langue" est visible aux côtés des sous-onglets "Agents" et "CLI"

#### Scenario: Contenu du sous-onglet Langue
- **WHEN** l'utilisateur active le sous-onglet "Langue"
- **THEN** le sélecteur de langue (EN | FR) s'affiche

### Requirement: Réglage des langues d'agents dans Configuration > Langue
Le système SHALL exposer, dans l'onglet "Langue" de Configuration, en plus du sélecteur de langue de l'interface, trois réglages : langue du chat, langue de la documentation (documentation générée et artefacts OpenSpec) et langue du code. Les réglages du chat et de la documentation SHALL proposer, en plus des langues supportées, une option "suivre la langue de l'application" (`auto`) ; le réglage du code SHALL proposer uniquement les langues supportées. Les langues proposées SHALL provenir d'une liste unique partagée par les trois réglages. Les valeurs SHALL être persistées via `PATCH /api/preferences`, comme décrit par `agent-language-settings`.

#### Scenario: Affichage des valeurs par défaut
- **WHEN** l'utilisateur ouvre Configuration > Langue sans réglage de langue d'agent enregistré
- **THEN** le chat et la documentation affichent l'option "suivre la langue de l'application"
- **THEN** le code affiche l'anglais

#### Scenario: Option auto indiquant la langue résolue
- **WHEN** le chat ou la documentation vaut `auto`
- **THEN** l'interface indique la langue actuellement résolue (celle de l'application)

#### Scenario: Le code n'offre pas d'option auto
- **WHEN** l'utilisateur ouvre la liste du réglage de la langue du code
- **THEN** seules les langues supportées sont proposées, sans option "suivre la langue de l'application"

#### Scenario: Enregistrement d'un réglage
- **WHEN** l'utilisateur change la langue de la documentation en français et enregistre
- **THEN** `preferences.json` reflète ce réglage et `GET /api/preferences` le retourne
- **THEN** les réglages du chat et du code restent inchangés

#### Scenario: Ajout d'une langue supportée
- **WHEN** une nouvelle langue est ajoutée à la liste des langues supportées
- **THEN** elle apparaît dans les trois listes de Configuration > Langue sans autre modification de l'écran

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

### Requirement: Commande de validation par défaut dans Configuration
Configuration > Agent Pool SHALL permettre de définir une commande de validation par défaut (`validationCommand`, texte libre) exécutée par les workers du pool après chaque invocation de l'agent. Cette valeur SHALL être persistée dans `preferences.json` avec les autres défauts du pool, retournée par `GET /api/preferences`, et servir de base à tous les workspaces qui ne la surchargent pas dans Settings. À défaut de valeur enregistrée, le champ SHALL rester vide et les workers SHALL utiliser l'auto-détection de la commande de validation.

#### Scenario: Aucun défaut enregistré
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool sans commande de validation enregistrée
- **THEN** le champ est vide et indique que la commande est auto-détectée

#### Scenario: Défaut enregistré
- **WHEN** l'utilisateur saisit `make test` comme commande de validation par défaut et enregistre
- **THEN** la valeur est persistée dans `preferences.json`, retournée par `GET /api/preferences`, et utilisée par les workers de tout workspace sans surcharge

#### Scenario: Effacement du défaut
- **WHEN** l'utilisateur vide le champ et enregistre
- **THEN** la valeur enregistrée est supprimée et les workers reviennent à l'auto-détection
