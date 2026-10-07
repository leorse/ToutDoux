import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { setBackend } from './api'
import { MeetingsTab } from './components/MeetingsTab'
import { NotesTab } from './components/NotesTab'
import { ProjectSidebar } from './components/ProjectSidebar'
import { RichEditor } from './components/RichEditor'
import { FILTRES_PAR_DEFAUT, TasksTab } from './components/TasksTab'
import { COULEURS, teinte } from './couleurs'
import { demoBackend } from './demoBackend'
import type { domain } from '../wailsjs/go/models'

/**
 * Finitions de la v1.2.0 : couleur des notes et des réunions, teinte du projet
 * « Transverse / Divers », échéance des tâches terminées.
 */

const rien = () => {}

/** Les boutons de la barre d'outils réagissent à `mousedown`, pour ne pas voler le focus à l'éditeur. */
function appuyer(element: HTMLElement) {
  fireEvent.mouseDown(element)
}

function ouvrirPalette(): HTMLElement {
  appuyer(screen.getByRole('button', { name: 'Couleur' }))
  return screen.getByRole('menu', { name: 'Couleurs' })
}

function choisirCouleur(nom: string) {
  appuyer(within(ouvrirPalette()).getByRole('menuitemradio', { name: nom }))
}

describe('Palette', () => {
  it('compte dix pastels, les mêmes clés que le backend', () => {
    expect(COULEURS.map((c) => c.cle)).toEqual([
      'rose', 'peche', 'jaune', 'anis', 'menthe', 'turquoise', 'ciel', 'lavande', 'lilas', 'sable',
    ])
  })

  it('ne rend aucune teinte pour l’absence de couleur ou une clé inconnue', () => {
    expect(teinte('ciel')).toBe('#bae6fd')
    expect(teinte('')).toBeUndefined()
    expect(teinte(undefined)).toBeUndefined()
    expect(teinte('fuchsia')).toBeUndefined()
  })
})

describe('Bouton de couleur de l’éditeur', () => {
  it('n’apparaît pas quand l’éditeur n’a rien à colorer', () => {
    render(<RichEditor content="<p>x</p>" onChange={rien} />)
    expect(screen.queryByRole('button', { name: 'Couleur' })).toBeNull()
  })

  it('se place juste après les deux boutons de plein écran', () => {
    render(<RichEditor content="<p>x</p>" onChange={rien} onColorChange={rien} />)
    const precedent = screen.getByRole('button', { name: 'Couleur' }).parentElement!.previousElementSibling
    expect(precedent).toHaveAttribute('aria-label', 'Agrandir en plein écran')
    expect(precedent!.previousElementSibling).toHaveAttribute('aria-label', 'Afficher en plein écran (lecture seule)')
  })

  it('propose dix pastilles et « Aucune couleur », la couleur courante cochée', () => {
    render(<RichEditor content="<p>x</p>" onChange={rien} color="menthe" onColorChange={rien} />)
    const choix = within(ouvrirPalette()).getAllByRole('menuitemradio')

    expect(choix).toHaveLength(11)
    expect(choix.map((c) => c.getAttribute('aria-label')).at(-1)).toBe('Aucune couleur')
    expect(choix.filter((c) => c.getAttribute('aria-checked') === 'true').map((c) => c.getAttribute('aria-label'))).toEqual([
      'Menthe',
    ])
  })

  it('applique la couleur choisie et referme la palette', () => {
    const onColorChange = vi.fn()
    render(<RichEditor content="<p>x</p>" onChange={rien} onColorChange={onColorChange} />)

    choisirCouleur('Ciel')

    expect(onColorChange).toHaveBeenCalledWith('ciel')
    expect(screen.queryByRole('menu', { name: 'Couleurs' })).toBeNull()
  })

  it('« Aucune couleur » retire la couleur', () => {
    const onColorChange = vi.fn()
    render(<RichEditor content="<p>x</p>" onChange={rien} color="ciel" onColorChange={onColorChange} />)

    choisirCouleur('Aucune couleur')

    expect(onColorChange).toHaveBeenCalledWith('')
  })

  it('se referme sur Échap ou sur un clic ailleurs, sans rien changer', () => {
    const onColorChange = vi.fn()
    render(<RichEditor content="<p>x</p>" onChange={rien} onColorChange={onColorChange} />)

    ouvrirPalette()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('menu', { name: 'Couleurs' })).toBeNull()

    ouvrirPalette()
    fireEvent.mouseDown(document.body)
    expect(screen.queryByRole('menu', { name: 'Couleurs' })).toBeNull()

    expect(onColorChange).not.toHaveBeenCalled()
  })

  it('en plein écran modifiable, Échap ferme la palette et pas le plein écran', () => {
    render(<RichEditor content="<p>x</p>" onChange={rien} onColorChange={rien} />)
    appuyer(screen.getByRole('button', { name: 'Agrandir en plein écran' }))

    ouvrirPalette()
    fireEvent.keyDown(window, { key: 'Escape' })

    expect(screen.queryByRole('menu', { name: 'Couleurs' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Fermer le plein écran' })).toBeInTheDocument()
  })

  it('disparaît avec la barre d’outils en plein écran lecture seule', () => {
    render(<RichEditor content="<p>x</p>" onChange={rien} onColorChange={rien} />)
    appuyer(screen.getByRole('button', { name: 'Afficher en plein écran (lecture seule)' }))
    expect(screen.queryByRole('button', { name: 'Couleur' })).toBeNull()
  })
})

describe('Couleur d’une note', () => {
  const PROJET = 'p-couleurs'

  async function semer(...titres: string[]): Promise<Record<string, string>> {
    const ids: Record<string, string> = {}
    for (const titre of [...titres].reverse()) ids[titre] = (await demoBackend.CreateNote(PROJET, titre)).id
    return ids
  }

  async function afficher() {
    render(<NotesTab projectId={PROJET} onDataChanged={rien} montrerCaches={false} onToggleCaches={rien} />)
    return screen.findByRole('list', { name: 'Notes' })
  }

  const note = (titre: string) => screen.getByRole('button', { name: titre })

  it('colore la ligne de la note ouverte, qui reste marquée comme ouverte', async () => {
    await semer('A', 'B')
    await afficher()
    await screen.findByRole('button', { name: 'Couleur' })

    choisirCouleur('Ciel')

    await waitFor(() => expect(note('A')).toHaveStyle({ backgroundColor: '#bae6fd' }))
    expect(note('A')).toHaveAttribute('aria-current', 'true')
    expect(note('A').className).toContain('ring-[var(--color-selection)]')
    expect(note('B').style.backgroundColor).toBe('')
    expect((await demoBackend.GetNotes(PROJET)).map((n) => n.color ?? '')).toEqual(['ciel', ''])
  })

  it('« Aucune couleur » rend la note à son apparence ordinaire', async () => {
    const ids = await semer('A')
    await demoBackend.SetNoteColor(ids.A, 'rose')
    await afficher()
    await waitFor(() => expect(note('A')).toHaveStyle({ backgroundColor: '#fbcfe8' }))

    choisirCouleur('Aucune couleur')

    await waitFor(() => expect(note('A').style.backgroundColor).toBe(''))
    expect(note('A')).toHaveClass('bg-neutral-200')
  })

  it('colore aussi une note rangée dans un groupe, sans toucher au cadre', async () => {
    const ids = await semer('A', 'B')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.A, ids.B])
    await demoBackend.SetNoteColor(ids.B, 'rose')
    await afficher()

    const cadre = await screen.findByRole('group', { name: 'G' })
    await waitFor(() => expect(within(cadre).getByRole('button', { name: 'B' })).toHaveStyle({ backgroundColor: '#fbcfe8' }))
    expect(cadre.style.backgroundColor).toBe('')
    expect(cadre).toHaveClass('border-neutral-400')
  })

  it('garde la couleur d’une note cachée, atténuée comme le reste de la ligne', async () => {
    const ids = await semer('A')
    await demoBackend.SetNoteColor(ids.A, 'jaune')
    await demoBackend.SetNoteHidden(ids.A, true)
    render(<NotesTab projectId={PROJET} onDataChanged={rien} montrerCaches onToggleCaches={rien} />)

    await waitFor(() => expect(note('A')).toHaveStyle({ backgroundColor: '#fef08a' }))
    expect(note('A')).toHaveClass('italic', 'opacity-50')
  })

  it('enregistre la frappe en cours quand on choisit une couleur juste après', async () => {
    const user = userEvent.setup()
    await semer('A')
    await afficher()

    await user.type(await screen.findByRole('textbox', { name: 'Titre de la note' }), '+')
    choisirCouleur('Lilas')

    await waitFor(() => expect(note('A+')).toHaveStyle({ backgroundColor: '#e9d5ff' }))
    const [enregistree] = await demoBackend.GetNotes(PROJET)
    expect(enregistree.title).toBe('A+')
    expect(enregistree.color).toBe('lilas')
  })
})

describe('Couleur d’une réunion', () => {
  // Le jeu de démonstration porte « Comité de pilotage » et ses deux instances.
  async function afficher() {
    render(
      <MeetingsTab projectId="p-mig" onDataChanged={rien} montrerCaches={false} onToggleCaches={rien} onRevelerCachee={rien} />,
    )
    await screen.findByRole('button', { name: 'Couleur' })
  }

  const reunion = () => within(screen.getByRole('list', { name: 'Réunions' })).getByRole('button', { name: 'Comité de pilotage' })
  const instances = () => within(screen.getByRole('list', { name: 'Instances' })).getAllByRole('button')

  it('colore la réunion dans sa liste, et laisse les instances neutres', async () => {
    await afficher()

    choisirCouleur('Jaune')

    await waitFor(() => expect(reunion()).toHaveStyle({ backgroundColor: '#fef08a' }))
    expect(reunion()).toHaveAttribute('aria-current', 'true')
    expect(instances()).toHaveLength(2)
    for (const ligne of instances()) expect(ligne.style.backgroundColor).toBe('')
  })

  it('montre la même couleur quelle que soit l’instance ouverte', async () => {
    const user = userEvent.setup()
    await afficher()
    choisirCouleur('Jaune')
    await waitFor(() => expect(reunion()).toHaveStyle({ backgroundColor: '#fef08a' }))

    await user.click(instances()[1])

    await waitFor(() =>
      expect(within(ouvrirPalette()).getByRole('menuitemradio', { name: 'Jaune' })).toHaveAttribute('aria-checked', 'true'),
    )
    expect(reunion()).toHaveStyle({ backgroundColor: '#fef08a' })
  })
})

describe('Teinte du projet « Transverse / Divers »', () => {
  const PROJETS = [
    { id: 'project-divers', name: 'Transverse / Divers', locked: true },
    { id: 'p-ordinaire', name: 'Ordinaire', locked: false },
  ] as unknown as domain.Project[]

  const stats = (hasCritical: boolean, hasHigh: boolean) => ({
    activeCount: 1, hasCritical, hasHigh, dueIcon: '',
    notesCount: 0, meetingsCount: 0, hiddenNotesCount: 0, hiddenMeetingsCount: 0,
  })

  function sidebar(revision: number) {
    return (
      <ProjectSidebar
        projects={PROJETS}
        selectedId="p-ordinaire"
        onSelect={rien}
        onChanged={rien}
        onError={rien}
        revision={revision}
        montrerCaches={false}
        onToggleCaches={rien}
      />
    )
  }

  const ligne = (nom: RegExp) => screen.getByRole('button', { name: nom })

  it('prend une teinte atténuée, distincte de celle d’un projet ordinaire', async () => {
    setBackend({ GetSidebarStats: async () => stats(true, false) as never })
    render(sidebar(0))

    await waitFor(() => expect(ligne(/Transverse/)).toHaveClass('bg-[var(--color-critique-attenuee)]'))
    expect(ligne(/Ordinaire/)).toHaveClass('bg-[var(--color-critique)]')
  })

  it('prend la teinte atténuée « haute » sans tâche critique', async () => {
    setBackend({ GetSidebarStats: async () => stats(false, true) as never })
    render(sidebar(0))

    await waitFor(() => expect(ligne(/Transverse/)).toHaveClass('bg-[var(--color-haute-attenuee)]'))
    expect(ligne(/Ordinaire/)).toHaveClass('bg-[var(--color-haute)]')
  })

  it('reste gris sans tâche importante, et y revient dès que la tâche est terminée', async () => {
    let critique = true
    setBackend({ GetSidebarStats: async () => stats(critique, false) as never })
    const { rerender } = render(sidebar(0))
    await waitFor(() => expect(ligne(/Transverse/)).toHaveClass('bg-[var(--color-critique-attenuee)]'))

    // Une écriture ailleurs dans l'application fait avancer `revision`.
    critique = false
    rerender(sidebar(1))

    await waitFor(() => expect(ligne(/Transverse/)).toHaveClass('bg-[var(--color-annulee)]'))
    expect(ligne(/Ordinaire/)).toHaveClass('bg-white')
  })
})

describe('Échéance d’une tâche terminée', () => {
  // « Rassembler les chiffres » est active et en retard dans le jeu de démonstration.
  const NOM = 'Rassembler les chiffres'
  const tous = { ...FILTRES_PAR_DEFAUT, status: ['active', 'completed', 'cancelled'] } as typeof FILTRES_PAR_DEFAUT

  const ligne = () => screen.getByText(NOM).closest('[role="treeitem"]') as HTMLElement

  it('n’affiche plus ni retard ni horloge, dans l’arbre comme dans le détail, et les retrouve si on la rouvre', async () => {
    const user = userEvent.setup()
    render(<TasksTab projectId="p-mig" filtres={tous} onFiltres={rien} onDataChanged={rien} />)

    await waitFor(() => expect(ligne()).toHaveTextContent('En retard'))
    expect(ligne()).toHaveTextContent('⏰')

    await user.click(within(ligne()).getByRole('checkbox', { name: `Terminer ${NOM}` }))
    await waitFor(() => expect(ligne()).not.toHaveTextContent('En retard'))
    expect(ligne()).not.toHaveTextContent('⏰')

    // Détail : plus de délai, mais la date est conservée et reste retirable.
    await user.click(screen.getByText(NOM))
    const date = await screen.findByLabelText('Échéance précise')
    expect(date).not.toHaveValue('')
    expect(screen.getByRole('button', { name: 'Retirer l’échéance' })).toBeInTheDocument()
    expect(screen.queryByText(/En retard/)).toBeNull()

    await user.click(within(ligne()).getByRole('checkbox', { name: `Terminer ${NOM}` }))
    await waitFor(() => expect(ligne()).toHaveTextContent('En retard'))
  })

  it('une tâche annulée n’affiche pas non plus de délai', async () => {
    await demoBackend.ToggleTaskCancelled('t2')
    render(<TasksTab projectId="p-mig" filtres={tous} onFiltres={rien} onDataChanged={rien} />)

    await waitFor(() => expect(screen.getByText(NOM)).toBeInTheDocument())
    expect(ligne()).not.toHaveTextContent('En retard')
  })
})
