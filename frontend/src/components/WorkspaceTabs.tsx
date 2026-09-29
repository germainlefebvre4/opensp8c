import { NavLink, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export function WorkspaceTabs() {
  const { t } = useTranslation('navigation')
  const [searchParams] = useSearchParams()

  return (
    <nav className="border-b border-slate-200 px-4 flex items-center h-11 gap-0 shrink-0">
      {([
        { path: '/', label: t('kanban') },
        { path: '/specs', label: t('specs') },
        { path: '/timeline', label: t('timeline') },
        { path: '/agents', label: t('agents') },
        { path: '/settings', label: t('settings') },
      ] as const).map(({ path, label }) => (
        <NavLink
          key={path}
          to={{ pathname: path, search: searchParams.toString() }}
          end
          className={({ isActive }) =>
            `px-3 h-full flex items-center text-xs font-medium border-b-2 transition-colors no-underline ${
              isActive
                ? 'text-blue-600 border-blue-600'
                : 'text-slate-500 border-transparent hover:text-slate-700 hover:border-slate-300'
            }`
          }
        >
          {label}
        </NavLink>
      ))}
    </nav>
  )
}
