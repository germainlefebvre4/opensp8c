## 1. Backend — Timeout sur la détection des agents

- [x] 1.1 Dans `backend/internal/agents/agents.go`, modifier `Detect()` pour utiliser `exec.CommandContext` avec un `context.WithTimeout` de 3 secondes au lieu de `exec.Command`

## 2. Frontend — Dropdown position: fixed

- [x] 2.1 Dans `AgentSelector.tsx`, ajouter un `ref` sur le `<button>` (`buttonRef`)
- [x] 2.2 Remplacer `onClick={() => setOpen(o => !o)}` par un handler `handleOpen` qui calcule `getBoundingClientRect()` et stocke `{ top, left, width }` dans un state `dropdownPos`
- [x] 2.3 Remplacer `className="absolute left-2 right-2 top-full mt-1 z-50 ..."` par un `style` inline avec `position: fixed`, `top: dropdownPos.top`, `left: dropdownPos.left`, `width: dropdownPos.width`
- [x] 2.4 Supprimer `relative` du className du wrapper `div` parent (plus nécessaire avec `position: fixed`)
- [x] 2.5 Ajouter un `useEffect` qui écoute `window.resize` et ferme le dropdown (`setOpen(false)`)
