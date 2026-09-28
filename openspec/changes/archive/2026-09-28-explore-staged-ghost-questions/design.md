# Design

## Context

See `proposal.md` for the motivation.

Le backend (`internal/api/handlers/explore.go` et `internal/session/manager.go`) traite les flux de sortie des sous-processus agents (Antigravity, Gemini, Claude, etc.). Actuellement, `ExtractGhostQuestion` ne retourne que la première question trouvée dans une ligne ou un delta, et `stripGhostQuestionMarker` n'élimine pas toujours tous les marqueurs lorsque l'agent en enchaîne plusieurs ou lorsque le flux arrive en morceaux de streaming.

Côté frontend (`frontend/src/hooks/exploreChat.ts`, `QuestionCard.tsx`, `ExplorePanel.tsx`, `ExploreAnonymousPanel.tsx`), la fonction `appendQuestionMessage` marque systématiquement toute question antérieure comme `superseded`, rendant impossible la réponse à plusieurs questions posées lors du même tour. Enfin, `QuestionCard` déclenche un envoi WebSocket immédiat (`buildAnswerWSPayload`) lors de la sélection d'une option et n'offre pas de champ de saisie direct pour les questions libres.

## Goals / Non-Goals

**Goals:**
- Permettre l'extraction et l'émission WebSocket de **toutes** les questions `ghost_question` générées par l'agent dans un même tour.
- Éliminer toute fuite de JSON brut dans le chat par un nettoyage strict côté backend et un filtrage défensif côté frontend.
- Permettre à plusieurs cartes de questions d'être actives et interactives simultanément.
- Proposer une saisie directe sur chaque carte de question avec validation locale (« staging »).
- Matérialiser les réponses préparées sous forme d'encarts temporaires dans la fenêtre de conversation avec options de modification et suppression.
- Assurer un envoi consolidé (réponses préparées + prompt libre) déclenché uniquement par l'action explicite de l'utilisateur.
- Conserver ouvertes pour les tours suivants les questions auxquelles l'utilisateur n'a pas encore répondu.

**Non-Goals:**
- Modifier le format sous-jacent de l'outil natif Claude `AskUserQuestion` (qui utilise `native_question` et `tool_use`), bien que l'interface frontend traite la saisie utilisateur de manière cohérente.
- Modifier le protocole WebSocket existant : le client envoie un message standard `{"type": "user", "message": {"role": "user", "content": "..."}}`.

## Decisions

### 1. Extraction multiple et nettoyage strict (Backend)
- **Extraction complète** : Créer `ExtractAllGhostQuestions(line []byte) []string` qui scanne l'intégralité du texte pour extraire tous les blocs JSON correspondant à `ghost_question`.
- **Émission distincte** : Dans `detectGhostQuestion` (`explore.go`), pour chaque question extraite, émettre un événement WebSocket `{"type":"ghost_question","question":"..."}`.
- **Purge de flux** : `stripGhostQuestionMarker` applique une regex itérative remplaçant toutes les occurrences du marqueur dans les champs textuels des messages consolidés et des `content_block_delta`.
- *Alternative considérée* : Nettoyer uniquement côté frontend. Rejetée car cela polluerait les logs de session stockés sur le disque du backend.

### 2. Gestion des questions sans statut « superseded » abusif (Frontend)
- Dans `exploreChat.ts`, `appendQuestionMessage` ajoute les nouvelles questions sans marquer les questions précédentes non répondues comme `superseded`.
- Plusieurs questions peuvent ainsi rester dans l'état interactif en même temps.

### 3. État des réponses préparées (`stagedAnswers`)
- Introduire un état `stagedAnswers: Record<string, string>` (clé : `question.id`, valeur : texte saisi) dans le cycle de vie du chat d'exploration.
- **Dans chaque `QuestionCard`** :
  - Si la question n'a pas de réponse préparée : un champ texte (et/ou boutons d'options) avec un bouton « Valider » ou touche Entrée enregistre la réponse dans `stagedAnswers`.
  - Si la question a une réponse préparée : la carte affiche un badge « Réponse en attente d'envoi » et le texte préparé.
- **Encarts temporaires dans le fil de discussion** :
  - Positionnés en fin de liste de messages (avant la zone d'input).
  - Pour chaque question présente dans `stagedAnswers` : affichage d'un encart avec le résumé de la question, la réponse saisie, un bouton `Modifier` (qui rouvre l'édition dans la carte ou dans l'encart) et un bouton `Supprimer` (qui retire la clé de `stagedAnswers`).

### 4. Formatage du message consolidé lors de l'envoi
- Au clic sur le bouton « Envoyer » du chat (ou `Entrée` dans le textarea principal) :
  - Si `stagedAnswers` n'est pas vide :
    - On génère le texte structuré :
      ```text
      Réponses aux questions :
      • <Texte question 1> : <Réponse 1>
      • <Texte question 2> : <Réponse 2>

      <Prompt libre de l'utilisateur si renseigné>
      ```
  - Les questions associées aux clés de `stagedAnswers` passent à l'état définitif `answer: text` (cartes réduites en résumé d'historique).
  - Les encarts temporaires sont purgés (`stagedAnswers` réinitialisé à `{}`).
  - Les questions non présentes dans `stagedAnswers` demeurent actives et ouvertes pour les tours de conversation suivants.
  - Le message est envoyé via WebSocket au backend.
- **Activation du bouton Envoyer** : Le bouton Envoyer est activé si `input.trim() !== ''` OU `Object.keys(stagedAnswers).length > 0`.

## Risks / Trade-offs

- **[Fragmentation du streaming]** : Si les tokens d'un marqueur `ghost_question` sont scindés sur plusieurs deltas SSE/stdout, une regex naïve sur chaque delta ne matche pas.
  → *Mitigation* : Côté backend, le gestionnaire assemble les deltas et applique le nettoyage ; côté frontend, on ajoute un filtre de rendu défensif dans le composant Markdown pour masquer tout résidu JSON `{"event":"ghost_question"}` qui aurait pu passer.
- **[Questions multiples et reprise de session]** : Lorsqu'une session redémarre, plusieurs questions peuvent être en attente.
  → *Mitigation* : Le hook recharge l'historique des messages, les questions déjà répondues restent sous forme résumée, et les questions non répondues réapparaissent avec leur champ de saisie prêt à l'emploi.
