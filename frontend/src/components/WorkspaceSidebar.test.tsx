// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { WorkspaceSidebar } from './WorkspaceSidebar'
import type { Workspace } from '../hooks/useWorkspaces'
import frWorkspace from '../locales/fr/workspace.json'
import frCommon from '../locales/fr/common.json'

const addMutate = vi.fn()
const removeMutate = vi.fn()
const removeState = { isPending: false }
vi.mock('../hooks/useWorkspaces', () => ({
  useAddWorkspace: () => ({ mutateAsync: addMutate }),
  useRemoveWorkspace: () => ({ mutateAsync: removeMutate, isPending: removeState.isPending }),
}))

beforeAll(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'fr',
    fallbackLng: 'fr',
    ns: ['workspace', 'common'],
    defaultNS: 'workspace',
    resources: { fr: { workspace: frWorkspace, common: frCommon } },
    interpolation: { escapeValue: false },
  })
})

beforeEach(() => {
  addMutate.mockReset()
  removeMutate.mockReset()
  removeMutate.mockResolvedValue(undefined)
})
afterEach(cleanup)

const ALL_COUNTS = { 'to-explore': 1, ready: 1, todo: 1, 'in-progress': 1, verifying: 1, 'to-review': 1, done: 1 }

function ws(over: Partial<Workspace> & { id: string; name: string }): Workspace {
  return { path: '/x', task_counts: {}, attention: [], ...over }
}

function Where() {
  const l = useLocation()
  return <div data-testid="where">{l.pathname + l.search}</div>
}

function setup(workspaces: Workspace[], opts: { isOpen?: boolean; activeId?: string | null } = {}) {
  const onSelect = vi.fn()
  const utils = render(
    <MemoryRouter>
      <WorkspaceSidebar
        workspaces={workspaces}
        activeId={opts.activeId === undefined ? workspaces[0]?.id ?? null : opts.activeId}
        onSelect={onSelect}
        isOpen={opts.isOpen ?? true}
        onToggle={vi.fn()}
      />
      <Where />
    </MemoryRouter>,
  )
  return { onSelect, ...utils }
}

describe('WorkspaceSidebar', () => {
  it('has no Configuration link nor agent selector', () => {
    const { container } = setup([ws({ id: 'a', name: 'Alpha' })])
    expect(container.innerHTML).not.toContain('/configuration')
    expect(container.innerHTML).not.toContain('agent-selector')
  })

  it('shows project list, add button and toggle', () => {
    setup([ws({ id: 'a', name: 'Alpha' })])
    expect(screen.getByText('Alpha')).toBeTruthy()
    expect(screen.getByText(frWorkspace.addProject)).toBeTruthy()
    expect(screen.getByLabelText(frWorkspace.closeMenu)).toBeTruthy()
  })

  it('uses w-72 open and w-10 collapsed', () => {
    const open = setup([ws({ id: 'a', name: 'Alpha' })])
    expect(open.container.querySelector('aside')?.className).toContain('w-72')
    cleanup()
    const closed = setup([ws({ id: 'a', name: 'Alpha' })], { isOpen: false })
    expect(closed.container.querySelector('aside')?.className).toContain('w-10')
    expect(screen.getByLabelText(frWorkspace.openMenu)).toBeTruthy()
  })

  it('renders one segment per status with the total', () => {
    setup([ws({ id: 'a', name: 'Alpha', task_counts: ALL_COUNTS })])
    const bar = screen.getByTestId('status-bar')
    expect(bar.children).toHaveLength(7)
    expect(Array.from(bar.children).map(c => c.getAttribute('data-testid'))).toEqual(
      ['to-explore', 'ready', 'todo', 'in-progress', 'verifying', 'to-review', 'done'].map(s => `segment-${s}`),
    )
    expect(screen.getByTestId('status-total').textContent).toBe('7')
    expect(screen.getByTestId('segment-in-progress').getAttribute('title')).toBe('En cours : 1')
  })

  it('renders an empty bar with total 0', () => {
    setup([ws({ id: 'a', name: 'Alpha' })])
    expect(screen.getByTestId('status-bar').children).toHaveLength(0)
    expect(screen.getByTestId('status-total').textContent).toBe('0')
  })

  it('selects on click outside the controls and by keyboard, with aria-current', () => {
    const { onSelect } = setup([ws({ id: 'a', name: 'Alpha' }), ws({ id: 'b', name: 'Beta' })])
    const rows = screen.getAllByRole('button').filter(b => b.getAttribute('tabindex') === '0' && b.getAttribute('aria-expanded') === null && !b.getAttribute('aria-label'))
    expect(rows[0].getAttribute('aria-current')).toBe('true')
    expect(rows[1].getAttribute('aria-current')).toBeNull()
    fireEvent.click(screen.getByText('Beta'))
    expect(onSelect).toHaveBeenCalledWith('b')
    onSelect.mockClear()
    fireEvent.keyDown(rows[1], { key: 'Enter' })
    fireEvent.keyDown(rows[1], { key: ' ' })
    expect(onSelect).toHaveBeenCalledTimes(2)
  })

  describe('attention', () => {
    const attention = [
      { change: 'fix-export', signals: [{ kind: 'paused' as const, reason: 'tests' }, { kind: 'hitl' as const, reason: 'walk' }, { kind: 'hitl' as const, reason: 'check' }] },
      { change: 'later', signals: [{ kind: 'review' as const }] },
    ]

    it('shows the distinct change count and hides it at 0', () => {
      setup([ws({ id: 'a', name: 'Alpha', attention }), ws({ id: 'b', name: 'Beta' })])
      expect(screen.getAllByTestId('attention-count')).toHaveLength(1)
      expect(screen.getByTestId('attention-count').textContent).toBe('2')
      expect(screen.getAllByLabelText(frWorkspace.expand)).toHaveLength(1)
    })

    it('expands independently, one title per change and one line per signal', () => {
      setup([ws({ id: 'a', name: 'Alpha', attention }), ws({ id: 'b', name: 'Beta', attention })])
      expect(screen.queryByText('fix-export')).toBeNull()
      fireEvent.click(screen.getAllByLabelText(frWorkspace.expand)[0])
      expect(screen.getAllByText('fix-export')).toHaveLength(1)
      expect(screen.getAllByTestId('signal-line')).toHaveLength(4)
      expect(screen.getAllByLabelText(frWorkspace.expand)).toHaveLength(1)
    })

    it('caps hitl lines at three and shows "+N autres"', () => {
      const many = [{ change: 'big', signals: Array.from({ length: 5 }, (_, i) => ({ kind: 'hitl' as const, reason: `t${i}` })) }]
      setup([ws({ id: 'a', name: 'Alpha', attention: many })])
      fireEvent.click(screen.getByLabelText(frWorkspace.expand))
      expect(screen.getAllByTestId('signal-line')).toHaveLength(3)
      expect(screen.getByText('+2 autres')).toBeTruthy()
    })

    it('opens a change in the Kanban of its workspace', () => {
      setup([ws({ id: 'a', name: 'Alpha', attention })])
      fireEvent.click(screen.getByLabelText(frWorkspace.expand))
      fireEvent.click(screen.getByText('later'))
      expect(screen.getByTestId('where').textContent).toBe('/?workspace=a&change=later')
    })

    it('shows a rail dot only for projects with attention when collapsed', () => {
      setup([ws({ id: 'a', name: 'Alpha', attention }), ws({ id: 'b', name: 'Beta' })], { isOpen: false })
      expect(screen.getAllByTestId('attention-dot')).toHaveLength(1)
      expect(screen.getByTitle('Alpha').textContent).toContain('AL')
    })

    it('selects a project from the rail', () => {
      const { onSelect } = setup([ws({ id: 'a', name: 'Alpha' })], { isOpen: false })
      fireEvent.click(screen.getByTitle('Alpha'))
      expect(onSelect).toHaveBeenCalledWith('a')
    })
  })

  describe('actions menu and removal', () => {
    const openMenu = (name: string) => {
      const trigger = screen.getByLabelText(`Actions du projet ${name}`)
      trigger.focus()
      fireEvent.keyDown(trigger, { key: 'Enter' })
    }

    it('opens from the keyboard and asks confirmation naming the project', async () => {
      setup([ws({ id: 'a', name: 'Alpha' })])
      openMenu('Alpha')
      fireEvent.click(await screen.findByText(frWorkspace.removeFromTracking))
      const dialog = await screen.findByRole('dialog')
      expect(within(dialog).getByText(/Alpha/)).toBeTruthy()
      expect(within(dialog).getByText(frWorkspace.removeBody)).toBeTruthy()
      expect(removeMutate).not.toHaveBeenCalled()
      fireEvent.click(within(dialog).getByText(frWorkspace.removeConfirm))
      expect(removeMutate).toHaveBeenCalledWith('a')
    })

    it('keeps the dialog open with the error when removal fails', async () => {
      removeMutate.mockRejectedValueOnce(new Error('disk error'))
      setup([ws({ id: 'a', name: 'Alpha' })])
      openMenu('Alpha')
      fireEvent.click(await screen.findByText(frWorkspace.removeFromTracking))
      const dialog = await screen.findByRole('dialog')
      fireEvent.click(within(dialog).getByText(frWorkspace.removeConfirm))
      expect(await within(dialog).findByText('disk error')).toBeTruthy()
      expect(screen.queryByRole('dialog')).not.toBeNull()
    })

    it('does nothing when cancelled', async () => {
      setup([ws({ id: 'a', name: 'Alpha' })])
      openMenu('Alpha')
      fireEvent.click(await screen.findByText(frWorkspace.removeFromTracking))
      const dialog = await screen.findByRole('dialog')
      fireEvent.click(within(dialog).getByText(frCommon.cancel))
      await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
      expect(removeMutate).not.toHaveBeenCalled()
    })
  })

  describe('add errors', () => {
    const submit = async (value: string) => {
      fireEvent.click(screen.getByText(frWorkspace.addProject))
      const input = screen.getByPlaceholderText(frWorkspace.projectPathPlaceholder) as HTMLInputElement
      fireEvent.change(input, { target: { value } })
      fireEvent.submit(input.closest('form')!)
      return input
    }

    it.each([
      ['directory does not contain an openspec/ folder'],
      ['workspace already exists'],
      ['path does not exist: stat /nope'],
    ])('keeps the form open with the value and shows "%s"', async message => {
      addMutate.mockRejectedValueOnce(new Error(message))
      setup([ws({ id: 'a', name: 'Alpha' })])
      const input = await submit('/some/path')
      expect((await screen.findByRole('alert')).textContent).toBe(message)
      expect(input.value).toBe('/some/path')
    })

    it('clears the error when the field changes or on cancel', async () => {
      addMutate.mockRejectedValue(new Error('boom'))
      setup([ws({ id: 'a', name: 'Alpha' })])
      const input = await submit('/p')
      await screen.findByRole('alert')
      fireEvent.change(input, { target: { value: '/p2' } })
      expect(screen.queryByRole('alert')).toBeNull()
      fireEvent.submit(input.closest('form')!)
      await screen.findByRole('alert')
      fireEvent.click(screen.getByText(frCommon.cancel))
      expect(screen.queryByRole('alert')).toBeNull()
    })
  })
})
