export interface ActivityColorInfo {
  color: string
  badgeBg: string
  badgeText: string
  badgeBorder: string
  defaultLabel: string
}

export const ACTIVITY_COLORS: Record<string, ActivityColorInfo> = {
  kanban: {
    color: '#6366f1', // indigo-500
    badgeBg: 'rgba(99, 102, 241, 0.15)',
    badgeText: '#818cf8',
    badgeBorder: 'rgba(99, 102, 241, 0.3)',
    defaultLabel: 'Kanban',
  },
  pool: {
    color: '#f97316', // orange-500
    badgeBg: 'rgba(249, 115, 22, 0.15)',
    badgeText: '#fb923c',
    badgeBorder: 'rgba(249, 115, 22, 0.3)',
    defaultLabel: 'Pool',
  },
  git: {
    color: '#64748b', // slate-500
    badgeBg: 'rgba(100, 116, 139, 0.15)',
    badgeText: '#94a3b8',
    badgeBorder: 'rgba(100, 116, 139, 0.3)',
    defaultLabel: 'Git',
  },
  agent: {
    color: '#10b981', // emerald-500
    badgeBg: 'rgba(16, 185, 129, 0.15)',
    badgeText: '#34d399',
    badgeBorder: 'rgba(16, 185, 129, 0.3)',
    defaultLabel: 'Agent',
  },
  'tool:bash': {
    color: '#f59e0b', // amber-500
    badgeBg: 'rgba(245, 158, 11, 0.15)',
    badgeText: '#fbbf24',
    badgeBorder: 'rgba(245, 158, 11, 0.3)',
    defaultLabel: 'Bash',
  },
  'tool:read': {
    color: '#3b82f6', // blue-500
    badgeBg: 'rgba(59, 130, 246, 0.15)',
    badgeText: '#60a5fa',
    badgeBorder: 'rgba(59, 130, 246, 0.3)',
    defaultLabel: 'Read',
  },
  'tool:edit': {
    color: '#8b5cf6', // purple-500
    badgeBg: 'rgba(139, 92, 246, 0.15)',
    badgeText: '#a78bfa',
    badgeBorder: 'rgba(139, 92, 246, 0.3)',
    defaultLabel: 'Edit',
  },
  'tool:write': {
    color: '#ec4899', // pink-500
    badgeBg: 'rgba(236, 72, 153, 0.15)',
    badgeText: '#f472b6',
    badgeBorder: 'rgba(236, 72, 153, 0.3)',
    defaultLabel: 'Write',
  },
  'tool:other': {
    color: '#06b6d4', // cyan-500
    badgeBg: 'rgba(6, 182, 212, 0.15)',
    badgeText: '#22d3ee',
    badgeBorder: 'rgba(6, 182, 212, 0.3)',
    defaultLabel: 'Tool',
  },
  fallback: {
    color: '#94a3b8', // slate-400
    badgeBg: 'rgba(148, 163, 184, 0.15)',
    badgeText: '#cbd5e1',
    badgeBorder: 'rgba(148, 163, 184, 0.3)',
    defaultLabel: 'Activity',
  },
}

export function getActivityColor(category: string, type?: string): ActivityColorInfo {
  const cat = (category || '').toLowerCase()
  const typ = (type || '').toLowerCase()

  if (cat === 'tool') {
    if (typ === 'bash' || typ.includes('command') || typ === 'exec') {
      return ACTIVITY_COLORS['tool:bash']
    }
    if (typ.includes('read') || typ.includes('view') || typ === 'cat') {
      return ACTIVITY_COLORS['tool:read']
    }
    if (typ.includes('edit') || typ.includes('replace') || typ === 'patch') {
      return ACTIVITY_COLORS['tool:edit']
    }
    if (typ.includes('write') || typ.includes('create')) {
      return ACTIVITY_COLORS['tool:write']
    }
    return ACTIVITY_COLORS['tool:other']
  }

  if (ACTIVITY_COLORS[cat]) {
    return ACTIVITY_COLORS[cat]
  }

  return ACTIVITY_COLORS.fallback
}
