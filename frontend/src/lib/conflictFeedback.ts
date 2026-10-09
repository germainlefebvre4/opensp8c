import type { TFunction } from 'i18next'

/** Beyond this many files the list is cut and followed by a count. */
export const MAX_CONFLICT_FILES = 20

/**
 * Text prefilled in the correction dialog to ask the worker to resolve an
 * integration conflict. Its first line stands alone (it is the one that ends up
 * on the task line); the files and the guideline follow. `t` is bound to the
 * `dialogs` namespace.
 */
export function buildConflictFeedback(t: TFunction<'dialogs'>, target: string | undefined, files: string[] | undefined): string {
  const lines = [target ? t('reviewConflict.intro', { target }) : t('reviewConflict.introNoTarget')]
  if (files && files.length > 0) {
    lines.push('', t('reviewConflict.files'))
    for (const file of files.slice(0, MAX_CONFLICT_FILES)) lines.push(`- ${file}`)
    if (files.length > MAX_CONFLICT_FILES) {
      lines.push(t('reviewConflict.more', { count: files.length - MAX_CONFLICT_FILES }))
    }
  }
  lines.push('', t('reviewConflict.keepBoth'))
  return lines.join('\n')
}
