interface Props {
  assistantName?: string
  showLabel?: boolean
}

export function TypingBubble({ assistantName = 'Claude', showLabel = false }: Props) {
  return (
    <div className="self-start bg-slate-100 text-slate-800 max-w-[85%] px-3 py-2 rounded-xl text-sm">
      {showLabel && (
        <p className="text-xs text-slate-500 mb-1.5">{assistantName} réfléchit...</p>
      )}
      <div className="flex gap-1 items-center h-4">
        <span className="typing-dot w-1.5 h-1.5 rounded-full bg-slate-400 inline-block" />
        <span className="typing-dot w-1.5 h-1.5 rounded-full bg-slate-400 inline-block" />
        <span className="typing-dot w-1.5 h-1.5 rounded-full bg-slate-400 inline-block" />
      </div>
    </div>
  )
}
