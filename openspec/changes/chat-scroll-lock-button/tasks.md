## 1. Setup and Localization

- [x] 1.1 Add `scrollToBottom` translation key to the English locale file (`frontend/src/locales/en/explore.json`)
- [x] 1.2 Add `scrollToBottom` translation key to the French locale file (`frontend/src/locales/fr/explore.json`)

## 2. Named Exploration Session Scroll Logic

- [x] 2.1 Update imports in `frontend/src/components/ExplorePanel.tsx` to include `ArrowDown` from `lucide-react`
- [x] 2.2 Add `containerRef` and `isAtBottom` state inside `ExplorePanel.tsx`
- [x] 2.3 Implement the `handleScroll` and `scrollToBottom` callbacks inside `ExplorePanel.tsx`
- [x] 2.4 Update the streaming `useEffect` to use the `isAtBottom` condition, and update `handleSend` to force scrolling
- [x] 2.5 Wrap the message list in a relative div, attach `containerRef` and `onScroll` to the list, and render the absolute floating scroll button

## 3. Anonymous Exploration Session Scroll Logic

- [x] 3.1 Update imports in `frontend/src/components/ExploreAnonymousPanel.tsx` to include `ArrowDown` from `lucide-react`
- [x] 3.2 Add `containerRef` and `isAtBottom` state inside `ExploreAnonymousPanel.tsx`
- [x] 3.3 Implement the `handleScroll` and `scrollToBottom` callbacks inside `ExploreAnonymousPanel.tsx`
- [x] 3.4 Update the streaming `useEffect` to use the `isAtBottom` condition, and update `handleSend` to force scrolling
- [x] 3.5 Wrap the message list in a relative div, attach `containerRef` and `onScroll` to the list, and render the absolute floating scroll button

## 4. Validation and Testing

- [x] 4.1 Run the frontend linter and verify there are no TypeScript compile or lint errors
- [x] 4.2 Verify correct rendering and interaction of the button and scroll lock in both named and anonymous explore views
