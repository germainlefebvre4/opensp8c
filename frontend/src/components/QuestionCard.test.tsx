import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { QuestionCard } from './QuestionCard'
import type { QuestionCardData } from '../hooks/exploreChat'
import frExplore from '../locales/fr/explore.json'

// QuestionCard renders text via useTranslation('explore'); initialize a
// minimal i18next instance with the fr resources so assertions can check the
// actual rendered copy instead of raw translation keys.
void i18n.use(initReactI18next).init({
  lng: 'fr',
  fallbackLng: 'fr',
  ns: ['explore'],
  defaultNS: 'explore',
  resources: { fr: { explore: frExplore } },
  interpolation: { escapeValue: false },
})

const baseQuestion: QuestionCardData = { id: 'q1', source: 'ghost', text: 'Quel est le périmètre ?' }

describe('QuestionCard', () => {
  it('renders the answered summary once the question has a final answer, with no input', () => {
    const html = renderToStaticMarkup(
      <QuestionCard
        question={{ ...baseQuestion, answer: 'Le périmètre X' }}
        isEditing={false}
        onStartEdit={vi.fn()}
        onStage={vi.fn()}
        onCancelStaged={vi.fn()}
      />
    )
    expect(html).toContain('Le périmètre X')
    expect(html).not.toContain('<input')
  })

  it('renders a free-text input with a disabled Validate button when there is no staged answer yet', () => {
    const html = renderToStaticMarkup(
      <QuestionCard
        question={baseQuestion}
        isEditing={false}
        onStartEdit={vi.fn()}
        onStage={vi.fn()}
        onCancelStaged={vi.fn()}
      />
    )
    expect(html).toContain('<input')
    expect(html).toContain('disabled=""')
  })

  it('renders quick options plus an "other answer" link when the question has options and no staged answer', () => {
    const html = renderToStaticMarkup(
      <QuestionCard
        question={{ ...baseQuestion, options: [{ label: 'JSON' }, { label: 'YAML' }] }}
        isEditing={false}
        onStartEdit={vi.fn()}
        onStage={vi.fn()}
        onCancelStaged={vi.fn()}
      />
    )
    expect(html).toContain('JSON')
    expect(html).toContain('YAML')
    expect(html).toContain('Autre réponse...')
    expect(html).not.toContain('<input')
  })

  it('shows the free-text input instead of options while isEditing is true', () => {
    const html = renderToStaticMarkup(
      <QuestionCard
        question={{ ...baseQuestion, options: [{ label: 'JSON' }] }}
        isEditing
        onStartEdit={vi.fn()}
        onStage={vi.fn()}
        onCancelStaged={vi.fn()}
      />
    )
    expect(html).not.toContain('>JSON<')
    expect(html).toContain('<input')
  })

  it('renders the "staged" badge with the prepared answer text and edit/cancel controls', () => {
    const html = renderToStaticMarkup(
      <QuestionCard
        question={baseQuestion}
        stagedAnswer="Ma réponse préparée"
        isEditing={false}
        onStartEdit={vi.fn()}
        onStage={vi.fn()}
        onCancelStaged={vi.fn()}
      />
    )
    expect(html).toContain("Réponse en attente d&#x27;envoi")
    expect(html).toContain('Ma réponse préparée')
    expect(html).toContain('Modifier')
    expect(html).toContain('Annuler')
    expect(html).not.toContain('<input')
  })

  it('pre-fills the input with the staged answer when re-opened for editing', () => {
    const html = renderToStaticMarkup(
      <QuestionCard
        question={baseQuestion}
        stagedAnswer="Ma réponse préparée"
        isEditing
        onStartEdit={vi.fn()}
        onStage={vi.fn()}
        onCancelStaged={vi.fn()}
      />
    )
    expect(html).toContain('value="Ma réponse préparée"')
    expect(html).not.toContain('disabled=""')
  })

  it('renders a superseded question without any interactive controls', () => {
    const html = renderToStaticMarkup(
      <QuestionCard
        question={{ ...baseQuestion, superseded: true }}
        isEditing={false}
        onStartEdit={vi.fn()}
        onStage={vi.fn()}
        onCancelStaged={vi.fn()}
      />
    )
    expect(html).not.toContain('<input')
    expect(html).not.toContain('<button')
  })
})
