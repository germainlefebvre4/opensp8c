## Why

The Atlassian Design System (ADS) has a highly recognizable palette that instantly gives a professional, "pro tool" feel to an application. Currently, the application uses Tailwind CSS v4 with an alias system in `frontend/src/index.css` that maps `slate` to `zinc` and `blue` to `emerald`. Migrating these aliases to Atlassian's official hex codes will elevate the aesthetic to match Jira without requiring changes to the React components themselves.

## What Changes

- Replace the `slate` color palette in `frontend/src/index.css` with Atlassian Neutral colors (N10 -> N900).
- Replace the `blue` color palette in `frontend/src/index.css` with Atlassian Blue brand colors (B50 -> B500).
- All buttons, badges, panels, and text using these Tailwind classes will automatically inherit the new Jira-like aesthetic.

## Capabilities

### New Capabilities
- `ui-theme-jira`: Introduces the Jira/Atlassian Design System color palette using Tailwind variable overrides for neutrals (slate) and brand colors (blue).

### Modified Capabilities

## Impact

- **Frontend Styling:** `frontend/src/index.css` will be modified.
- **Components:** Visual appearance of existing UI components will change globally, but no React component code will need to be rewritten.
