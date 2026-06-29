import { useCallback, useEffect, useRef, useState } from 'react'
import { X, Code, Eye } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { useExploreSession } from '../hooks/useExploreSession'
import { useExploreViewMode } from '../hooks/useExploreViewMode'
import { TypingBubble } from './TypingBubble'

interface Props {
  workspaceId: string
  changeName: string
  onClose: () => void
}

export function ExplorePanel({ workspaceId, changeName, onClose }: Props) {
  const { messages, connected, expired, waiting, agentInfo, send, reconnect } = useExploreSession(workspaceId, changeName)
  const { mode, setMode } = useExploreViewMode()
  const [input, setInput] = useState('')
  const [showSlowLabel, setShowSlowLabel] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, waiting])

  useEffect(() => {
    if (!waiting) {
      setShowSlowLabel(false)
      return
    }
    const timer = setTimeout(() => setShowSlowLabel(true), 5000)
    return () => clearTimeout(timer)
  }, [waiting])

  useEffect(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = el.scrollHeight + 'px'
  }, [input])

  const handleSend = useCallback(() => {
    if (!input.trim()) return
    send(input.trim())
    setInput('')
  }, [input, send])

  const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }, [handleSend])

  return (
    <div className="h-full flex flex-col bg-white">
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-200 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm font-semibold text-slate-800 truncate">Explorer : {changeName}</span>
          <span className={`text-[10px] font-semibold shrink-0 ${connected ? 'text-emerald-500' : 'text-amber-500'}`}>
            {connected ? '● connecté' : '○ déconnecté'}
          </span>
          {agentInfo && (
            <span className="text-[10px] font-medium text-slate-400 shrink-0 bg-slate-100 px-1.5 py-0.5 rounded">
              {agentInfo.label}{agentInfo.version ? ` ${agentInfo.version}` : ''}
            </span>
          )}
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
            onClick={onClose}
            className="p-1 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors"
          >
            <X size={15} />
          </button>
        </div>
      </div>

      {/* Messages */}
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
        {waiting && <TypingBubble assistantName={agentInfo?.label ?? 'Claude'} showLabel={showSlowLabel} />}
        {expired && (
          <div className="text-center text-amber-600 text-xs">
            Session expirée.{' '}
            <button
              onClick={reconnect}
              className="text-blue-600 underline cursor-pointer bg-transparent border-0 text-xs"
            >
              Reprendre
            </button>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="p-3 border-t border-slate-200 flex gap-2 shrink-0 items-end">
        <textarea
          ref={textareaRef}
          rows={1}
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Votre message... (Shift+Enter pour aller à la ligne)"
          disabled={!connected}
          className="flex-1 px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50 placeholder:text-slate-400 resize-none overflow-y-auto"
          style={{ maxHeight: '160px' }}
        />
        <button
          onClick={handleSend}
          disabled={!connected || !input.trim()}
          className="px-3 py-2 text-xs font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed shrink-0"
        >
          Envoyer
        </button>
      </div>
    </div>
  )
}
