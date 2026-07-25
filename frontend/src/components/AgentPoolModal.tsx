import { useState } from 'react'
import { X, Play, ShieldAlert, Cpu } from 'lucide-react'

export interface AgentPoolConfig {
  size: number
  delegation_mode: 'full-autonomy' | 'hitl-review'
  max_attempts: number
}

interface Props {
  isOpen: boolean
  onClose: () => void
  onStart: (config: AgentPoolConfig) => void
}

export function AgentPoolModal({ isOpen, onClose, onStart }: Props) {
  const [size, setSize] = useState(3)
  const [mode, setMode] = useState<'full-autonomy' | 'hitl-review'>('hitl-review')

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-slate-900/20 backdrop-blur-sm flex items-center justify-center z-50 p-4 animate-in fade-in duration-200">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-lg overflow-hidden animate-in zoom-in-95 duration-200">
        <div className="flex items-center justify-between p-4 border-b border-slate-100">
          <div className="flex items-center gap-2">
            <Cpu className="w-5 h-5 text-violet-600" />
            <h2 className="text-lg font-semibold text-slate-800">Lancer le Pool d'Agents</h2>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 p-1 rounded-lg hover:bg-slate-100 transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        <div className="p-6 space-y-6">
          <p className="text-sm text-slate-600">
            Vous vous apprêtez à lancer le traitement automatique des cartes de la colonne Todo. Les tâches seront distribuées et traitées en parallèle selon les dépendances.
          </p>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                👥 Nombre de Workers Parallèles
              </label>
              <div className="flex items-center gap-4">
                <button
                  onClick={() => setSize(s => Math.max(1, s - 1))}
                  className="w-10 h-10 rounded-lg border border-slate-200 flex items-center justify-center text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                  disabled={size <= 1}
                >
                  -
                </button>
                <div className="flex-1 text-center font-semibold text-slate-800">
                  {size} Agent{size > 1 ? 's' : ''}
                </div>
                <button
                  onClick={() => setSize(s => Math.min(5, s + 1))}
                  className="w-10 h-10 rounded-lg border border-slate-200 flex items-center justify-center text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                  disabled={size >= 5}
                >
                  +
                </button>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                🛡️ Mode de Délégation global
              </label>
              
              <div className="space-y-3">
                <label className={`flex p-3 rounded-xl border-2 cursor-pointer transition-colors ${mode === 'full-autonomy' ? 'border-violet-600 bg-violet-50' : 'border-slate-200 hover:border-slate-300'}`}>
                  <input
                    type="radio"
                    name="delegation"
                    value="full-autonomy"
                    checked={mode === 'full-autonomy'}
                    onChange={() => setMode('full-autonomy')}
                    className="sr-only"
                  />
                  <div className="flex-1 ml-2">
                    <div className="font-medium text-slate-900">Autonomie Totale</div>
                    <div className="text-xs text-slate-500 mt-1">
                      Les agents modifient le code principal et déplacent les cartes directement dans DONE.
                    </div>
                  </div>
                </label>

                <label className={`flex p-3 rounded-xl border-2 cursor-pointer transition-colors ${mode === 'hitl-review' ? 'border-violet-600 bg-violet-50' : 'border-slate-200 hover:border-slate-300'}`}>
                  <input
                    type="radio"
                    name="delegation"
                    value="hitl-review"
                    checked={mode === 'hitl-review'}
                    onChange={() => setMode('hitl-review')}
                    className="sr-only"
                  />
                  <div className="flex-1 ml-2">
                    <div className="font-medium text-slate-900">Développement avec Review (HITL)</div>
                    <div className="text-xs text-slate-500 mt-1">
                      Les agents travaillent sur des branches isolées. Les cartes terminées attendent votre validation dans la colonne TO REVIEW avant fusion Git.
                    </div>
                  </div>
                  <ShieldAlert className={`w-5 h-5 ${mode === 'hitl-review' ? 'text-violet-600' : 'text-slate-400'}`} />
                </label>
              </div>
            </div>
          </div>
        </div>

        <div className="p-4 bg-slate-50 border-t border-slate-100 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-200 rounded-lg transition-colors cursor-pointer"
          >
            Annuler
          </button>
          <button
            onClick={() => {
              onStart({ size, delegation_mode: mode, max_attempts: 3 })
              onClose()
            }}
            className="flex items-center gap-2 px-6 py-2 bg-violet-600 hover:bg-violet-700 text-white text-sm font-medium rounded-lg shadow-sm transition-colors cursor-pointer"
          >
            <Play size={16} />
            Démarrer le Pool
          </button>
        </div>
      </div>
    </div>
  )
}
