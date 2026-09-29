# Proposal

## Why

Rouvrir une exploration (ghost) existante relance la conversation : le frontend réinjecte à chaque ouverture tout le transcript stocké en localStorage sous forme d'un message `[Reprise de session]`, et l'agent répond en résumant ce qui a déjà été dit. Cela arrive même quand le backend a simplement rattaché un sous-processus encore vivant qui a déjà tout le contexte. Les sessions anonymes démarrent en effet sans `--resume`, contrairement aux changes nommés.

Par ailleurs, quand un tour d'agent se bloque (sous-processus vivant mais muet, outil qui pend, réponse perdue), l'utilisateur n'a aucun moyen de débloquer la conversation autre que fermer et rouvrir le panneau.

## What Changes

- Les explorations (sessions anonymes) obtiennent une vraie continuité de session : un `claudeSessionId` propre au ghost est persisté, le premier démarrage utilise `--session-id`, les redémarrages utilisent `--resume` (agents Claude et Gemini), avec le même fallback que les changes nommés si `--resume` échoue au démarrage.
- Le frontend cesse d'envoyer le message `[Reprise de session]` à l'ouverture d'un ghost. La réinjection du transcript n'a lieu que lorsque le backend signale un démarrage sans continuité de contexte, via un nouvel événement WebSocket `session_restarted`. Cela couvre le fallback après échec de `--resume` et les agents sans support de reprise (Antigravity, Codex, Copilot).
- Le fil d'exploration ne doit plus afficher deux fois les réponses déjà connues quand le backend rejoue son buffer à la réattache (une seule source de vérité pour l'historique affiché).
- Nouveau bouton « Relancer l'agent » dans les panneaux d'exploration (nommé et anonyme), affiché uniquement quand la session est en attente sans aucune activité depuis 60 s (180 s tant qu'un appel d'outil est en cours). Il redémarre le sous-processus via `--resume`, sans perdre l'historique, et ajoute un message système discret « Agent relancé » dans le fil. Fonctionnalité validée pour les sessions Claude uniquement.
- Une exploration promue en change démarre toujours une nouvelle session : le `claudeSessionId` du ghost n'est pas transmis au change. Ce comportement existe déjà (`runPromoteFF` lance un sous-processus neuf) et est simplement rendu explicite dans la spec.

Non couvert : interruption douce du tour en cours, relance par message « continue », source de vérité backend pour le contexte de promotion (`getStoredContext` reste utilisé par la promotion).

## Capabilities

### New Capabilities
- `explore-session-restart`: détection d'une session d'exploration bloquée (silence prolongé pendant l'attente) et action de redémarrage du sous-processus depuis l'interface, avec message système dans le fil.

### Modified Capabilities
- `explore-session-resume`: les sessions anonymes ne sont plus « non persistées » ; elles portent un `claudeSessionId` et reprennent via `--resume`. La reprise d'une question restée en attente ne dépend plus d'un « replay de contexte » pour les sessions anonymes.
- `exploration-conversation-persistence`: l'injection du contexte localStorage à l'ouverture devient conditionnelle à l'événement `session_restarted` ; le localStorage reste un cache d'affichage et de contexte de promotion, sans doublons avec le replay backend.
- `exploration-promote-to-change`: précise qu'une promotion démarre une nouvelle session indépendante de celle de l'exploration.

## Impact

- Backend : `backend/internal/session/manager.go` (`StartAnonymous`, génération et transmission du `claudeSessionId`, fallback, émission de `session_restarted`), `backend/internal/preferences/preferences.go` (`ExplorationRecord.ClaudeSessionId`), `backend/internal/api/handlers/explore.go` (`createGhostRecord`, rattachement à une session vivante, événements WebSocket).
- Frontend : `useAnonymousExploreSession.ts` (suppression de l'injection à `onopen`, gestion de `session_restarted`, dédoublonnage du replay, `restart`), `useExploreSession.ts` (`restart`, détection de silence), `ExplorePanel.tsx` et `ExploreAnonymousPanel.tsx` (bouton et message système), `i18n` (`locales/fr|en/explore.json`).
- Données : nouveau champ optionnel dans `preferences.json` ; les ghosts existants sans `claudeSessionId` en reçoivent un au prochain démarrage sans migration.
- Aucune nouvelle route HTTP : le redémarrage compose les endpoints existants (`DELETE` de session puis reconnexion).
