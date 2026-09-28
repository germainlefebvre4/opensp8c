# Spec Delta

## MODIFIED Requirements

### Requirement: Détection du marqueur de question

Le backend SHALL détecter, dans le texte produit par le subprocess agent (tout agent CLI supporté), tous les marqueurs `ghost_question` signalant des questions de clarification, qu'ils apparaissent isolément ou de façon multiple dans un même tour ou flux de streaming. Dès détection de chaque marqueur, le backend SHALL émettre un événement WebSocket dédié contenant le texte de la question et mémoriser l'état de question en attente tant qu'aucune réponse n'a été envoyée. De plus, le backend SHALL nettoyer intégralement les marqueurs `ghost_question` du flux textuel retransmis au client (notamment les deltas de streaming `content_block_delta` et messages consolidés) afin qu'aucun JSON brut n'apparaisse dans les bulles de texte.

#### Scenario: Marqueur détecté dans le flux de sortie
- **WHEN** le subprocess agent écrit une ou plusieurs lignes contenant un ou des marqueurs `ghost_question` avec un texte de question
- **THEN** le backend extrait chaque question, diffuse un événement `ghost_question` pour chacune au frontend, et purge complètement les marqueurs du texte retransmis

#### Scenario: Réponse de l'utilisateur efface l'état en attente
- **WHEN** l'utilisateur envoie son message contenant les réponses aux questions
- **THEN** l'état "question en attente" de la session est effacé

### Requirement: Une question affichée à la fois

Le frontend SHALL afficher chaque question reçue sous forme de carte interactive distincte dans le fil de conversation. Contrairement au comportement antérieur où une nouvelle question rendait obsolète la précédente, le frontend SHALL permettre à plusieurs cartes de questions de coexister et de demeurer actives simultanément tant que l'utilisateur n'y a pas répondu.

#### Scenario: Nouvelle question avant réponse à la précédente
- **WHEN** l'agent émet une nouvelle question alors qu'une question précédente est encore sans réponse
- **THEN** la nouvelle question est affichée à la suite dans le fil, et les deux questions demeurent actives, éligibles à une réponse, sans qu'aucune ne soit marquée obsolète

### Requirement: Redirection de "Autre réponse" vers le champ principal

Lorsqu'une carte de question propose des options de réponse rapide, l'utilisateur SHALL pouvoir choisir une option prédéfinie ou opter pour une réponse personnalisée. Si l'utilisateur choisit une réponse personnalisée, la saisie textuelle s'effectue directement sur la carte de question pour préparer sa réponse sans envoi prématuré.

#### Scenario: Clic sur "Autre réponse..."
- **WHEN** l'utilisateur clique sur "Autre réponse..." dans une carte de question
- **THEN** un champ de saisie textuel s'active directement dans la carte de question pour lui permettre de renseigner sa réponse personnalisée localement

## ADDED Requirements

### Requirement: Saisie et staging des réponses sur chaque question

Le frontend SHALL permettre à l'utilisateur de saisir et valider localement une réponse sur chaque carte de question active sans que cette réponse ne soit immédiatement transmise au serveur. Dès qu'une réponse est validée pour une question, la carte affiche un indicateur visuel de réponse préparée, et la réponse est conservée dans l'état local du client en attente de l'envoi global.

#### Scenario: Validation locale d'une réponse
- **WHEN** l'utilisateur renseigne et valide sa réponse dans le champ d'une carte de question
- **THEN** la réponse est enregistrée localement en attente d'envoi, la carte affiche le statut de réponse préparée, et aucun message WebSocket n'est émis vers le serveur

### Requirement: Encarts de réponses préparées dans le fil de conversation

Tant qu'au moins une réponse est préparée et en attente d'envoi, le frontend SHALL matérialiser chaque réponse préparée par un encart temporaire dédié intégré directement dans la fenêtre du fil de conversation (au bas des messages, avant la barre de saisie). Chaque encart SHALL afficher un rappel de la question associée, le texte de la réponse préparée, un bouton pour modifier la réponse, et un bouton pour l'annuler.

#### Scenario: Affichage des encarts temporaires
- **WHEN** une ou plusieurs réponses sont préparées par l'utilisateur
- **THEN** des encarts temporaires correspondants s'insèrent dans le fil de conversation, récapitulant chaque réponse préparée avec les contrôles d'édition et de suppression

#### Scenario: Modification d'une réponse préparée depuis l'encart
- **WHEN** l'utilisateur clique sur le bouton de modification d'un encart
- **THEN** le champ de saisie de la carte correspondante est réactivé avec le texte actuel pour permettre sa modification

#### Scenario: Annulation d'une réponse préparée
- **WHEN** l'utilisateur clique sur le bouton de suppression d'un encart
- **THEN** la réponse préparée est retirée de l'état local, l'encart disparaît, et la carte de question redevient vide et active

### Requirement: Envoi consolidé et maintien des questions ouvertes

L'utilisateur SHALL être le seul initiateur de l'envoi de ses réponses préparées via le bouton d'envoi principal du chat. Le bouton d'envoi SHALL être actif dès lors qu'au moins une réponse est préparée, même si le champ de prompt libre est vide. À l'envoi, le frontend SHALL formater un message consolidé combinant les questions/réponses traitées et le prompt libre éventuel, supprimer les encarts temporaires, basculer les questions traitées en état répondu, et laisser intactes et ouvertes les questions auxquelles l'utilisateur n'a pas répondu.

#### Scenario: Envoi des réponses avec ou sans prompt libre
- **WHEN** l'utilisateur clique sur Envoyer avec des réponses préparées et optionnellement un prompt libre
- **THEN** un message consolidé est transmis au subprocess, les encarts temporaires disparaissent du fil, les questions ayant reçu une réponse se réduisent en résumé, et les questions restées sans réponse demeurent ouvertes pour le tour suivant
