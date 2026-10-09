// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { afterEach } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { SubTabs } from './SubTabs'

const tabs = [
  { id: 'a', label: 'Alpha' },
  { id: 'b', label: 'Beta' },
]

afterEach(cleanup)

describe('SubTabs', () => {
  it('marks the active tab', () => {
    render(<SubTabs tabs={tabs} active="b" onChange={() => {}} />)
    expect(screen.getByRole('tab', { name: 'Beta' }).getAttribute('aria-selected')).toBe('true')
    expect(screen.getByRole('tab', { name: 'Alpha' }).getAttribute('aria-selected')).toBe('false')
  })

  it('calls onChange with the clicked tab id', () => {
    const onChange = vi.fn()
    render(<SubTabs tabs={tabs} active="a" onChange={onChange} />)
    fireEvent.click(screen.getByRole('tab', { name: 'Beta' }))
    expect(onChange).toHaveBeenCalledWith('b')
  })

  it('renders trailing content right-aligned and not as a tab', () => {
    const onChange = vi.fn()
    render(<SubTabs tabs={tabs} active="a" onChange={onChange} trailing={<span>Workspace A</span>} />)
    expect(screen.getByTestId('subtabs-trailing').className).toContain('ml-auto')
    expect(screen.getAllByRole('tab')).toHaveLength(2)
    fireEvent.click(screen.getByText('Workspace A'))
    expect(onChange).not.toHaveBeenCalled()
  })

  it('renders no trailing area without trailing', () => {
    render(<SubTabs tabs={tabs} active="a" onChange={() => {}} />)
    expect(screen.queryByTestId('subtabs-trailing')).toBeNull()
  })
})
