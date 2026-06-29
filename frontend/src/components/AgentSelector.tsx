import { useEffect, useRef, useState } from 'react'
import { Bot, ChevronDown } from 'lucide-react'
import { useAgents, usePatchPreferences, usePreferences } from '../hooks/useAgentPreferences'

export function AgentSelector() {
  const { data: agents = [] } = useAgents()
  const { data: prefs } = usePreferences()
  const patch = usePatchPreferences()
  const [open, setOpen] = useState(false)
  const [dropdownPos, setDropdownPos] = useState<{ top: number; left: number; width: number } | null>(null)
  const ref = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)

  const currentAgent = agents.find(a => a.id === prefs?.defaultAgent)
    ?? agents.find(a => a.id === 'claude')

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  useEffect(() => {
    if (!open) return
    const handler = () => setOpen(false)
    window.addEventListener('resize', handler)
    return () => window.removeEventListener('resize', handler)
  }, [open])

  const handleOpen = () => {
    if (!open && buttonRef.current) {
      const r = buttonRef.current.getBoundingClientRect()
      setDropdownPos({ top: r.bottom + 4, left: r.left, width: r.width })
    }
    setOpen(o => !o)
  }

  const select = (id: string) => {
    patch.mutate({ defaultAgent: id })
    setOpen(false)
  }

  return (
    <div ref={ref} className="px-2 pb-2">
      <button
        ref={buttonRef}
        onClick={handleOpen}
        className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md text-xs font-medium text-slate-600 hover:bg-white hover:text-slate-800 transition-colors border border-slate-200 bg-slate-50"
        title="Choisir l'agent de code"
      >
        <Bot size={12} className="shrink-0 text-slate-400" />
        <span className="flex-1 text-left truncate">
          {currentAgent ? currentAgent.label : 'Agent…'}
        </span>
        {currentAgent?.version && (
          <span className="text-[9px] text-slate-400 shrink-0 font-normal">
            {currentAgent.version}
          </span>
        )}
        <ChevronDown size={11} className="shrink-0 text-slate-400" />
      </button>

      {open && dropdownPos && (
        <div
          style={{ position: 'fixed', top: dropdownPos.top, left: dropdownPos.left, width: dropdownPos.width }}
          className="z-50 bg-white border border-slate-200 rounded-md shadow-md py-1"
        >
          {agents.map(agent => (
            <button
              key={agent.id}
              disabled={!agent.installed}
              onClick={() => agent.installed && select(agent.id)}
              className={`w-full flex items-center gap-2 px-2.5 py-1.5 text-xs text-left transition-colors ${
                agent.id === prefs?.defaultAgent
                  ? 'bg-blue-50 text-blue-700 font-medium'
                  : agent.installed
                    ? 'text-slate-700 hover:bg-slate-50 cursor-pointer'
                    : 'text-slate-300 cursor-not-allowed'
              }`}
            >
              <span className="flex-1 truncate">{agent.label}</span>
              {agent.installed ? (
                <span className="text-[9px] text-slate-400 shrink-0">{agent.version}</span>
              ) : (
                <span className="text-[9px] text-slate-300 shrink-0">non installé</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
