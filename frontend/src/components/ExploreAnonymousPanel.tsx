import { useCallback, useEffect, useRef, useState } from 'react'
import { X, Code, Eye, Trash2, Maximize2, Minimize2, Sparkles } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { useTranslation } from 'react-i18next'
import { useAnonymousExploreSession } from '../hooks/useAnonymousExploreSession'
import { useExploreViewMode } from '../hooks/useExploreViewMode'
import { TypingBubble } from './TypingBubble'
import { DraftSidePanel } from './DraftSidePanel'

interface Props {
  workspaceId: string
  resumeGhostId?: string
  isMaximized?: boolean
  onMaximizeToggle?: () => void
  onClose: () => void
  onDelete?: () => void
  onGhostReady?: (ghostId: string) => void
  onPromote?: () => void
}

export function ExploreAnonymousPanel({ workspaceId, resumeGhostId, isMaximized, onMaximizeToggle, onClose, onDelete, onGhostReady, onPromote }: Props) {
  const { t } = useTranslation('explore')
  const { messages, connected, expired, waiting, ghostId, ghostName, agentInfo, send, stop } = useAnonymousExploreSession(workspaceId, resumeGhostId)
  const { mode, setMode } = useExploreViewMode()
  const [input, setInput] = useState('')
  const [showSlowLabel, setShowSlowLabel] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, waiting])

  useEffect(() => {
    if (ghostId) onGhostReady?.(ghostId)
  }, [ghostId, onGhostReady])

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

  const displayName = ghostName ?? resumeGhostId ?? null

  return (
    <div className="h-full flex flex-col bg-white @container">
      <div className="px-4 py-3 border-b border-slate-200 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm font-semibold text-slate-800 truncate">
            {displayName ? t('headerExplore', { name: displayName }) : t('headerNew')}
          </span>
          <span className={`text-[10px] font-semibold shrink-0 ${connected ? 'text-emerald-500' : 'text-amber-500'}`}>
            {connected ? t('connected') : t('disconnected')}
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
              title={t('rawText', { ns: 'detailPanel' })}
              className={`p-1 rounded transition-colors cursor-pointer ${mode === 'raw' ? 'bg-white text-slate-700 shadow-sm' : 'text-slate-400 hover:text-slate-600'}`}
            >
              <Code size={13} />
            </button>
            <button
              onClick={() => setMode('rendered')}
              title={t('markdownRendered', { ns: 'detailPanel' })}
              className={`p-1 rounded transition-colors cursor-pointer ${mode === 'rendered' ? 'bg-white text-slate-700 shadow-sm' : 'text-slate-400 hover:text-slate-600'}`}
            >
              <Eye size={13} />
            </button>
          </div>
          {ghostId && ghostName && onPromote && (
            <button
              onClick={onPromote}
              title={t('createChangeTooltip')}
              className="flex items-center gap-1 px-2 py-1 rounded bg-violet-50 text-violet-700 hover:bg-violet-100 transition-colors cursor-pointer border border-violet-100 text-xs font-semibold"
            >
              <Sparkles size={13} className="text-violet-500 shrink-0" />
              <span className="hidden @[350px]:inline">{t('createChange')}</span>
            </button>
          )}
          {ghostId && onDelete && (
            <button
              onClick={onDelete}
              title={t('deleteExploration')}
              className="p-1 rounded-md text-slate-400 hover:text-red-500 hover:bg-red-50 transition-colors cursor-pointer"
            >
              <Trash2 size={15} />
            </button>
          )}
          {onMaximizeToggle && (
            <button
              onClick={onMaximizeToggle}
              title={isMaximized ? t('minimize') : t('maximize')}
              className="p-1 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
            >
              {isMaximized ? <Minimize2 size={15} /> : <Maximize2 size={15} />}
            </button>
          )}
          <button
            onClick={() => { stop(); onClose() }}
            className="p-1 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
          >
            <X size={15} />
          </button>
        </div>
      </div>

      {/* Main Split Area */}
      <div className="flex-1 min-h-0 flex flex-row">
        {/* Left Pane: Chat Conversation */}
        <div className="flex-1 min-w-0 flex flex-col h-full">
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
              <div className="text-center text-amber-600 text-xs">{t('sessionExpired')}</div>
            )}
            <div ref={bottomRef} />
          </div>

          <div className="p-3 border-t border-slate-200 flex gap-2 shrink-0 items-end">
            <textarea
              ref={textareaRef}
              rows={1}
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={t('anonymousPlaceholder')}
              disabled={!connected}
              className="flex-1 px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50 placeholder:text-slate-400 resize-none overflow-y-auto"
              style={{ maxHeight: '160px' }}
            />
            <button
              onClick={handleSend}
              disabled={!connected || !input.trim()}
              className="px-3 py-2 text-xs font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed shrink-0"
            >
              {t('send')}
            </button>
          </div>
        </div>

        {/* Right Pane: Interactive Draft Checklist */}
        {ghostId && (
          <div className="w-[320px] shrink-0 h-full">
            <DraftSidePanel workspaceId={workspaceId} ghostId={ghostId} />
          </div>
        )}
      </div>
    </div>
  )
}
