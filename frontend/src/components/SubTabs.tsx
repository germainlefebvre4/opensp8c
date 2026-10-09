import type { ReactNode } from 'react'

export interface SubTab<T extends string> {
  id: T
  label: string
  testId?: string
}

interface Props<T extends string> {
  tabs: SubTab<T>[]
  active: T
  onChange: (id: T) => void
  /** Non-interactive info pinned to the right end of the bar. */
  trailing?: ReactNode
  'aria-label'?: string
}

const subTabClass = (active: boolean) =>
  `px-3 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer ${
    active
      ? 'border-blue-600 text-blue-700'
      : 'border-transparent text-slate-500 hover:text-slate-800'
  }`

/** Sub-tab bar shared by the workspace pages; controlled, the page owns the state. */
export function SubTabs<T extends string>({ tabs, active, onChange, trailing, 'aria-label': ariaLabel }: Props<T>) {
  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      className="shrink-0 flex items-center gap-1 px-4 border-b border-slate-200 bg-white"
    >
      {tabs.map(tab => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={tab.id === active}
          data-testid={tab.testId}
          onClick={() => onChange(tab.id)}
          className={subTabClass(tab.id === active)}
        >
          {tab.label}
        </button>
      ))}
      {trailing != null && <div className="ml-auto" data-testid="subtabs-trailing">{trailing}</div>}
    </div>
  )
}
