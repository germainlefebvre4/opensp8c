# explore-session-restart Specification

## Purpose

Permettre à l'utilisateur de débloquer une conversation d'exploration dont le tour d'agent semble figé, en redémarrant le sous-processus de l'agent sans perdre l'historique ni le contexte de la conversation.

## Requirements

### Requirement: Détection d'une session d'exploration bloquée
Le frontend SHALL considérer une session d'exploration comme potentiellement bloquée lorsqu'elle est en attente d'une réponse de l'agent (`waiting` à `true`) et qu'aucun message n'a été reçu de sa part depuis un délai de silence. Ce délai SHALL être de 60 secondes, porté à 180 secondes tant qu'un appel d'outil de l'agent est en cours (appel sans résultat associé). Toute réception d'un message de l'agent SHALL remettre le délai à zéro.

#### Scenario: Silence prolongé pendant l'attente
- **WHEN** `waiting` est `true` et aucun message de l'agent n'a été reçu depuis 60 secondes, sans appel d'outil en cours
- **THEN** la session est considérée comme potentiellement bloquée et le bouton « Relancer l'agent » devient visible

#### Scenario: Appel d'outil en cours — délai étendu
- **WHEN** `waiting` est `true`, un appel d'outil est en cours (sans résultat reçu) et 60 secondes se sont écoulées sans message
- **THEN** le bouton « Relancer l'agent » reste masqué, et devient visible seulement à 180 secondes de silence

#### Scenario: Activité reçue — délai réinitialisé
- **WHEN** un message de l'agent (texte partiel, appel d'outil ou résultat) est reçu alors que le bouton est masqué ou visible
- **THEN** le délai de silence repart à zéro et le bouton est masqué

#### Scenario: Pas d'attente — pas de bouton
- **WHEN** `waiting` est `false` (aucune réponse attendue), quelle que soit la durée écoulée depuis le dernier message
- **THEN** le bouton « Relancer l'agent » n'est pas affiché

### Requirement: Bouton « Relancer l'agent » dans les panneaux d'exploration
Les panneaux d'exploration nommé et anonyme SHALL afficher un bouton « Relancer l'agent » à proximité du fil de messages lorsque la session est considérée comme bloquée. Le bouton SHALL être proposé uniquement pour les sessions dont l'agent actif est Claude.

#### Scenario: Bouton affiché pour une session Claude bloquée
- **WHEN** une session dont l'agent actif est Claude est considérée comme bloquée
- **THEN** le bouton « Relancer l'agent » est affiché dans le panneau, sans masquer le champ de saisie

#### Scenario: Agent autre que Claude
- **WHEN** l'agent actif de la session est Gemini, Antigravity, Codex ou Copilot
- **THEN** le bouton « Relancer l'agent » n'est jamais affiché, même après un long silence

### Requirement: Redémarrage de l'agent avec conservation de la conversation
Le clic sur « Relancer l'agent » SHALL arrêter le sous-processus de la session puis en démarrer un nouveau qui reprend la même conversation (même identifiant de session Claude, via `--resume`). L'historique affiché SHALL être conservé, `waiting` SHALL revenir à `false`, et aucun message utilisateur ne SHALL être renvoyé automatiquement.

#### Scenario: Redémarrage d'une exploration nommée
- **WHEN** l'utilisateur clique sur « Relancer l'agent » dans le panneau d'un change nommé
- **THEN** la session backend est arrêtée puis rouverte avec `--resume <claudeSessionId>`, le fil de messages existant reste affiché, et l'agent reprend le contexte de la conversation

#### Scenario: Redémarrage d'une exploration anonyme (ghost)
- **WHEN** l'utilisateur clique sur « Relancer l'agent » dans le panneau d'une exploration anonyme
- **THEN** la session backend est arrêtée puis rouverte sous le même identifiant de ghost avec `--resume <claudeSessionId>` et le fil de messages existant reste affiché, sans réinjection du transcript

#### Scenario: Aucun renvoi automatique
- **WHEN** la session redémarre alors que le dernier message utilisateur est resté sans réponse
- **THEN** ce message n'est pas renvoyé automatiquement ; l'utilisateur peut le renvoyer ou en saisir un nouveau

#### Scenario: Question en attente au moment du redémarrage
- **WHEN** une question de clarification restait sans réponse au moment du redémarrage
- **THEN** le comportement de reprise d'une question en attente de la capability `explore-session-resume` s'applique

#### Scenario: Échec du redémarrage
- **WHEN** le nouveau sous-processus ne peut pas être démarré
- **THEN** un message d'avertissement est affiché dans le fil, `waiting` est `false`, et le bouton « Relancer l'agent » reste affiché (pour un agent Claude) jusqu'à la prochaine tentative ou le prochain message de l'utilisateur, afin de pouvoir réessayer

### Requirement: Message système de relance dans le fil
Après un redémarrage demandé par l'utilisateur, le fil de messages SHALL afficher un message système discret « Agent relancé », distinct des messages de l'utilisateur et de l'assistant, et ne SHALL PAS être envoyé à l'agent.

#### Scenario: Message affiché après relance
- **WHEN** le redémarrage d'une session réussit
- **THEN** une ligne système « Agent relancé » est ajoutée à la fin du fil de messages

#### Scenario: Message non transmis à l'agent
- **WHEN** la ligne système « Agent relancé » est ajoutée
- **THEN** aucun message correspondant n'est écrit sur l'entrée du sous-processus de l'agent
