## 1. Core Frontend Changes

- [ ] 1.1 Import `useChanges` in `frontend/src/hooks/useAnonymousExploreSession.ts`
- [ ] 1.2 Call `useChanges` inside `useAnonymousExploreSession`
- [ ] 1.3 Find the matching ghost card from changes using `resumeGhostId`
- [ ] 1.4 Use a `useEffect` to sync the name of the matched ghost card to the `ghostName` state

## 2. Verification and Testing

- [ ] 2.1 Verify that when starting a brand new session, the "create change" button is shown upon sending the first message
- [ ] 2.2 Verify that when resuming an existing named ghost session, the "create change" button appears immediately without waiting for a new message
- [ ] 2.3 Run linter and type-checker to ensure everything is correct and there are no warnings or errors
