import { useEffect, useRef, useState } from 'react'
import { X, Code, Eye } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { useAnonymousExploreSession } from '../hooks/useAnonymousExploreSession'
import { useExploreViewMode } from '../hooks/useExploreViewMode'

interface Props {
  workspaceId: string
  onClose: () => void
  onPromoted: (name: string) => void
}

export function ExploreAnonymousPanel({ workspaceId, onClose, onPromoted }: Props) {
  const { messages, connected, expired, promotedName, send, stop } = useAnonymousExploreSession(workspaceId)
  const { mode, setMode } = useExploreViewMode()
  const [input, setInput] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  useEffect(() => {
    if (promotedName) {
      onPromoted(promotedName)
    }
  }, [promotedName, onPromoted])

  const handleSend = (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim()) return
    send(input.trim())
    setInput('')
  }

  return (
    <div className="h-full flex flex-col bg-white">
      <div className="px-4 py-3 border-b border-slate-200 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm font-semibold text-slate-800 truncate">
            {promotedName ? `Explorer : ${promotedName}` : 'Nouvelle exploration'}
          </span>
          <span className={`text-[10px] font-semibold shrink-0 ${connected ? 'text-emerald-500' : 'text-amber-500'}`}>
            {connected ? '● connecté' : '○ déconnecté'}
          </span>
        </div>
        <div className="flex items-center gap-1.5 shrink-0 ml-2">
          <div className="flex items-center gap-0.5 bg-slate-100 rounded-md p-0.5">
            <button
              onClick={() => setMode('raw')}
              title="Texte brut"
              className={`p-1 rounded transition-colors cursor-pointer ${mode === 'raw' ? 'bg-white text-slate-700 shadow-sm' : 'text-slate-400 hover:text-slate-600'}`}
            >
              <Code size={13} />
            </button>
            <button
              onClick={() => setMode('rendered')}
              title="Markdown rendu"
              className={`p-1 rounded transition-colors cursor-pointer ${mode === 'rendered' ? 'bg-white text-slate-700 shadow-sm' : 'text-slate-400 hover:text-slate-600'}`}
            >
              <Eye size={13} />
            </button>
          </div>
          <button
            onClick={() => { stop(); onClose() }}
            className="p-1 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors"
          >
            <X size={15} />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-3">
        {messages.map((msg, i) => (
          <div
            key={i}
            className={`max-w-[85%] px-3 py-2 rounded-xl text-sm break-words ${
              msg.role === 'user'
                ? 'self-end bg-blue-600 text-white whitespace-pre-wrap'
                : 'self-start bg-slate-100 text-slate-800'
            }`}
          >
            {msg.role === 'assistant' && mode === 'rendered' ? (
              <article className="prose prose-slate prose-sm max-w-none text-left">
                <ReactMarkdown>{msg.content}</ReactMarkdown>
                {msg.partial && <span className="opacity-50">▊</span>}
              </article>
            ) : (
              <span className="whitespace-pre-wrap">
                {msg.content}
                {msg.partial && <span className="opacity-50">▊</span>}
              </span>
            )}
          </div>
        ))}
        {expired && (
          <div className="text-center text-amber-600 text-xs">Session expirée.</div>
        )}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={handleSend} className="p-3 border-t border-slate-200 flex gap-2 shrink-0">
        <input
          type="text"
          value={input}
          onChange={e => setInput(e.target.value)}
          placeholder="Décrivez ce que vous voulez explorer..."
          disabled={!connected}
          className="flex-1 px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50 placeholder:text-slate-400"
        />
        <button
          type="submit"
          disabled={!connected || !input.trim()}
          className="px-3 py-2 text-xs font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed shrink-0"
        >
          Envoyer
        </button>
      </form>
    </div>
  )
}
