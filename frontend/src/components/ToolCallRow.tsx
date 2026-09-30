import { useState } from 'react'
import { ChevronRight, FileText, FilePen, Pencil, Search, Terminal, FolderSearch, Wrench, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { ToolCall } from '../hooks/exploreChat'

const TOOL_ICONS: Record<string, typeof Wrench> = {
  Read: FileText,
  Write: FilePen,
  Edit: Pencil,
  Grep: Search,
  Bash: Terminal,
  Glob: FolderSearch,
}

interface Props {
  toolCall: ToolCall
}

export function ToolCallRow({ toolCall }: Props) {
  const { t } = useTranslation('explore')
  const [expanded, setExpanded] = useState(false)
  const Icon = TOOL_ICONS[toolCall.name] ?? Wrench

  return (
    <div className="border border-slate-200 rounded-lg bg-slate-50 text-xs overflow-hidden">
      <button
        type="button"
        onClick={() => setExpanded(prev => !prev)}
        className="w-full flex items-center gap-1.5 px-2 py-1.5 text-left text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
      >
        <ChevronRight size={12} className={`shrink-0 text-slate-400 transition-transform ${expanded ? 'rotate-90' : ''}`} />
        <Icon size={13} className="shrink-0 text-slate-500" />
        <span className="font-medium text-slate-700 shrink-0">{toolCall.name}</span>
        {toolCall.target && <span className="truncate text-slate-500">{toolCall.target}</span>}
        {toolCall.status === 'pending' && (
          <Loader2 size={12} className="shrink-0 ml-auto text-slate-400 animate-spin" />
        )}
      </button>
      {expanded && (
        <div className="px-2.5 pb-2 pt-0.5 border-t border-slate-200 text-slate-500">
          {toolCall.input && Object.keys(toolCall.input).length > 0 && (
            <div className="mb-1.5">
              <div className="text-[10px] font-medium uppercase tracking-wider text-slate-400">{t('toolCall.input')}</div>
              <pre data-testid="tool-input" className="whitespace-pre-wrap break-words font-mono text-[11px]">{JSON.stringify(toolCall.input, null, 2)}</pre>
            </div>
          )}
          {toolCall.status === 'pending' ? (
            <span className="italic">{t('toolCall.pending')}</span>
          ) : (
            <div>
              {toolCall.input && <div className="text-[10px] font-medium uppercase tracking-wider text-slate-400">{t('toolCall.result')}</div>}
              <pre data-testid="tool-result" className="whitespace-pre-wrap break-words font-sans">{toolCall.resultPreview || t('toolCall.noPreview')}</pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
