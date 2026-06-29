## Purpose

Fournir un indicateur visuel d'attente dans les panneaux Explore, afin de signaler à l'utilisateur que sa requête est en cours de traitement avant la réception du premier token de réponse.

## Requirements

### Requirement: Afficher une bulle d'attente animée après l'envoi d'un message
Dès qu'un message utilisateur est envoyé dans un panneau Explore, le système SHALL afficher une bulle animée (trois points clignotants) à la position assistant dans le fil de messages, jusqu'à la réception du premier token de réponse.

#### Scenario: Bulle apparaît immédiatement après envoi
- **WHEN** l'utilisateur soumet un message dans le champ de saisie
- **THEN** une bulle animée à trois points apparaît immédiatement à la position assistant dans le fil de chat

#### Scenario: Bulle disparaît à l'arrivée du premier token
- **WHEN** le premier contenu texte non-vide de l'assistant est reçu via WebSocket
- **THEN** la bulle animée disparaît et le streaming normal commence (curseur ▊ visible)

#### Scenario: Bulle disparaît sur déconnexion
- **WHEN** la connexion WebSocket se ferme ou produit une erreur pendant l'attente
- **THEN** la bulle animée disparaît immédiatement

### Requirement: Afficher un label "réfléchit" après 5 secondes d'attente
Si l'attente dépasse 5 secondes sans réponse, le système SHALL afficher le nom de l'assistant suivi de "réfléchit..." au-dessus de la bulle animée.

#### Scenario: Label affiché après délai
- **WHEN** 5 secondes s'écoulent depuis l'envoi du message sans réception du premier token
- **THEN** un texte du type "Claude réfléchit..." apparaît au-dessus des points animés dans la même bulle

#### Scenario: Label supprimé quand la réponse arrive
- **WHEN** le premier token arrive après que le label était déjà visible
- **THEN** la bulle complète (label + points) disparaît et le streaming démarre normalement

#### Scenario: Label réinitialisé sur le message suivant
- **WHEN** l'utilisateur envoie un nouveau message après une réponse complète
- **THEN** le label "réfléchit..." ne s'affiche pas immédiatement — le timer de 5s repart à zéro

### Requirement: Nom de l'assistant configurable via prop
Les panneaux Explore SHALL accepter une prop `assistantName` (string, optionnel, défaut `"Claude"`) utilisée dans le label "réfléchit..." et dans tout indicateur identifiant l'assistant.

#### Scenario: Prop non fournie — valeur par défaut
- **WHEN** `ExplorePanel` ou `ExploreAnonymousPanel` est rendu sans la prop `assistantName`
- **THEN** le label affiché après 5s est "Claude réfléchit..."

#### Scenario: Prop fournie — valeur personnalisée
- **WHEN** `ExplorePanel` ou `ExploreAnonymousPanel` est rendu avec `assistantName="Gemini"`
- **THEN** le label affiché après 5s est "Gemini réfléchit..."

### Requirement: Input non désactivé pendant l'attente
Le champ de saisie SHALL rester actif et éditable pendant toute la durée de l'attente d'une réponse. L'utilisateur SHALL pouvoir saisir et envoyer un nouveau message sans attendre la réponse en cours.

#### Scenario: Saisie possible pendant l'attente
- **WHEN** une bulle d'attente est affichée (waiting=true)
- **THEN** le champ de saisie est enabled et le bouton "Envoyer" peut être utilisé si le champ n'est pas vide

#### Scenario: Envoi d'un second message pendant l'attente
- **WHEN** l'utilisateur envoie un second message alors que la bulle d'attente est visible
- **THEN** le second message apparaît dans le fil et la bulle reste visible (toujours en attente de la première réponse)
