// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { MermaidDiagram } from './MermaidDiagram'
import enSpecs from '../locales/en/specs.json'

const renderMock = vi.hoisted(() => vi.fn())
vi.mock('mermaid', () => ({
  default: { initialize: vi.fn(), render: renderMock },
}))

void i18n.use(initReactI18next).init({
  lng: 'en',
  ns: ['specs'],
  defaultNS: 'specs',
  resources: { en: { specs: enSpecs } },
  interpolation: { escapeValue: false },
})

afterEach(() => {
  cleanup()
  renderMock.mockReset()
})

const SVG = '<svg viewBox="0 0 10 10" style="max-width: 10px"><rect id="r" width="10" height="10"/></svg>'

async function renderDiagram() {
  renderMock.mockResolvedValue({ svg: SVG })
  render(<MermaidDiagram code="graph TD; A-->B" />)
  return screen.findByRole('button', { name: 'Enlarge diagram' })
}

describe('MermaidDiagram', () => {
  it('renders the diagram as a labelled button', async () => {
    const button = await renderDiagram()
    expect(button.querySelector('svg')).not.toBeNull()
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('opens a modal containing the SVG on click, without re-rendering', async () => {
    fireEvent.click(await renderDiagram())
    const dialog = await screen.findByRole('dialog')
    expect(dialog.querySelector('svg')).not.toBeNull()
    expect(screen.getByTestId('mermaid-modal-diagram').className).toContain('[&_svg]:!max-w-none')
    expect(renderMock).toHaveBeenCalledTimes(1)
  })

  it('closes on Escape', async () => {
    fireEvent.click(await renderDiagram())
    const dialog = await screen.findByRole('dialog')
    fireEvent.keyDown(dialog, { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })

  it('closes with the close button', async () => {
    fireEvent.click(await renderDiagram())
    await screen.findByRole('dialog')
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })

  it('falls back to the raw code block, not clickable, on render failure', async () => {
    renderMock.mockRejectedValue(new Error('parse'))
    const { container } = render(<MermaidDiagram code="not mermaid" />)
    await waitFor(() => expect(container.querySelector('pre code')).not.toBeNull())
    expect(screen.queryByRole('button')).toBeNull()
    expect(screen.queryByRole('dialog')).toBeNull()
  })
})
