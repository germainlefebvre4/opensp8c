## Why

During a chat exploration session, as the agent streams its response, the viewport automatically forces a scroll to the bottom. This hijacks the user's focus and makes it impossible to scroll up to review previous parts of the conversation or look at previous messages while the response is in progress. Adding a "scroll lock" behavior and a floating "scroll to bottom" button resolves this friction and significantly improves the navigation experience.

## What Changes

- **Scroll Lock during streaming**: If the user manually scrolls up while the agent is typing/streaming a response, the viewport will freeze at the user's scroll position instead of being forced back down to the bottom.
- **Auto-scroll when at the bottom**: If the user remains scrolled at the bottom, the chat will continue to automatically scroll down as new tokens/messages arrive to show the live response in real time.
- **Scroll to Bottom Button**: A floating button with a down arrow will appear at the bottom-right of the messages area when the user has scrolled up (and is not at the bottom). Clicking this button will smoothly scroll the view to the bottom and re-enable the auto-scroll behavior.
- **Envoi de message forces scroll**: Sending a message will automatically reset the scroll position to the bottom and re-enable auto-scroll.

## Capabilities

### New Capabilities
<!-- None -->

### Modified Capabilities
- `explore-session`: Update the named exploration panel (`ExplorePanel.tsx`) to implement the scroll lock behavior and the scroll down floating button.
- `anonymous-explore-session`: Update the anonymous exploration panel (`ExploreAnonymousPanel.tsx`) to implement the scroll lock behavior and the scroll down floating button.

## Impact

- **Frontend Components**: `ExplorePanel.tsx` and `ExploreAnonymousPanel.tsx` will be modified to introduce scroll tracking (`onScroll` handler, `isAtBottom` state) and render the floating button.
- **Assets/Icons**: Import and render the `ArrowDown` icon from `lucide-react`.
- **Localization**: Add `scrollToBottom` localization keys for both English and French translation files.
