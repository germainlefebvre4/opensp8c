// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, render, screen } from '@testing-library/react'
import { ToastProvider } from './Toast'
import { useToast } from '../../hooks/useToast'
import type { ToastOptions } from '../../lib/toastContext'

afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

function Trigger({ options }: { options: ToastOptions }) {
  const { toast } = useToast()
  return <button onClick={() => toast(options)}>go</button>
}

function show(options: ToastOptions) {
  render(<ToastProvider><Trigger options={options} /></ToastProvider>)
  act(() => { screen.getByText('go').click() })
}

const iconClass = (title: string) =>
  screen.getByText(title).parentElement!.querySelector('svg')!.getAttribute('class') ?? ''

describe('Toast', () => {
  it('renders the warning variant with an amber icon', () => {
    show({ title: 'careful', variant: 'warning' })
    expect(iconClass('careful')).toContain('text-amber-500')
  })

  it('keeps the error and success variants unchanged', () => {
    show({ title: 'bad', variant: 'error' })
    expect(iconClass('bad')).toContain('text-red-500')
    cleanup()
    show({ title: 'good' })
    expect(iconClass('good')).toContain('text-emerald-500')
  })

  it('closes after 4 s by default', () => {
    vi.useFakeTimers()
    show({ title: 'short' })
    expect(screen.queryByText('short')).not.toBeNull()
    act(() => { vi.advanceTimersByTime(4500) })
    expect(screen.queryByText('short')).toBeNull()
  })

  it('honours a custom duration', () => {
    vi.useFakeTimers()
    show({ title: 'long', variant: 'warning', duration: 10_000 })
    act(() => { vi.advanceTimersByTime(5000) })
    expect(screen.queryByText('long')).not.toBeNull()
    act(() => { vi.advanceTimersByTime(6000) })
    expect(screen.queryByText('long')).toBeNull()
  })
})
