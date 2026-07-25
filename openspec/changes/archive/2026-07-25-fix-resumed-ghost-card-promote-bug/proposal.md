## Why

Lorsqu'un utilisateur reprend une session d'exploration (ghost card) après un redémarrage de l'application, le bouton de création de change n'apparaît plus, car l'état local `ghostName` dans le frontend est réinitialisé à `null` et n'est jamais reconstitué à partir du nom persistant du ghost card.

## What Changes

- Récupérer et restaurer réactivement le nom réel du ghost card lors de la reprise d'une session d'exploration anonyme en s'appuyant sur la liste des changes existants (`useChanges`).
- S'assurer que le bouton de création de change/promotion s'affiche correctement dès la reprise de session.

## Capabilities

### New Capabilities

- None

### Modified Capabilities

- `explore-ghost-card`: Préciser la gestion de la restauration du nom du ghost card lors de la reprise de session.
- `explore-session-resume`: S'assurer que l'état d'exploration (y compris le nom du ghost card) est entièrement restauré au chargement.

## Impact

- `frontend/src/hooks/useAnonymousExploreSession.ts`
- `frontend/src/components/ExploreAnonymousPanel.tsx`
