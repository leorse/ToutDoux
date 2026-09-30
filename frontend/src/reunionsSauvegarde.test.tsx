import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { Editor } from '@tiptap/core'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { domain } from '../wailsjs/go/models'
import App from './App'
import { useState } from 'react'
import { MeetingsTab } from './components/MeetingsTab'
import { setBackend } from './api'
import { demoBackend } from './demoBackend'

/**
 * Sauvegarde du compte rendu d'une instance de réunion (correctif 1.1.1).
 *
 * En 1.1.0, chaque sauvegarde relançait le rendu d'App, qui recréait le rappel
 * `onRevelerCachee` ; l'effet de chargement de l'onglet Réunions repartait et
 * resélectionnait la première réunion — l'utilisateur perdait son éditeur.
 *
 * Le jeu de démonstration n'a d'instance que sur la première réunion : on en
 * donne deux à « Point technique » (la deuxième), pour éditer une instance qui
 * n'est ni dans la première réunion, ni la plus récente de la sienne.
 */

const RECENTE = 'i-m2-recente'
const ANCIENNE = 'i-m2-ancienne'

function instancesPointTechnique() {
  const heure = (h: number) => new Date(Date.now() + h * 3_600_000).toISOString()
  let liste = [
    { id: RECENTE, meetingId: 'm2', notes: '<p>Instance récente.</p>', timestamp: heure(-1), createdAt: heure(-1), updatedAt: heure(-1) },
    { id: ANCIENNE, meetingId: 'm2', notes: '<p>Instance ancienne.</p>', timestamp: heure(-30), createdAt: heure(-30), updatedAt: heure(-30) },
  ] as unknown as domain.MeetingInstance[]
  const sauvegardes: { id: string; html: string }[] = []
  const { GetInstances, UpdateInstanceNotes } = demoBackend
  setBackend({
    GetInstances: async (meetingId: string) => (meetingId === 'm2' ? liste : GetInstances(meetingId)),
    UpdateInstanceNotes: async (id: string, html: string) => {
      if (!liste.some((i) => i.id === id)) return UpdateInstanceNotes(id, html)
      sauvegardes.push({ id, html })
      liste = liste.map((i) => (i.id === id ? { ...i, notes: html } : i)) as unknown as domain.MeetingInstance[]
      return liste.find((i) => i.id === id)!
    },
  })
  return sauvegardes
}

/** L'instance TipTap de l'éditeur affiché, accrochée par TipTap à son nœud DOM. */
function editeur(): Editor {
  const dom = document.querySelector('.ProseMirror') as (HTMLElement & { editor?: Editor }) | null
  if (!dom?.editor) throw new Error('éditeur introuvable')
  return dom.editor
}

async function ouvrirAncienneInstance(user: ReturnType<typeof userEvent.setup>) {
  render(<App />)
  await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
  await user.click(screen.getByRole('tab', { name: 'Réunions' }))
  const reunions = await screen.findByRole('list', { name: 'Réunions' })
  await user.click(within(reunions).getByRole('button', { name: 'Point technique' }))
  const instances = screen.getByRole('list', { name: 'Instances' })
  await waitFor(() => expect(within(instances).getAllByRole('button')).toHaveLength(2))
  await user.click(within(instances).getAllByRole('button')[1])
  await waitFor(() => expect(screen.getByText('Instance ancienne.')).toBeInTheDocument())
}

function selectionIntacte() {
  const reunions = screen.getByRole('list', { name: 'Réunions' })
  expect(within(reunions).getByRole('button', { name: 'Point technique' })).toHaveClass('bg-neutral-200')
  expect(within(reunions).getByRole('button', { name: 'Comité de pilotage' })).not.toHaveClass('bg-neutral-200')
  const lignes = within(screen.getByRole('list', { name: 'Instances' })).getAllByRole('button')
  expect(lignes[1]).toHaveClass('bg-neutral-200')
  expect(lignes[0]).not.toHaveClass('bg-neutral-200')
}

afterEach(() => {
  vi.useRealTimers()
})

describe('Sauvegarde du compte rendu (1.1.1)', () => {
  it('garde la réunion et l’instance sélectionnées après une sauvegarde au blur', async () => {
    const sauvegardes = instancesPointTechnique()
    const user = userEvent.setup()
    await ouvrirAncienneInstance(user)

    act(() => {
      editeur().chain().focus('end').insertContent(' Ajout au blur.').run()
    })
    fireEvent.blur(document.querySelector('.ProseMirror')!)

    await waitFor(() => expect(sauvegardes).toHaveLength(1))
    expect(sauvegardes[0].id).toBe(ANCIENNE)
    expect(sauvegardes[0].html).toContain('Ajout au blur.')
    // Laisse le temps à un éventuel rechargement parasite de se produire.
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50))
    })
    selectionIntacte()
    expect(screen.getByText(/Sauvegardé à/)).toBeInTheDocument()
    expect(screen.getByText(/Instance ancienne\. Ajout au blur\./)).toBeInTheDocument()
  })

  it('garde la réunion et l’instance sélectionnées après la sauvegarde différée', async () => {
    const sauvegardes = instancesPointTechnique()
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    await ouvrirAncienneInstance(user)

    act(() => {
      editeur().chain().focus('end').insertContent(' Ajout différé.').run()
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2100)
    })

    await waitFor(() => expect(sauvegardes.map((s) => s.id)).toEqual([ANCIENNE]))
    await act(async () => {
      await vi.advanceTimersByTimeAsync(50)
    })
    selectionIntacte()
    expect(screen.getByText(/Sauvegardé à/)).toBeInTheDocument()
    expect(screen.getByText(/Instance ancienne\. Ajout différé\./)).toBeInTheDocument()
  })

  it('ne dépend pas de la stabilité des rappels du parent', async () => {
    const sauvegardes = instancesPointTechnique()
    // Parent volontairement négligent : rappels recréés à chaque rendu, et un
    // rendu à chaque `onDataChanged` — ce que faisait App en 1.1.0.
    function Parent() {
      const [, setRevision] = useState(0)
      return (
        <MeetingsTab
          projectId="p-mig"
          onDataChanged={() => setRevision((n) => n + 1)}
          montrerCaches={false}
          onToggleCaches={() => {}}
          onRevelerCachee={() => {}}
        />
      )
    }
    const user = userEvent.setup()
    render(<Parent />)
    const reunions = await screen.findByRole('list', { name: 'Réunions' })
    await user.click(await within(reunions).findByRole('button', { name: 'Point technique' }))
    const instances = screen.getByRole('list', { name: 'Instances' })
    await waitFor(() => expect(within(instances).getAllByRole('button')).toHaveLength(2))
    await user.click(within(instances).getAllByRole('button')[1])
    await waitFor(() => expect(screen.getByText('Instance ancienne.')).toBeInTheDocument())

    act(() => {
      editeur().chain().focus('end').insertContent(' Parent instable.').run()
    })
    fireEvent.blur(document.querySelector('.ProseMirror')!)

    await waitFor(() => expect(sauvegardes.map((s) => s.id)).toEqual([ANCIENNE]))
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50))
    })
    selectionIntacte()
  })

  it('laisse le focus dans l’éditeur pendant la sauvegarde différée', async () => {
    const sauvegardes = instancesPointTechnique()
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    await ouvrirAncienneInstance(user)
    const dom = document.querySelector('.ProseMirror') as HTMLElement
    // jsdom ne rend pas focalisable un `contenteditable` sans tabindex ; un vrai
    // navigateur, si. Sans cela, `focus()` n'aurait aucun effet ici.
    dom.tabIndex = 0

    act(() => {
      // `view.focus()` et non la commande `focus()` de TipTap : celle-ci attend
      // un requestAnimationFrame, que les faux timers retiennent.
      editeur().view.focus()
      editeur().chain().setTextSelection(editeur().state.doc.content.size - 1).insertContent(' Avant.').run()
    })
    expect(dom.contains(document.activeElement)).toBe(true)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2100)
    })
    await waitFor(() => expect(sauvegardes).toHaveLength(1))
    await act(async () => {
      await vi.advanceTimersByTimeAsync(50)
    })

    // Même nœud d'éditeur, toujours focalisé : la frappe continue là où elle était.
    expect(document.querySelector('.ProseMirror')).toBe(dom)
    expect(dom.contains(document.activeElement)).toBe(true)
    act(() => {
      editeur().commands.insertContent(' Après.')
    })
    expect(screen.getByText(/Instance ancienne\. Avant\. Après\./)).toBeInTheDocument()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2100)
    })
    await waitFor(() => expect(sauvegardes).toHaveLength(2))
    expect(sauvegardes[1]).toMatchObject({ id: ANCIENNE })
    expect(sauvegardes[1].html).toContain('Avant. Après.')
  })
})
