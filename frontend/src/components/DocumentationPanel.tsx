import { useEffect, useState } from 'react'
import * as ScrollArea from '@radix-ui/react-scroll-area'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { useDocPage, useDocs } from '../hooks/useDocs'
import { triggerDocsGenerate } from '../lib/api'
import { Markdown } from './Markdown'

interface Props {
  workspaceId: string
  generating: boolean
}

export function DocumentationPanel({ workspaceId, generating }: Props) {
  const { t } = useTranslation('specs')
  const { t: tCommon } = useTranslation('common')
  const qc = useQueryClient()

  const { data, isLoading } = useDocs(workspaceId)
  const pages = data?.pages ?? []
  const isStale = data?.is_stale ?? false
  const isGenerating = generating || (data?.generating ?? false)

  const [selectedPage, setSelectedPage] = useState<string | null>(null)
  const { data: pageDetail } = useDocPage(workspaceId, selectedPage)

  useEffect(() => {
    if (pages.length === 0) {
      if (selectedPage !== null) setSelectedPage(null)
      return
    }
    if (!selectedPage || !pages.includes(selectedPage)) {
      setSelectedPage(pages[0])
    }
  }, [pages, selectedPage])

  const handleGenerate = async () => {
    if (isGenerating) return
    await triggerDocsGenerate(workspaceId)
    qc.invalidateQueries({ queryKey: ['docs', workspaceId] })
  }

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center text-sm text-slate-400">
        {tCommon('loading')}
      </div>
    )
  }

  if (pages.length === 0) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center gap-3">
        <p className="text-sm text-slate-400">{t('docs.emptyState')}</p>
        <button
          onClick={handleGenerate}
          disabled={isGenerating}
          className="px-4 py-2 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-md transition-colors"
        >
          {isGenerating ? t('docs.generating') : t('docs.generate')}
        </button>
      </div>
    )
  }

  return (
    <div className="flex-1 flex overflow-hidden">
      <aside className="w-44 shrink-0 border-r border-slate-200 bg-slate-50 flex flex-col">
        <div className="px-4 pt-4 pb-2 shrink-0">
          <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-400">
            {t('docs.title')}
          </span>
        </div>
        <div className="px-3 pb-2 shrink-0 flex flex-col gap-1.5">
          <button
            onClick={handleGenerate}
            disabled={isGenerating}
            className="w-full px-2.5 py-1.5 text-xs font-medium text-slate-600 hover:text-slate-800 hover:bg-white disabled:opacity-50 disabled:cursor-not-allowed border border-slate-200 rounded-md transition-colors"
          >
            {isGenerating ? t('docs.generating') : t('docs.regenerate')}
          </button>
          {isStale && (
            <span className="text-[10px] text-amber-500 font-medium" title={t('docs.staleTooltip')}>
              ⚠ {t('docs.stale')}
            </span>
          )}
        </div>
        <ScrollArea.Root className="flex-1 overflow-hidden">
          <ScrollArea.Viewport className="h-full w-full">
            <div className="px-2 pb-2 flex flex-col gap-0.5">
              {pages.map(page => (
                <button
                  key={page}
                  onClick={() => setSelectedPage(page)}
                  className={`w-full text-left px-2.5 py-2 rounded-md text-xs cursor-pointer transition-colors truncate ${
                    page === selectedPage
                      ? 'bg-blue-50 text-blue-700 font-semibold'
                      : 'text-slate-600 hover:bg-white hover:text-slate-800 font-medium'
                  }`}
                >
                  {t(`docs.pages.${page}`)}
                </button>
              ))}
            </div>
          </ScrollArea.Viewport>
          <ScrollArea.Scrollbar orientation="vertical" className="flex w-1.5 touch-none select-none p-0.5">
            <ScrollArea.Thumb className="relative flex-1 rounded-full bg-slate-300" />
          </ScrollArea.Scrollbar>
        </ScrollArea.Root>
      </aside>

      <div className="flex-1 flex flex-col overflow-hidden">
        <ScrollArea.Root className="flex-1 overflow-hidden">
          <ScrollArea.Viewport className="h-full w-full">
            <div className="px-8 py-4 max-w-3xl text-left">
              <Markdown mermaid size="sm">{pageDetail?.content ?? ''}</Markdown>
            </div>
          </ScrollArea.Viewport>
          <ScrollArea.Scrollbar orientation="vertical" className="flex w-1.5 touch-none select-none p-0.5">
            <ScrollArea.Thumb className="relative flex-1 rounded-full bg-slate-300" />
          </ScrollArea.Scrollbar>
        </ScrollArea.Root>
      </div>
    </div>
  )
}
