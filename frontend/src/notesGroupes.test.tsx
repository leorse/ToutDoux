import { createEvent, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { NotesTab } from './components/NotesTab'
import { demoBackend } from './demoBackend'

/**
 * Ordre manuel et groupes de notes (v1.2.0).
 *
 * Les tests sèment leurs notes dans un projet vide du backend de démonstration,
 * puis lisent la liste telle qu'elle est affichée : « A », ou « Nom[A,B] » pour
 * un groupe et ses notes.
 */

const PROJET = 'p-groupes'

/** Crée des notes dont la liste se lit dans l'ordre donné. Rend leurs identifiants par titre. */
async function semer(...titres: string[]): Promise<Record<string, string>> {
  const ids: Record<string, string> = {}
  // Chaque note créée passe en tête : on crée donc à rebours.
  for (const titre of [...titres].reverse()) ids[titre] = (await demoBackend.CreateNote(PROJET, titre)).id
  return ids
}

async function afficher(montrerCaches = false) {
  const rendu = render(
    <NotesTab projectId={PROJET} onDataChanged={() => {}} montrerCaches={montrerCaches} onToggleCaches={() => {}} />,
  )
  await screen.findByRole('list', { name: 'Notes' })
  return rendu
}

function liste(): HTMLElement {
  return screen.getByRole('list', { name: 'Notes' })
}

function note(titre: string): HTMLElement {
  return within(liste()).getByRole('button', { name: titre })
}

function bandeau(nom: string): HTMLElement {
  return within(screen.getByRole('group', { name: nom })).getByText(nom)
}

function affichage(): string[] {
  return Array.from(liste().children)
    .filter((li) => li.hasAttribute('data-ligne'))
    .map((li) => {
      const cadre = li.querySelector('[role="group"]') as HTMLElement | null
      if (!cadre) return li.textContent ?? ''
      const nom = document.getElementById(cadre.getAttribute('aria-labelledby')!)!.textContent
      const notes = within(cadre).getAllByRole('button').map((b) => b.textContent)
      return `${nom}[${notes.join(',')}]`
    })
}

async function attendreAffichage(...attendu: string[]) {
  await waitFor(() => expect(affichage()).toEqual(attendu))
}

async function clicDroit(element: HTMLElement, user: ReturnType<typeof userEvent.setup>) {
  await user.pointer({ keys: '[MouseRight]', target: element })
}

async function ctrlClic(element: HTMLElement, user: ReturnType<typeof userEvent.setup>) {
  await user.keyboard('{Control>}')
  await user.click(element)
  await user.keyboard('{/Control}')
}

/* ---- Glisser-déposer : jsdom n'a ni mise en page ni DragEvent complet ---- */

// Tout élément mesure 20 px de haut à partir de 0 : y = 5 tombe dans la moitié
// haute, y = 15 dans la moitié basse.
const Y = { haut: 5, bas: 15 }
const dataTransfer = { setData: () => {}, getData: () => '', effectAllowed: 'none' }

function evenement(type: 'dragStart' | 'dragOver' | 'drop' | 'dragEnd', element: HTMLElement, y = 0) {
  const e = createEvent[type](element, { dataTransfer })
  Object.defineProperty(e, 'clientY', { value: y })
  fireEvent(element, e)
}

function survoler(source: HTMLElement, cible: HTMLElement, moitie: keyof typeof Y = 'bas') {
  evenement('dragStart', source)
  evenement('dragOver', cible, Y[moitie])
}

function glisser(source: HTMLElement, cible: HTMLElement, moitie: keyof typeof Y = 'bas') {
  survoler(source, cible, moitie)
  evenement('drop', cible, Y[moitie])
  evenement('dragEnd', source)
}

let prompt: ReturnType<typeof vi.spyOn>

beforeEach(() => {
  prompt = vi.spyOn(window, 'prompt').mockReturnValue(null)
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue({
    x: 0, y: 0, top: 0, left: 0, right: 100, bottom: 20, width: 100, height: 20, toJSON: () => ({}),
  } as DOMRect)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('Sélection multiple', () => {
  it('Ctrl+clic ajoute une note à la sélection sans changer la note ouverte', async () => {
    const user = userEvent.setup()
    await semer('A', 'B', 'C')
    await afficher()

    await user.click(note('A'))
    await ctrlClic(note('C'), user)

    expect(note('A')).toHaveAttribute('aria-pressed', 'true')
    expect(note('C')).toHaveAttribute('aria-pressed', 'true')
    expect(note('B')).toHaveAttribute('aria-pressed', 'false')
    // La note ouverte se distingue des autres notes sélectionnées.
    expect(note('A')).toHaveAttribute('aria-current', 'true')
    expect(note('C')).not.toHaveAttribute('aria-current')
    expect(note('C')).toHaveClass('bg-sky-100')
    expect(screen.getByRole('textbox', { name: 'Titre de la note' })).toHaveValue('A')
  })

  it('Maj+clic sélectionne la plage depuis la note ouverte', async () => {
    const user = userEvent.setup()
    await semer('A', 'B', 'C', 'D')
    await afficher()

    await user.click(note('A'))
    await user.keyboard('{Shift>}')
    await user.click(note('C'))
    await user.keyboard('{/Shift}')

    for (const titre of ['A', 'B', 'C']) expect(note(titre)).toHaveAttribute('aria-pressed', 'true')
    expect(note('D')).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('textbox', { name: 'Titre de la note' })).toHaveValue('A')
  })

  it('un clic simple ramène la sélection à la note cliquée et l’ouvre', async () => {
    const user = userEvent.setup()
    await semer('A', 'B', 'C', 'D')
    await afficher()

    await user.click(note('A'))
    await ctrlClic(note('B'), user)
    await ctrlClic(note('C'), user)
    await user.click(note('D'))

    for (const titre of ['A', 'B', 'C']) expect(note(titre)).toHaveAttribute('aria-pressed', 'false')
    expect(note('D')).toHaveAttribute('aria-pressed', 'true')
    await waitFor(() => expect(screen.getByRole('textbox', { name: 'Titre de la note' })).toHaveValue('D'))
  })
})

describe('Regrouper', () => {
  it('rassemble les notes sélectionnées dans un groupe nommé, à la place de la première', async () => {
    const user = userEvent.setup()
    await semer('A', 'B', 'C', 'D')
    await afficher()
    prompt.mockReturnValue('  Sprint 12 ')

    await user.click(note('A'))
    await ctrlClic(note('C'), user)
    await ctrlClic(note('D'), user)
    await clicDroit(note('C'), user)
    await user.click(await screen.findByRole('menuitem', { name: 'Regrouper' }))

    await attendreAffichage('Sprint 12[A,C,D]', 'B')
    expect(prompt).toHaveBeenCalledWith('Nom du groupe ?')
  })

  it('clic droit hors de la sélection : seule la note visée est regroupée', async () => {
    const user = userEvent.setup()
    await semer('A', 'B')
    await afficher()
    prompt.mockReturnValue('Idées')

    await user.click(note('A'))
    await clicDroit(note('B'), user)
    expect(note('A')).toHaveAttribute('aria-pressed', 'false')
    expect(note('B')).toHaveAttribute('aria-pressed', 'true')
    await user.click(await screen.findByRole('menuitem', { name: 'Regrouper' }))

    await attendreAffichage('A', 'Idées[B]')
  })

  it('ne crée rien si la popup est annulée ou le nom blanc', async () => {
    const user = userEvent.setup()
    await semer('A', 'B')
    await afficher()

    for (const reponse of [null, '   ']) {
      prompt.mockReturnValue(reponse)
      await clicDroit(note('A'), user)
      await user.click(await screen.findByRole('menuitem', { name: 'Regrouper' }))
    }

    expect(affichage()).toEqual(['A', 'B'])
    expect(await demoBackend.GetNoteGroups(PROJET)).toHaveLength(0)
  })
})

describe('Cadre d’un groupe', () => {
  it('affiche le nom dans le bandeau et les notes dans le cadre, sous un nom accessible', async () => {
    const ids = await semer('A', 'B', 'C')
    await demoBackend.GroupNotes(PROJET, 'Sprint 12', [ids.A, ids.B])
    await afficher()

    await attendreAffichage('Sprint 12[A,B]', 'C')
    const cadre = screen.getByRole('group', { name: 'Sprint 12' })
    expect(cadre).toHaveClass('border-2')
    expect(bandeau('Sprint 12')).toHaveClass('truncate')
  })

  it('un double-clic ou un clic droit sur le bandeau ne crée pas de note', async () => {
    const user = userEvent.setup()
    const ids = await semer('A')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.A])
    await afficher()
    await attendreAffichage('G[A]')

    await user.dblClick(bandeau('G'))
    expect(prompt).not.toHaveBeenCalled()

    await clicDroit(bandeau('G'), user)
    expect(await screen.findByRole('menuitem', { name: 'Renommer le groupe' })).toBeInTheDocument()
    expect(screen.queryByRole('menuitem', { name: '+ Nouvelle note' })).toBeNull()
  })

  it('renomme le groupe depuis le menu du bandeau, et refuse un nom vide', async () => {
    const user = userEvent.setup()
    const ids = await semer('A', 'B')
    await demoBackend.GroupNotes(PROJET, 'Sprint 12', [ids.B])
    await afficher()
    await attendreAffichage('A', 'Sprint 12[B]')

    prompt.mockReturnValue('Sprint 13')
    await clicDroit(bandeau('Sprint 12'), user)
    await user.click(await screen.findByRole('menuitem', { name: 'Renommer le groupe' }))
    await attendreAffichage('A', 'Sprint 13[B]')
    expect(prompt).toHaveBeenCalledWith('Nouveau nom ?', 'Sprint 12')

    prompt.mockReturnValue('')
    await clicDroit(bandeau('Sprint 13'), user)
    await user.click(await screen.findByRole('menuitem', { name: 'Renommer le groupe' }))
    expect(affichage()).toEqual(['A', 'Sprint 13[B]'])
  })

  it('dissout le groupe en laissant ses notes en place', async () => {
    const user = userEvent.setup()
    const ids = await semer('A', 'B', 'C', 'D')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.B, ids.C])
    await afficher()
    await attendreAffichage('A', 'G[B,C]', 'D')

    await clicDroit(bandeau('G'), user)
    await user.click(await screen.findByRole('menuitem', { name: 'Dissoudre le groupe' }))

    await attendreAffichage('A', 'B', 'C', 'D')
  })

  it('œil fermé, filtre les notes cachées du groupe et n’affiche pas un groupe entièrement caché', async () => {
    const ids = await semer('A', 'H', 'K', 'B')
    await demoBackend.GroupNotes(PROJET, 'Mixte', [ids.A, ids.H])
    await demoBackend.GroupNotes(PROJET, 'Fantôme', [ids.K])
    await demoBackend.SetNoteHidden(ids.H, true)
    await demoBackend.SetNoteHidden(ids.K, true)

    const { unmount } = await afficher(false)
    await attendreAffichage('Mixte[A]', 'B')
    unmount()

    await afficher(true)
    await attendreAffichage('Mixte[A,H]', 'Fantôme[K]', 'B')
    expect(note('K')).toHaveClass('italic', 'opacity-50')
  })
})

describe('Réordonner par glisser-déposer', () => {
  it('remonte une note déposée sur la moitié haute d’une autre, et l’ordre est enregistré', async () => {
    await semer('A', 'B', 'C')
    await afficher()
    await attendreAffichage('A', 'B', 'C')

    glisser(note('C'), note('A'), 'haut')

    await attendreAffichage('C', 'A', 'B')
    expect((await demoBackend.GetNotes(PROJET)).map((n) => n.title)).toEqual(['C', 'A', 'B'])
  })

  it('envoie en fin de liste une note déposée sur le fond', async () => {
    await semer('A', 'B', 'C')
    await afficher()
    await attendreAffichage('A', 'B', 'C')

    glisser(note('A'), liste())

    await attendreAffichage('B', 'C', 'A')
  })

  it('montre où la note sera insérée', async () => {
    await semer('A', 'B')
    await afficher()
    await attendreAffichage('A', 'B')

    survoler(note('A'), note('B'), 'bas')
    expect(note('B').closest('li')).toHaveAttribute('data-insertion', 'after')

    evenement('dragOver', note('B'), Y.haut)
    expect(note('B').closest('li')).toHaveAttribute('data-insertion', 'before')
  })

  it('ne change rien quand la note est relâchée hors de la liste', async () => {
    await semer('A', 'B')
    await afficher()
    await attendreAffichage('A', 'B')
    const ecrire = vi.spyOn(demoBackend, 'SetNotesLayout')

    survoler(note('A'), note('B'), 'bas')
    evenement('dragEnd', note('A'))

    expect(note('B').closest('li')).not.toHaveAttribute('data-insertion')
    expect(ecrire).not.toHaveBeenCalled()
    expect(affichage()).toEqual(['A', 'B'])
  })

  it('n’écrit rien quand la note est déposée là où elle est déjà', async () => {
    await semer('A', 'B')
    await afficher()
    await attendreAffichage('A', 'B')
    const ecrire = vi.spyOn(demoBackend, 'SetNotesLayout')

    glisser(note('A'), note('B'), 'haut')

    await new Promise((r) => setTimeout(r, 0))
    expect(ecrire).not.toHaveBeenCalled()
  })

  it('œil fermé, une note cachée garde sa place', async () => {
    const ids = await semer('A', 'H', 'B', 'C')
    await demoBackend.SetNoteHidden(ids.H, true)
    await afficher()
    await attendreAffichage('A', 'B', 'C')

    glisser(note('C'), note('B'), 'haut')

    await attendreAffichage('A', 'C', 'B')
    expect((await demoBackend.GetNotes(PROJET)).map((n) => n.title)).toEqual(['A', 'H', 'C', 'B'])
  })

  it('enregistre la frappe en cours avant de réordonner', async () => {
    const user = userEvent.setup()
    await semer('A', 'B', 'C')
    await afficher()
    await user.click(note('A'))

    await user.type(screen.getByRole('textbox', { name: 'Titre de la note' }), '+')
    glisser(note('C'), note('A'), 'haut')

    await attendreAffichage('C', 'A+', 'B')
    expect((await demoBackend.GetNotes(PROJET)).map((n) => n.title)).toEqual(['C', 'A+', 'B'])
  })
})

describe('Glisser-déposer et groupes', () => {
  it('ajoute au groupe une note déposée sur son bandeau', async () => {
    const ids = await semer('A', 'B', 'C')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.A, ids.B])
    await afficher()
    await attendreAffichage('G[A,B]', 'C')

    survoler(note('C'), bandeau('G'))
    expect(screen.getByRole('group', { name: 'G' })).toHaveAttribute('data-accueille', 'true')
    evenement('drop', bandeau('G'))

    await attendreAffichage('G[A,B,C]')
  })

  it('insère la note à l’endroit visé dans le groupe', async () => {
    const ids = await semer('A', 'B', 'C')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.A, ids.B])
    await afficher()
    await attendreAffichage('G[A,B]', 'C')

    glisser(note('C'), note('B'), 'haut')

    await attendreAffichage('G[A,C,B]')
  })

  it('fait passer une note d’un groupe à l’autre', async () => {
    const ids = await semer('A', 'B', 'C')
    await demoBackend.GroupNotes(PROJET, 'G1', [ids.A, ids.B])
    await demoBackend.GroupNotes(PROJET, 'G2', [ids.C])
    await afficher()
    await attendreAffichage('G1[A,B]', 'G2[C]')

    glisser(note('A'), bandeau('G2'))

    await attendreAffichage('G1[B]', 'G2[C,A]')
  })

  it('retire du groupe une note glissée hors du cadre', async () => {
    const ids = await semer('A', 'B', 'C')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.A, ids.B])
    await afficher()
    await attendreAffichage('G[A,B]', 'C')

    glisser(note('B'), note('C'), 'bas')

    await attendreAffichage('G[A]', 'C', 'B')
  })

  it('fait disparaître le groupe quand sa dernière note en sort', async () => {
    const ids = await semer('A', 'B')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.A])
    await afficher()
    await attendreAffichage('G[A]', 'B')

    glisser(note('A'), liste())

    await attendreAffichage('B', 'A')
    expect(await demoBackend.GetNoteGroups(PROJET)).toHaveLength(0)
  })

  it('déplace le groupe entier quand on le prend par son bandeau', async () => {
    const ids = await semer('X', 'Y', 'A', 'B')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.X, ids.Y])
    await afficher()
    await attendreAffichage('G[X,Y]', 'A', 'B')

    glisser(bandeau('G'), note('B'), 'bas')

    await attendreAffichage('A', 'B', 'G[X,Y]')
  })

  it('pose un groupe relâché sur un autre groupe après lui, jamais dedans', async () => {
    const ids = await semer('X', 'A', 'B', 'C')
    await demoBackend.GroupNotes(PROJET, 'G1', [ids.X])
    await demoBackend.GroupNotes(PROJET, 'G2', [ids.A, ids.B])
    await afficher()
    await attendreAffichage('G1[X]', 'G2[A,B]', 'C')

    // Relâché sur une note de G2 : c'est le groupe entier qui est visé.
    glisser(bandeau('G1'), note('B'), 'bas')

    await attendreAffichage('G2[A,B]', 'G1[X]', 'C')
  })

  it('glisser une note d’un groupe ne déplace que cette note', async () => {
    const ids = await semer('X', 'Y', 'A', 'B')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.X, ids.Y])
    await afficher()
    await attendreAffichage('G[X,Y]', 'A', 'B')

    glisser(note('Y'), note('B'), 'bas')

    await attendreAffichage('G[X]', 'A', 'B', 'Y')
  })
})

describe('Nouvelle note', () => {
  it('arrive en tête, hors groupe, et s’ouvre dans l’éditeur', async () => {
    const user = userEvent.setup()
    const ids = await semer('A', 'B')
    await demoBackend.GroupNotes(PROJET, 'G', [ids.A])
    await afficher()
    await attendreAffichage('G[A]', 'B')
    prompt.mockReturnValue('Nouvelle')

    await clicDroit(note('B'), user)
    await user.click(await screen.findByRole('menuitem', { name: '+ Nouvelle note' }))

    await attendreAffichage('Nouvelle', 'G[A]', 'B')
    await waitFor(() => expect(screen.getByRole('textbox', { name: 'Titre de la note' })).toHaveValue('Nouvelle'))
  })
})
