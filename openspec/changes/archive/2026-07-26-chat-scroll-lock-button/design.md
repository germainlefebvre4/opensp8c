## Context

Currently, the `ExplorePanel` and `ExploreAnonymousPanel` components enforce a smooth scroll to the bottom on every message or streaming update using a simple `useEffect` hooked to the `messages` and `waiting` states. This behavior causes poor usability because any attempt by the user to scroll up while the agent is typing is immediately hijacked and forced back down.

## Goals / Non-Goals

**Goals:**
- Detect when the user has scrolled up away from the bottom of the message container.
- Disable auto-scrolling when the user is scrolled up (i.e. "scroll lock").
- Render a floating, non-obtrusive, styled "Scroll to bottom" button when scroll lock is active.
- Re-enable auto-scroll and hide the button when:
  1. The user manually scrolls back to the bottom.
  2. The user clicks the floating button.
  3. The user sends a new message in the chat.
- Add English & French localizations for the button tooltip / title.

**Non-Goals:**
- Modifying backend endpoints, WebSocket handlers, or message buffering logic.
- Custom scrollbar styling (rely on existing scroll styles).
- Restructuring the entire exploration panels layout.

## Decisions

### Decision 1: Tracking scroll state using React refs and scroll handlers
We will attach an `onScroll` event listener to the scrollable container and use a `containerRef` to query its actual scroll metrics.
- **Why**: React state transitions are fast and standard. This allows real-time rendering of the floating button based on current scroll position.
- **Alternatives considered**: IntersectionObserver on the bottom ref element. Although possible, IntersectionObserver can be more complex to calibrate during continuous streaming updates than basic scroll position offsets.

### Decision 2: Using a 20-pixel threshold for bottom detection
We will define the user as being "at the bottom" if:
`scrollHeight - scrollTop - clientHeight <= 20`
- **Why**: High-DPI screens, browser scaling/zoom, and sub-pixel rendering can cause `scrollTop` or `clientHeight` to be fractional. A 20px tolerance threshold is robust, preventing edge cases where the user is physically at the bottom but calculations think they are slightly above.

### Decision 3: Wrapping the message container in a relative-positioned div
We will place the scrollable container and the absolute-positioned floating button inside a parent container with class `relative flex-1 min-h-0`.
- **Why**: Placing the floating button inside the scrollable container itself would cause the button to scroll with the messages. Wrapping the scrollable container allows the button to remain floating, fixed at the bottom-right corner of the message area, right above the input panel.

### Decision 4: Using lucide-react's ArrowDown icon
- **Why**: Consistent with the design language of the project (uses `lucide-react` globally).
- **Alternatives considered**: `ChevronDown`. `ArrowDown` is more idiomatic for "scroll down / go to bottom" actions.

## Risks / Trade-offs

- **[Risk]** Smooth scrolling programmatically can trigger continuous `onScroll` events which may cause temporary state changes or re-renders.
  - **Mitigation**: The `onScroll` handler is highly optimized, and setting a single state boolean is lightweight. The 20px threshold ensures that the transitions between "at the bottom" and "scrolled up" are stable.
- **[Risk]** The floating button may cover message text on extremely narrow panels.
  - **Mitigation**: Standardize button size to `p-2 rounded-full` (about `36px` total) and position it at `bottom-4 right-4` with high visual clarity (white bg, subtle border, shadow). The absolute wrapper makes sure it doesn't leak into the bottom input panel.
