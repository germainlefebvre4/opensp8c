# Proposal

## Why

Aujourd'hui, ouvrir une exploration mène à un chat vide avec un message d'accueil générique codé en dur : aucun cadrage n'est collecté à l'entrée, contrairement à ce que décrit AGENTS.md pour la colonne **To Explore** (entrée attendue : description, questions ouvertes). Pire, toute question de clarification posée par l'agent se noie dans du texte libre indiscernable du reste de la conversation, car `AskUserQuestion` et les prompts à choix interactifs sont explicitement interdits au subprocess (`explore-session`, requirement "Interdire les interactions à choix multiples"). AGENTS.md attend pourtant, en sortie de la colonne To Explore, des "réponses aux questions ouvertes" comme artefact distinct du résumé — rien ne les distingue actuellement dans l'UI. Ce manque rend le cadrage d'une exploration et le suivi de ses questions/réponses pénibles à suivre pour l'utilisateur.

## What Changes

- Le premier vrai tour de l'agent (et non un message d'accueil statique côté frontend) porte le cadrage initial de l'exploration, via des questions posées progressivement dans la conversation.
- Toute question de clarification de l'agent est signalée par un marqueur texte dédié (`ghost_question`), sur le même principe que le marqueur `ghost_named` déjà utilisé. Le frontend la rend comme une carte détachée (fond blanc, bordure, ombre légère) distincte du flux conversationnel ordinaire, une question à la fois — jamais plusieurs cartes groupées dans un même message.
- Une fois répondue, la carte se réduit en une ligne de résumé compacte ("-> Votre réponse : ...") au sein de la même bulle, pour rester lisible en remontant l'historique.
- Un clic sur "Autre réponse..." ne déclenche aucun champ inline : il redirige simplement le focus vers le champ de saisie principal du chat, qui reste utilisable à tout moment.
- Ce mécanisme de marqueur est natif et actif par défaut pour tous les agents CLI supportés (Claude, Codex, Gemini, Copilot, Antigravity) — aucune configuration requise.
- Pour Claude uniquement, un réglage global optionnel (désactivé par défaut, activable dans `AgentSettingsModal`) réactive l'outil natif `AskUserQuestion` via un véritable échange `tool_use`/`tool_result` en stream-json. Sur tout autre agent, ou si le réglage est désactivé, le comportement retombe silencieusement sur le marqueur texte.
- À la reprise d'une session (reconnexion après fermeture d'onglet, timeout d'inactivité, ou redémarrage backend) alors qu'une question restait sans réponse, le backend invite l'agent à la reposer plutôt que de tenter de restaurer l'état d'un sous-processus interrompu (qui ne peut de toute façon pas survivre à la coupure).

## Capabilities

### New Capabilities
- `explore-structured-questions`: rendu structuré des questions de clarification de l'agent dans le chat (carte détachée, collapse en résumé, redirection "Autre réponse" vers le champ principal, pacing une question à la fois), détection du marqueur `ghost_question`, et logique de "reposer la question" à la reprise d'une session interrompue.
- `explore-native-question-mode`: bascule optionnelle réactivant l'outil natif `AskUserQuestion` pour l'agent Claude (parsing des blocs `tool_use`/`tool_result` en stream-json), avec repli automatique et silencieux sur le marqueur texte pour les autres agents ou quand la bascule est désactivée.

### Modified Capabilities
- `explore-session`: le prompt système par défaut n'interdit plus uniquement `AskUserQuestion` sans alternative — il instruit désormais l'agent à utiliser le marqueur `ghost_question` pour ses questions de cadrage et de clarification (sessions nommées et anonymes), avec une exception conditionnelle pour Claude quand `explore-native-question-mode` est activé.
- `explore-session-resume`: ajoute le comportement de reprise pour une question restée sans réponse au moment de l'interruption (re-déclenchement de la question plutôt que restauration d'un état de sous-processus).
- `agent-selection`: ajoute un réglage global dans `AgentSettingsModal` pour activer le mode question native (Claude uniquement), persisté dans `preferences.json`, sans effet sur les autres agents ni sur l'agent global par défaut.

## Impact

- **Frontend** : nouveau composant de rendu de carte-question dans le voisinage des composants de chat existants (`ExplorePanel.tsx`, `ExploreAnonymousPanel.tsx`, `TypingBubble.tsx`) ; état `pendingQuestion` et parsing du marqueur dans `useExploreSession.ts` / `useAnonymousExploreSession.ts` ; nouveau champ toggle dans `AgentSettingsModal.tsx` / `useAgentPreferences.ts` ; suppression du message d'accueil statique (`STATIC_GREETING`) ; nouvelles clés i18n (fr/en).
- **Backend** : prompt système étendu dans `session/manager.go` (`baseSystemPrompt`, `anonSystemPrompt`) ; détection du marqueur `ghost_question` et état de question en attente dans `Session` (`session/manager.go`) ; parsing des blocs `tool_use`/écriture de `tool_result` sur stdin pour Claude (`session/subprocess.go`, `api/handlers/explore.go`) ; nouveau champ booléen dans `preferences` (service + JSON persisté).
- Aucune rupture d'API publique attendue : les changements de prompt système et de comportement de reprise sont des détails d'implémentation ; les nouveaux champs de préférences et endpoints associés sont additifs.
