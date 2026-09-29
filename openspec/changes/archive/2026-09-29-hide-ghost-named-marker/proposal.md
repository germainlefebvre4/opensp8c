# Proposal

## Why

Dans un chat d'exploration anonyme, le marker de nommage `{"event":"ghost_named","name":"..."}` s'affiche en JSON brut dans la bulle assistant. Le backend ne retire que `ghost_question` du texte relayé, et le streaming fragmente le marker en plusieurs deltas (`{"`, `event"`, …) qu'aucune regex appliquée delta par delta ne peut reconnaître. Le nom est déjà visible dans le header et sur la carte kanban : le JSON n'apporte rien et dégrade l'expérience.

## What Changes

- Le marker `ghost_named` n'est plus jamais rendu dans le fil, qu'il arrive en un bloc, mélangé à du texte ou fragmenté sur plusieurs deltas de streaming.
- Le nettoyage des markers devient tolérant à la fragmentation (détection avec état sur le flux de texte), côté backend en priorité et côté frontend en filet de sécurité.
- Lorsque le nom est attribué, une notice système discrète (« Exploration nommée : `xxx` ») est insérée dans le fil, distincte d'un tour assistant, conservée avec l'historique et sans doublon à la reprise de session.
- Le nom affiché dans la notice est le nom final (suffixe de collision inclus).
- Le mécanisme de nettoyage est écrit de façon générique pour que `ghost_question` et `ghost_draft` puissent l'adopter plus tard ; leur migration est hors périmètre.

## Capabilities

### New Capabilities

### Modified Capabilities
- `explore-ghost-card`: le marker `ghost_named` ne doit jamais fuiter dans le texte affiché, même fragmenté ; une notice de nommage est affichée.
- `explore-message-layout`: nouveau type de ligne « notice système » dans le flux plein-largeur, rendu identiquement dans les deux panels d'exploration.

## Impact

- Backend : `backend/internal/api/handlers/explore.go` (relais des messages, `detectGhostQuestion`/stripping), `backend/internal/session/manager.go` (extraction `ghost_named`).
- Frontend : `frontend/src/hooks/exploreChat.ts` (type `Message`, `mergeAssistantText`, nettoyage résiduel), `useAnonymousExploreSession.ts`, `ExploreAnonymousPanel.tsx`, `ExplorePanel.tsx`, fichiers i18n `fr`/`en`.
- Docs : `docs/opensp8c/architecture.md` (section markers).
- Aucun changement d'API externe ; le protocole WS `ghost_named` est inchangé.
