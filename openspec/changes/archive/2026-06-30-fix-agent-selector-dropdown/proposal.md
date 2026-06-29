## Why

Le dropdown `AgentSelector` est inutilisable : il est clippé par le `overflow-hidden` du sidebar, ne rendant visible qu'une fine barre blanche. La position est également décalée à cause du `pb-2` sur le containing block.

## What Changes

- Le dropdown passe de `position: absolute` à `position: fixed` avec coordonnées calculées via `getBoundingClientRect()` au moment de l'ouverture
- Suppression de `relative` sur le wrapper parent de `AgentSelector` (plus nécessaire)
- Ajout d'un `ref` sur le `<button>` pour lire ses coordonnées
- Ajout d'un `useEffect` pour fermer le dropdown sur `window.resize`
- Correction du timeout manquant dans `agents.DetectAll()` : utiliser `exec.CommandContext` avec un timeout de 3s pour éviter que `gh copilot --version` bloque indéfiniment le endpoint `/api/agents`

## Capabilities

### New Capabilities

- `agent-selector-dropdown`: Dropdown de sélection d'agent correctement positionné et visible, non clippé par les ancêtres `overflow-hidden`

### Modified Capabilities

## Impact

- `frontend/src/components/AgentSelector.tsx` : refactoring du positionnement du dropdown
- `backend/internal/agents/agents.go` : ajout d'un timeout sur `exec.Command` dans `Detect()`
