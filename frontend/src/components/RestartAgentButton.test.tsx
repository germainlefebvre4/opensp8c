// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { RestartAgentButton } from './RestartAgentButton'
import { canOfferRestart } from '../hooks/exploreChat'

describe('RestartAgentButton', () => {
  it('calls onRestart when clicked', () => {
    const onRestart = vi.fn()
    render(<RestartAgentButton onRestart={onRestart} />)
    fireEvent.click(screen.getByRole('button'))
    expect(onRestart).toHaveBeenCalledTimes(1)
  })
})

describe('canOfferRestart', () => {
  const agent = (id: string) => ({ id, label: id, version: '' })
  it('is offered only when stalled with a Claude agent', () => {
    expect(canOfferRestart(true, agent('claude'))).toBe(true)
    expect(canOfferRestart(false, agent('claude'))).toBe(false)
    for (const id of ['gemini', 'antigravity', 'codex', 'copilot']) {
      expect(canOfferRestart(true, agent(id))).toBe(false)
    }
    expect(canOfferRestart(true, null)).toBe(false)
  })
})
