# Spec: Explore Native Question Mode

## Purpose

Permet, en option et pour l'agent Claude uniquement, de réactiver l'outil natif `AskUserQuestion` via un véritable échange `tool_use`/`tool_result` en stream-json, plutôt que la seule convention de marqueur texte.

## Requirements

### Requirement: Activation conditionnelle réservée à Claude

Le mode question native SHALL uniquement produire un effet lorsque l'agent actif de la session est Claude ET que le réglage global (exposé par `agent-selection`) est activé. Pour tout autre agent, ou lorsque le réglage est désactivé, le comportement de la session SHALL rester celui du marqueur texte décrit par `explore-structured-questions`, sans erreur ni différence visible pour l'utilisateur autre que l'absence du rendu natif.

#### Scenario: Réglage activé avec agent Claude
- **WHEN** une session Claude démarre et le réglage global du mode question native est activé
- **THEN** le prompt système autorise l'outil `AskUserQuestion` au lieu de l'interdire

#### Scenario: Réglage activé avec un agent non-Claude
- **WHEN** une session utilisant Codex, Gemini, Copilot ou Antigravity démarre, réglage global activé ou non
- **THEN** le comportement retombe sur la convention de marqueur texte, comme si le réglage était désactivé

#### Scenario: Réglage désactivé
- **WHEN** une session Claude démarre et que le réglage global est désactivé
- **THEN** le comportement retombe sur la convention de marqueur texte, identique au comportement par défaut

### Requirement: Échange tool_use/tool_result natif

Lorsque le mode question native est effectif, le backend SHALL détecter les blocs `tool_use` de nom `AskUserQuestion` produits par le subprocess Claude dans le flux stream-json, transmettre leur contenu (questions, options) au frontend pour un rendu interactif natif, puis, une fois la réponse de l'utilisateur reçue, écrire un bloc `tool_result` correspondant sur le stdin du subprocess pour permettre la poursuite de la conversation.

#### Scenario: Bloc tool_use détecté
- **WHEN** le subprocess Claude produit un bloc `tool_use` de nom `AskUserQuestion`
- **THEN** le backend en extrait les questions et options et les transmet au frontend sans les traiter comme du texte conversationnel ordinaire

#### Scenario: Réponse renvoyée au subprocess
- **WHEN** l'utilisateur répond à une question native affichée
- **THEN** le backend écrit un bloc `tool_result` correspondant au `tool_use_id` d'origine sur le stdin du subprocess Claude, et la conversation se poursuit normalement

### Requirement: Repli silencieux sur le marqueur texte

Si un bloc `tool_use` de nom `AskUserQuestion` ne peut être correctement traité (agent non-Claude recevant malgré tout ce bloc, erreur de parsing), le backend SHALL ignorer ce bloc sans le transmettre tel quel au frontend, plutôt que d'afficher du JSON brut ou une erreur visible dans le chat.

#### Scenario: Bloc non traité par un pont d'agent non-Claude
- **WHEN** un bloc `tool_use` de nom `AskUserQuestion` traverse le pont de traduction d'un agent non-Claude (ex: Gemini)
- **THEN** le bloc est ignoré et n'apparaît pas comme du texte brut dans le fil de conversation
