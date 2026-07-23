## ADDED Requirements

### Requirement: Détection des agents CLI installés
Le système SHALL exposer un endpoint qui probe chaque agent CLI supporté et retourne son état d'installation et sa version.

#### Scenario: Agent installé
- **WHEN** `GET /api/agents` est appelé et le CLI de l'agent est présent sur le système
- **THEN** l'entrée de l'agent dans la réponse contient `installed: true` et la version détectée

#### Scenario: Agent non installé
- **WHEN** `GET /api/agents` est appelé et le CLI de l'agent est absent du PATH
- **THEN** l'entrée de l'agent dans la réponse contient `installed: false` et `version: null`

#### Scenario: Liste complète
- **WHEN** `GET /api/agents` est appelé
- **THEN** la réponse contient une entrée pour chacun des agents supportés : Claude, Codex, Gemini, Antigravity, Copilot

---

### Requirement: Sélecteur d'agent global dans le menu
Le système SHALL afficher un sélecteur d'agent dans le menu gauche, au-dessus de la liste des workspaces, permettant à l'utilisateur de définir l'agent par défaut pour les nouvelles conversations.

#### Scenario: Affichage du sélecteur
- **WHEN** l'utilisateur ouvre l'application
- **THEN** le sélecteur affiche l'agent actuellement sélectionné comme défaut

#### Scenario: Agents non installés grisés
- **WHEN** le sélecteur est ouvert
- **THEN** les agents dont le CLI n'est pas installé sont affichés en grisé et ne sont pas sélectionnables

#### Scenario: Agents installés avec version
- **WHEN** le sélecteur est ouvert
- **THEN** chaque agent installé affiche son numéro de version

#### Scenario: Changement d'agent global
- **WHEN** l'utilisateur sélectionne un agent différent dans le sélecteur
- **THEN** la préférence `defaultAgent` est mise à jour dans preferences.json
- **THEN** les conversations déjà ouvertes ne sont pas affectées

---

### Requirement: Persistance de la préférence d'agent
Le système SHALL persister la préférence d'agent de l'utilisateur dans un fichier `preferences.json` local à l'application, sans modifier les fichiers du projet.

#### Scenario: Lecture de la préférence
- **WHEN** `GET /api/preferences` est appelé
- **THEN** la réponse contient `defaultAgent` avec l'identifiant de l'agent sélectionné

#### Scenario: Mise à jour de la préférence
- **WHEN** `PATCH /api/preferences` est appelé avec `{ "defaultAgent": "<id>" }`
- **THEN** preferences.json est mis à jour avec le nouvel agent par défaut

#### Scenario: Initialisation au premier démarrage
- **WHEN** preferences.json est absent au démarrage de l'application
- **THEN** preferences.json est créé avec `defaultAgent: "claude"` par défaut

---

### Requirement: Verrouillage de l'agent par session
Le système SHALL verrouiller l'agent d'une conversation à sa création — il ne peut pas changer pendant toute la durée de la session, même si l'utilisateur change l'agent global entre-temps.

#### Scenario: Résolution de l'agent pour une nouvelle named session
- **WHEN** une named session est créée pour un Change qui n'a pas encore d'agent mémorisé
- **THEN** l'agent utilisé est `defaultAgent` depuis preferences.json
- **THEN** la correspondance `workspaceID/changeName → agentID` est écrite dans `sessionAgents`

#### Scenario: Résolution de l'agent pour une named session existante
- **WHEN** une named session est ouverte pour un Change qui a déjà un agent mémorisé dans `sessionAgents`
- **THEN** l'agent utilisé est celui mémorisé, même si `defaultAgent` a changé depuis

#### Scenario: Résolution de l'agent pour une anonymous session
- **WHEN** une anonymous session est créée
- **THEN** l'agent utilisé est `defaultAgent` depuis preferences.json
- **THEN** aucune entrée n'est écrite dans `sessionAgents` (sessions anonymes non persistées)

#### Scenario: Fallback si l'agent mémorisé n'est plus installé
- **WHEN** l'agent mémorisé pour une named session n'est plus installé sur le système
- **THEN** le système utilise Claude comme fallback
- **THEN** un message d'avertissement est envoyé dans la conversation pour informer l'utilisateur

---

### Requirement: Indicateur d'agent actif dans la conversation
Le système SHALL afficher un badge indiquant l'agent actif et sa version dans l'en-tête de chaque panneau de conversation (named et anonymous).

#### Scenario: Affichage du badge agent
- **WHEN** une conversation est ouverte
- **THEN** un badge affiche le nom de l'agent actif et sa version dans l'en-tête du panneau

#### Scenario: Badge pour named session avec agent mémorisé
- **WHEN** une named session est ouverte avec un agent différent du `defaultAgent` courant
- **THEN** le badge affiche l'agent réellement utilisé (celui mémorisé), pas l'agent global courant
