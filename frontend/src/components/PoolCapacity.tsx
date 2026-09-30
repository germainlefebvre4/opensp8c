import type { PoolStatus } from '../hooks/usePoolStatus'

interface Props {
  status: PoolStatus | undefined
}

export function PoolCapacity({ status }: Props) {
  if (!status?.is_running) return null
  return (
    <span data-testid="pool-capacity" className="text-xs font-semibold">
      {status.workers.length}/{status.config.size}
    </span>
  )
}
