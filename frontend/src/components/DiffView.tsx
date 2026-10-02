import { useMemo } from 'react'
import { parsePatch } from 'diff'
import { useTranslation } from 'react-i18next'

interface Props {
  patch: string
  binary?: boolean
  truncated?: boolean
}

interface Row {
  kind: 'add' | 'del' | 'ctx' | 'note'
  text: string
  oldNo?: number
  newNo?: number
}

interface Hunk {
  header: string
  rows: Row[]
}

function toHunks(patch: string): Hunk[] {
  const hunks: Hunk[] = []
  try {
    for (const file of parsePatch(patch)) {
      for (const h of file.hunks) {
        let oldNo = h.oldStart
        let newNo = h.newStart
        const rows: Row[] = []
        for (const line of h.lines) {
          const sign = line[0]
          const text = line.slice(1)
          if (sign === '+') rows.push({ kind: 'add', text, newNo: newNo++ })
          else if (sign === '-') rows.push({ kind: 'del', text, oldNo: oldNo++ })
          else if (sign === '\\') rows.push({ kind: 'note', text: line })
          else rows.push({ kind: 'ctx', text, oldNo: oldNo++, newNo: newNo++ })
        }
        hunks.push({ header: `@@ -${h.oldStart},${h.oldLines} +${h.newStart},${h.newLines} @@`, rows })
      }
    }
  } catch {
    // A patch cut mid-hunk by the size limit is not parseable: show what we have.
    return [{
      header: '',
      rows: patch.split('\n').map(line => ({
        kind: line.startsWith('+') ? 'add' : line.startsWith('-') ? 'del' : 'ctx',
        text: line,
      })),
    }]
  }
  return hunks
}

const ROW_CLASS: Record<Row['kind'], string> = {
  add: 'flex gap-2 bg-green-50 text-green-800 px-1 rounded-sm',
  del: 'flex gap-2 bg-red-50 text-red-800 px-1 rounded-sm',
  ctx: 'flex gap-2 text-slate-500 px-1',
  note: 'flex gap-2 text-slate-400 italic px-1',
}

const SIGN: Record<Row['kind'], string> = { add: '+', del: '-', ctx: ' ', note: ' ' }

/** Renders a unified patch: hunks, line numbers, added and removed lines. */
export function DiffView({ patch, binary, truncated }: Props) {
  const { t } = useTranslation('detailPanel')
  const hunks = useMemo(() => (binary ? [] : toHunks(patch)), [patch, binary])

  if (binary) {
    return <p className="text-xs text-slate-400 p-2">{t('review.binary')}</p>
  }

  return (
    <div className="font-mono text-[11px] leading-relaxed bg-slate-50 rounded-md py-1 overflow-x-auto">
      {hunks.length === 0 && !truncated && (
        <p className="text-slate-400 p-2 font-sans">{t('review.noDiff')}</p>
      )}
      {hunks.map((hunk, i) => (
        <div key={i} data-testid="diff-hunk" className="mb-1">
          {hunk.header && <div className="text-slate-400 px-1 select-none">{hunk.header}</div>}
          {hunk.rows.map((row, j) => (
            <div key={j} data-kind={row.kind} className={ROW_CLASS[row.kind]}>
              <span className="select-none w-8 shrink-0 text-right text-slate-400">{row.oldNo ?? ''}</span>
              <span className="select-none w-8 shrink-0 text-right text-slate-400">{row.newNo ?? ''}</span>
              <span className="select-none w-3 shrink-0 text-center">{SIGN[row.kind]}</span>
              <span className="break-all whitespace-pre-wrap">{row.text || ' '}</span>
            </div>
          ))}
        </div>
      ))}
      {truncated && <p className="text-amber-700 p-2 font-sans">{t('review.truncated')}</p>}
    </div>
  )
}
