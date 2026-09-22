# Spec: Explore Structured Questions

## Purpose

Rend visibles et distinctes, dans le fil de chat d'exploration, les questions de clarification posées par l'agent — via un marqueur texte universel, une carte dédiée, et un état "en attente" que la reprise de session sait exploiter.

## Requirements

### Requirement: Détection du marqueur de question

Le backend SHALL détecter, dans le texte produit par le subprocess agent (tout agent CLI supporté), un marqueur `ghost_question` signalant une question de clarification, sur le même principe de détection que le marqueur `ghost_named` existant. Dès détection, la session SHALL mémoriser un état "question en attente" tant qu'aucune réponse n'a été envoyée par l'utilisateur, et un événement dédié SHALL être diffusé au frontend actif.

#### Scenario: Marqueur détecté dans le flux de sortie
- **WHEN** le subprocess agent écrit une ligne contenant le marqueur `ghost_question` avec un texte de question
- **THEN** le backend extrait la question, marque la session comme ayant une "question en attente", et diffuse un événement au frontend contenant le texte de la question

#### Scenario: Réponse de l'utilisateur efface l'état en attente
- **WHEN** l'utilisateur envoie un message après qu'une question a été détectée
- **THEN** l'état "question en attente" de la session est effacé

### Requirement: Affichage d'une question sous forme de carte détachée

Le frontend SHALL afficher toute question détectée dans une carte visuellement distincte du reste du fil de conversation : fond blanc, bordure et ombre légère, séparée du style de bulle standard (`bg-slate-100`) utilisé pour le texte conversationnel ordinaire de l'agent.

#### Scenario: Rendu d'une question
- **WHEN** un événement de question est reçu par le frontend
- **THEN** une carte dédiée s'affiche dans le fil de chat, à la place d'une bulle de texte standard, contenant le texte de la question

### Requirement: Une question affichée à la fois

Le frontend SHALL n'afficher qu'une seule carte de question active (non répondue) à la fois dans le fil de conversation, quel que soit le nombre de questions que l'agent a pu poser en amont.

#### Scenario: Nouvelle question avant réponse à la précédente
- **WHEN** l'agent émet une nouvelle question alors qu'une question précédente est encore sans réponse
- **THEN** la nouvelle question est affichée à la suite dans le fil, sans qu'aucun regroupement de plusieurs questions actives dans une seule carte ne soit produit

### Requirement: Réduction en résumé après réponse

Une fois qu'une réponse a été envoyée par l'utilisateur suite à une question affichée, la carte correspondante SHALL se réduire en une ligne de résumé compacte au sein de la même carte, remplaçant les éléments de réponse interactifs par le texte de la réponse donnée.

#### Scenario: Carte réduite après réponse
- **WHEN** l'utilisateur répond à une question affichée (via une option proposée ou via le champ de saisie principal)
- **THEN** la carte affiche une ligne "-> Votre réponse : <texte de la réponse>" à la place des éléments de réponse interactifs, et reste visible sous cette forme réduite dans l'historique du fil

### Requirement: Redirection de "Autre réponse" vers le champ principal

Lorsqu'une carte de question propose des options de réponse rapide, un lien ou bouton "Autre réponse..." SHALL être proposé. Son activation SHALL uniquement déplacer le focus vers le champ de saisie principal du chat, sans ouvrir de champ de texte additionnel dans la carte elle-même. Le champ de saisie principal SHALL rester utilisable à tout moment, y compris lorsqu'une carte de question est affichée sans options de réponse rapide.

#### Scenario: Clic sur "Autre réponse..."
- **WHEN** l'utilisateur clique sur "Autre réponse..." dans une carte de question
- **THEN** le focus se déplace vers le champ de saisie principal du chat, sans qu'aucun champ de texte ne s'ouvre dans la carte
