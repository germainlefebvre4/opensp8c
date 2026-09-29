import { Link, NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AgentSelector } from './AgentSelector'

interface Props {
  active: 'projects' | 'configuration'
  projectsTo: string
}

const itemClass = (isActive: boolean) =>
  `px-3 h-full flex items-center text-xs font-medium border-b-2 transition-colors no-underline ${
    isActive
      ? 'text-blue-600 border-blue-600'
      : 'text-slate-500 border-transparent hover:text-slate-700 hover:border-slate-300'
  }`

export function TopBar({ active, projectsTo }: Props) {
  const { t } = useTranslation('navigation')

  return (
    <header className="border-b border-slate-200 px-4 flex items-center h-11 gap-0 shrink-0">
      <span className="text-[11px] font-bold uppercase tracking-widest text-slate-400 mr-4 select-none">
        OpenSpec
      </span>
      <Link to={projectsTo} className={itemClass(active === 'projects')}>
        {t('projects')}
      </Link>
      <NavLink to="/configuration" end className={itemClass(active === 'configuration')}>
        {t('configuration')}
      </NavLink>
      <div className="ml-auto">
        <AgentSelector />
      </div>
    </header>
  )
}
