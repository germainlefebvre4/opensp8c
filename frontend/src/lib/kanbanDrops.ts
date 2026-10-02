// Maps source status -> allowed drop target statuses (spec: kanban-drag-drop)
export const VALID_DROPS: Record<string, string[]> = {
  'to-explore': ['ready'],
  'ready': ['to-explore', 'todo'],
  'todo': ['ready', 'to-explore'],
  'in-progress': ['to-explore'],
}
