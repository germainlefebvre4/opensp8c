import { useTranslation } from 'react-i18next'
import type { Attention } from '../hooks/useWorkspaces'
import { capSignals } from '../lib/sidebarModel'

interface Props {
  attention: Attention[]
  onOpenChange: (change: string) => void
}

export function AttentionList({ attention, onOpenChange }: Props) {
  const { t } = useTranslation('workspace')
  return (
    <ul className="flex flex-col gap-1 pl-6 pr-1 pb-1.5">
      {attention.map(item => (
        <li key={item.change} className="flex flex-col min-w-0">
          <button
            type="button"
            onClick={() => onOpenChange(item.change)}
            title={t('openChange', { name: item.change })}
            className="text-left text-[11px] font-medium text-slate-700 truncate hover:text-blue-600 cursor-pointer"
          >
            {item.change}
          </button>
          {capSignals(item.signals).map((line, i) =>
            'more' in line ? (
              <span key={`more-${i}`} className="text-[10px] text-slate-400 pl-2">
                {t('moreSignals', { count: line.more })}
              </span>
            ) : (
              <span
                key={i}
                data-testid="signal-line"
                title={line.signal.reason ? `${t(`signal.${line.signal.kind}`)} — ${line.signal.reason}` : undefined}
                className="text-[10px] text-slate-500 pl-2 truncate"
              >
                <span className="font-medium">{t(`signal.${line.signal.kind}`)}</span>
                {line.signal.reason ? ` — ${line.signal.reason}` : ''}
              </span>
            ),
          )}
        </li>
      ))}
    </ul>
  )
}
