export function slugify(text: string): string {
  const slug = text
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\s-]/gu, '')
    .trim()
    .replace(/\s+/g, '-')
  return slug || 'section'
}

// Pure and idempotent: duplicates get `-1`, `-2`… suffixes, counters restart on each call.
export function buildHeadingIds(headings: string[]): string[] {
  const seen = new Map<string, number>()
  return headings.map(text => {
    const base = slugify(text)
    const n = seen.get(base) ?? 0
    seen.set(base, n + 1)
    return n === 0 ? base : `${base}-${n}`
  })
}
