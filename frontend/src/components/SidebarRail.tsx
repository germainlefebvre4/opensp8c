import { useTranslation } from 'react-i18next'
import type { Workspace } from '../hooks/useWorkspaces'
import { initials } from '../lib/sidebarModel'

interface Props {
  workspaces: Workspace[]
  activeId: string | null
  onSelect: (id: string) => void
}

export function SidebarRail({ workspaces, activeId, onSelect }: Props) {
  const { t } = useTranslation('workspace')
  return (
    <div className="flex flex-col items-center gap-1.5 py-1 overflow-y-auto">
      {workspaces.map(ws => (
        <button
          key={ws.id}
          type="button"
          title={ws.name}
          aria-label={ws.name}
          aria-current={ws.id === activeId ? 'true' : undefined}
          onClick={() => onSelect(ws.id)}
          className={`relative w-7 h-7 rounded-md text-[10px] font-semibold flex items-center justify-center cursor-pointer transition-colors ${
            ws.id === activeId ? 'bg-blue-100 text-blue-700' : 'bg-slate-200 text-slate-600 hover:bg-slate-300'
          }`}
        >
          {initials(ws.name)}
          {(ws.attention?.length ?? 0) > 0 && (
            <span
              data-testid="attention-dot"
              title={t('attentionDot')}
              className="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-red-500 border border-slate-50"
            />
          )}
        </button>
      ))}
    </div>
  )
}
