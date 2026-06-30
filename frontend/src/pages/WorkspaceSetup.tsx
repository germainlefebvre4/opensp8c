import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAddWorkspace } from '../hooks/useWorkspaces'

export function WorkspaceSetup() {
  const { t } = useTranslation('workspace')

  const [path, setPath] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState<string | null>(null)
  const add = useAddWorkspace()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    try {
      await add.mutateAsync({ path, name: name || undefined })
      setPath('')
      setName('')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', gap: 16 }}>
      <h2>{t('welcome')}</h2>
      <p>{t('welcomeSubtitle')}</p>
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 8, width: 400 }}>
        <input
          type="text"
          placeholder={t('projectPathExample')}
          value={path}
          onChange={e => setPath(e.target.value)}
          required
          style={{ padding: 8, fontSize: 14 }}
        />
        <input
          type="text"
          placeholder={t('projectNamePlaceholder')}
          value={name}
          onChange={e => setName(e.target.value)}
          style={{ padding: 8, fontSize: 14 }}
        />
        {error && <p style={{ color: 'red', margin: 0 }}>{error}</p>}
        <button type="submit" disabled={add.isPending} style={{ padding: 8, cursor: 'pointer' }}>
          {add.isPending ? t('adding') : t('addProject')}
        </button>
      </form>
    </div>
  )
}
