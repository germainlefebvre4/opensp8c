import { useTranslation } from 'react-i18next'
import { segments } from '../lib/sidebarModel'
import { STATUS_STYLES } from '../lib/statusColors'

interface Props {
  taskCounts: Record<string, number> | undefined
  total: number
}

export function StatusBar({ taskCounts, total }: Props) {
  const { t } = useTranslation('workspace')
  return (
    <div className="flex items-center gap-2 min-w-0">
      <div
        data-testid="status-bar"
        className="flex flex-1 h-1.5 rounded-full overflow-hidden bg-slate-200 min-w-0"
      >
        {segments(taskCounts).map(seg => (
          <div
            key={seg.status}
            data-testid={`segment-${seg.status}`}
            title={t('statusSegment', { label: t(`status.${seg.status}`), count: seg.count })}
            className={STATUS_STYLES[seg.status].dot}
            style={{ width: `${seg.percent}%` }}
          />
        ))}
      </div>
      <span data-testid="status-total" title={t('total', { count: total })} className="text-[10px] tabular-nums text-slate-400 shrink-0">
        {total}
      </span>
    </div>
  )
}
