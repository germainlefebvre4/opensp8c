export type Granularity = 'day' | 'week' | 'month' | 'quarter'

export const GRANULARITIES: Granularity[] = ['day', 'week', 'month', 'quarter']

const MONTH_LABELS = ['janv', 'févr', 'mars', 'avr', 'mai', 'juin', 'juil', 'août', 'sept', 'oct', 'nov', 'déc']

function pad(n: number): string {
  return String(n).padStart(2, '0')
}

function parseISODate(date: string): Date {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(Date.UTC(y, m - 1, d))
}

/** ISO 8601 week-year and week number (Monday-start weeks, week 1 contains the year's first Thursday). */
function isoWeekInfo(date: Date): { year: number; week: number } {
  const d = new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()))
  const dayNum = (d.getUTCDay() + 6) % 7 // Monday = 0 ... Sunday = 6
  d.setUTCDate(d.getUTCDate() - dayNum + 3) // move to Thursday of this ISO week
  const firstThursday = new Date(Date.UTC(d.getUTCFullYear(), 0, 4))
  const firstDayNum = (firstThursday.getUTCDay() + 6) % 7
  firstThursday.setUTCDate(firstThursday.getUTCDate() - firstDayNum + 3)
  const week = 1 + Math.round((d.getTime() - firstThursday.getTime()) / (7 * 24 * 3600 * 1000))
  return { year: d.getUTCFullYear(), week }
}

/** Bucket key for a given granularity. Zero-padded so string sort === chronological sort. */
export function bucketKey(date: string, granularity: Granularity): string {
  const d = parseISODate(date)
  const y = d.getUTCFullYear()
  const m = d.getUTCMonth() + 1
  switch (granularity) {
    case 'day':
      return date
    case 'week': {
      const { year, week } = isoWeekInfo(d)
      return `${year}-W${pad(week)}`
    }
    case 'month':
      return `${y}-${pad(m)}`
    case 'quarter': {
      const q = Math.floor((m - 1) / 3) + 1
      return `${y}-Q${q}`
    }
  }
}

/** Short human label for a bucket key, for column headers and tooltips. */
export function bucketLabel(key: string, granularity: Granularity): string {
  switch (granularity) {
    case 'day': {
      const [, m, d] = key.split('-')
      return `${parseInt(d)}/${parseInt(m)}`
    }
    case 'week': {
      const [y, w] = key.split('-W')
      return `S${w} '${y.slice(2)}`
    }
    case 'month': {
      const [y, m] = key.split('-')
      return `${MONTH_LABELS[parseInt(m) - 1]} ${y.slice(2)}`
    }
    case 'quarter': {
      const [y, q] = key.split('-')
      return `${q.replace('Q', 'T')} ${y.slice(2)}`
    }
  }
}
