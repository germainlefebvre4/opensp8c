## 1. Frontend Implementation

- [x] 1.1 Add `partial: false` explicitly to `session_warning` message insertions in both `frontend/src/hooks/useExploreSession.ts` and `frontend/src/hooks/useAnonymousExploreSession.ts`
- [x] 1.2 Update the text delta stream-appending conditions in both hooks to require `last.partial` rather than using the broader `(last.partial || isPartial)`

## 2. Validation

- [x] 2.1 Trigger an IDE companion extension connection error (by closing the IDE extension) and verify that the French warning text is rendered as a standalone message block
- [x] 2.2 Submit a follow-up query after the warning and verify that the assistant's streamed response renders in a completely separate message block
- [x] 2.3 Confirm that standard explore conversations (where no warning is present) continue to type smoothly and do not produce split message blocks
