import { act, fireEvent, render, screen } from '@testing-library/react'
import type { Editor } from '@tiptap/core'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { RichEditor } from './RichEditor'

/**
 * La barre d'outils suit le curseur (correctif 1.1.1).
 *
 * En 1.1.0, l'état « enfoncé » des boutons n'était recalculé qu'à la frappe :
 * TipTap 3 ne relance pas le rendu sur un simple déplacement du curseur. Les
 * tests déplacent donc le curseur **sans rien taper**.
 */

const CONTENU = '<p>plain <strong>bold</strong> <em><strong>both</strong></em></p><ul><li><p>item</p></li></ul>'

function editeur(): Editor {
  const dom = document.querySelector('.ProseMirror') as (HTMLElement & { editor?: Editor }) | null
  if (!dom?.editor) throw new Error('éditeur introuvable')
  return dom.editor
}

/** Place le curseur au milieu du premier mot `mot` du document. */
function curseurDans(mot: string) {
  const ed = editeur()
  let cible: number | null = null
  ed.state.doc.descendants((node, pos) => {
    if (cible !== null || !node.isText) return
    const i = node.text!.indexOf(mot)
    if (i >= 0) cible = pos + i + Math.floor(mot.length / 2)
  })
  if (cible === null) throw new Error(`« ${mot} » introuvable`)
  act(() => {
    ed.commands.setTextSelection(cible!)
  })
}

const bouton = (nom: string) => screen.getByRole('button', { name: nom })

function enfonces() {
  return ['Gras', 'Italique', 'Barré', 'Liste à puces'].filter(
    (nom) => bouton(nom).getAttribute('aria-pressed') === 'true',
  )
}

describe('Barre d’outils de l’éditeur (1.1.1)', () => {
  it('suit le curseur d’un format à l’autre sans frappe', () => {
    render(<RichEditor content={CONTENU} onChange={() => {}} />)

    curseurDans('bold')
    expect(enfonces()).toEqual(['Gras'])

    curseurDans('plain')
    expect(enfonces()).toEqual([])

    curseurDans('both')
    expect(enfonces()).toEqual(['Gras', 'Italique'])

    curseurDans('item')
    expect(enfonces()).toEqual(['Liste à puces'])

    curseurDans('plain')
    expect(enfonces()).toEqual([])
  })

  it('aligne le style enfoncé sur aria-pressed', () => {
    render(<RichEditor content={CONTENU} onChange={() => {}} />)

    curseurDans('bold')
    expect(bouton('Gras')).toHaveClass('border-2')
    expect(bouton('Italique')).not.toHaveClass('border-2')

    curseurDans('plain')
    expect(bouton('Gras')).toHaveAttribute('aria-pressed', 'false')
    expect(bouton('Gras')).not.toHaveClass('border-2')
  })

  it('suit aussi le curseur en plein écran modifiable', () => {
    render(<RichEditor content={CONTENU} onChange={() => {}} />)
    fireEvent.mouseDown(bouton('Agrandir en plein écran'))
    expect(bouton('Fermer le plein écran')).toBeInTheDocument()

    curseurDans('both')
    expect(bouton('Italique')).toHaveAttribute('aria-pressed', 'true')

    curseurDans('plain')
    expect(bouton('Italique')).toHaveAttribute('aria-pressed', 'false')
  })
})

/* ---------------- Copie en Markdown et ligne de séparation (1.2.1) ---------------- */

/** Presse-papier factice d'un événement copy/cut : jsdom n'a pas de DataTransfer. */
function pressePapier() {
  const donnees = new Map<string, string>()
  return {
    donnees,
    clipboardData: {
      setData: (type: string, valeur: string) => void donnees.set(type, valeur),
      getData: (type: string) => donnees.get(type) ?? '',
      clearData: () => donnees.clear(),
      types: [] as string[],
    },
  }
}

const surface = () => document.querySelector('.ProseMirror') as HTMLElement

/** Sélectionne le texte du début de `debut` à la fin de `fin`. */
function selectionner(debut: string, fin: string) {
  const ed = editeur()
  let de = -1
  let a = -1
  ed.state.doc.descendants((node, pos) => {
    if (!node.isText) return
    const i = node.text!.indexOf(debut)
    if (de < 0 && i >= 0) de = pos + i
    const j = node.text!.indexOf(fin)
    if (j >= 0) a = pos + j + fin.length
  })
  act(() => {
    ed.commands.setTextSelection({ from: de, to: a })
  })
}

/** Remplace `document.execCommand('copy')` par une vraie copie sur un presse-papier factice. */
function intercepterCopie() {
  const pp = pressePapier()
  const appel = vi.fn((commande: string) => {
    if (commande === 'copy') fireEvent.copy(surface(), { clipboardData: pp.clipboardData })
    return true
  })
  document.execCommand = appel as unknown as typeof document.execCommand
  return { pp, appel }
}

/** API asynchrone du presse-papier, absente de jsdom. */
function poserPressePapier(api: object) {
  Object.defineProperty(navigator, 'clipboard', { value: api, configurable: true })
}

/** Position de la première image du document. */
function positionImage(): number {
  let trouve = -1
  editeur().state.doc.descendants((node, pos) => {
    if (trouve < 0 && node.type.name === 'image') trouve = pos
  })
  return trouve
}

const LISTE = '<ul><li><p>text1</p></li><li><p>text2</p></li><li><p>text3</p></li></ul>'
const IMAGE = 'data:image/png;base64,iVBORw0KGgo='

describe('Copie en Markdown (1.2.1)', () => {
  afterEach(() => {
    Reflect.deleteProperty(navigator, 'clipboard')
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('Ctrl+C sur une liste met du Markdown en texte brut et garde le HTML', () => {
    render(<RichEditor content={LISTE} onChange={() => {}} />)
    selectionner('text1', 'text3')
    const pp = pressePapier()

    fireEvent.copy(surface(), { clipboardData: pp.clipboardData })

    expect(pp.donnees.get('text/plain')).toBe('* text1\n* text2\n* text3')
    expect(pp.donnees.get('text/html')).toContain('<li>')
  })

  it('Ctrl+X retire la sélection avec le même texte brut', () => {
    render(<RichEditor content="<h2>Titre</h2><p>reste</p>" onChange={() => {}} />)
    selectionner('Titre', 'Titre')
    const pp = pressePapier()

    fireEvent.cut(surface(), { clipboardData: pp.clipboardData })

    expect(pp.donnees.get('text/plain')).toBe('## Titre')
    expect(editeur().getHTML()).not.toContain('Titre')
  })

  it('une image seule part aussi en PNG', () => {
    const write = vi.fn((_items: unknown[]) => Promise.resolve())
    poserPressePapier({ write })
    vi.stubGlobal(
      'ClipboardItem',
      class {
        constructor(public items: Record<string, unknown>) {}
      },
    )
    render(<RichEditor content={`<p>avant</p><img src="${IMAGE}">`} onChange={() => {}} />)
    act(() => {
      editeur().commands.setNodeSelection(positionImage())
    })
    const pp = pressePapier()

    fireEvent.copy(surface(), { clipboardData: pp.clipboardData })

    expect(pp.donnees.get('text/plain')).toBe(`![image](${IMAGE})`)
    expect(write).toHaveBeenCalledOnce()
    const item = write.mock.calls[0][0][0] as { items: Record<string, unknown> }
    expect(Object.keys(item.items).sort()).toEqual(['image/png', 'text/html', 'text/plain'])
  })

  it('une écriture refusée laisse la copie ordinaire, sans erreur', async () => {
    const write = vi.fn(() => Promise.reject(new Error('refusé')))
    poserPressePapier({ write })
    vi.stubGlobal('ClipboardItem', class {})
    render(<RichEditor content={`<img src="${IMAGE}">`} onChange={() => {}} />)
    act(() => {
      editeur().commands.setNodeSelection(positionImage())
    })
    const pp = pressePapier()

    fireEvent.copy(surface(), { clipboardData: pp.clipboardData })
    await Promise.resolve()

    expect(write).toHaveBeenCalledOnce()
    expect(pp.donnees.get('text/plain')).toBe(`![image](${IMAGE})`)
  })

  it('du texte avec une image ne passe pas par l’écriture asynchrone', () => {
    const write = vi.fn(() => Promise.resolve())
    poserPressePapier({ write })
    vi.stubGlobal('ClipboardItem', class {})
    render(<RichEditor content={`<p>avant</p><img src="${IMAGE}"><p>après</p>`} onChange={() => {}} />)
    selectionner('avant', 'après')
    const pp = pressePapier()

    fireEvent.copy(surface(), { clipboardData: pp.clipboardData })

    expect(write).not.toHaveBeenCalled()
    expect(pp.donnees.get('text/plain')).toBe(`avant\n\n![image](${IMAGE})\n\naprès`)
  })

  it('le clic droit propose « Copier en Markdown », qui copie la sélection', () => {
    render(<RichEditor content={LISTE} onChange={() => {}} />)
    const { pp, appel } = intercepterCopie()
    selectionner('text1', 'text3')

    fireEvent.contextMenu(surface())
    fireEvent.click(screen.getByRole('menuitem', { name: 'Copier en Markdown' }))

    expect(appel).toHaveBeenCalledWith('copy')
    expect(pp.donnees.get('text/plain')).toBe('* text1\n* text2\n* text3')
  })

  it('sans sélection, l’entrée est grisée', () => {
    render(<RichEditor content={LISTE} onChange={() => {}} />)
    curseurDans('text2')
    fireEvent.contextMenu(surface())
    expect(screen.getByRole('menuitem', { name: 'Copier en Markdown' })).toBeDisabled()
  })

  it('le menu existe aussi en plein écran lecture seule', () => {
    render(<RichEditor content={LISTE} onChange={() => {}} />)
    fireEvent.mouseDown(bouton('Afficher en plein écran (lecture seule)'))
    fireEvent.contextMenu(surface())
    expect(screen.getByRole('menuitem', { name: 'Copier en Markdown' })).toBeInTheDocument()
  })

  it('« Copier tout en Markdown » copie tout et rend le curseur', () => {
    vi.useFakeTimers()
    render(<RichEditor content="<h2>Titre</h2><ul><li><p>a</p></li><li><p>b</p></li></ul>" onChange={() => {}} />)
    const { pp } = intercepterCopie()
    curseurDans('Titre')
    const avant = editeur().state.selection.from

    expect(bouton('Copier tout en Markdown')).toHaveTextContent(/^$/)
    expect(bouton('Copier tout en Markdown')).not.toHaveAttribute('aria-pressed')
    fireEvent.mouseDown(bouton('Copier tout en Markdown'))

    expect(pp.donnees.get('text/plain')).toBe('## Titre\n\n* a\n* b')
    expect(editeur().state.selection.from).toBe(avant)
    expect(editeur().state.selection.empty).toBe(true)
    expect(bouton('Copier tout en Markdown')).toHaveTextContent('✓')

    act(() => {
      vi.advanceTimersByTime(1500)
    })
    expect(bouton('Copier tout en Markdown')).toHaveTextContent(/^$/)
  })

  it('« Copier tout » sur un éditeur vide vide le presse-papier sans erreur', () => {
    const writeText = vi.fn(() => Promise.resolve())
    poserPressePapier({ writeText })
    render(<RichEditor content="" onChange={() => {}} />)
    fireEvent.mouseDown(bouton('Copier tout en Markdown'))
    expect(writeText).toHaveBeenCalledWith('')
  })
})

describe('Ligne de séparation (1.2.1)', () => {
  it('insère une ligne au curseur, la frappe continue en dessous', () => {
    const onChange = vi.fn()
    render(<RichEditor content="<p>Partie 1</p>" onChange={onChange} />)
    act(() => {
      editeur().commands.setTextSelection(editeur().state.doc.content.size - 1)
    })

    fireEvent.mouseDown(bouton('Ligne de séparation'))
    act(() => {
      editeur().commands.insertContent('Partie 2')
    })

    expect(onChange).toHaveBeenLastCalledWith('<p>Partie 1</p><hr><p>Partie 2</p>')
  })

  it('n’est jamais enfoncé', () => {
    render(<RichEditor content="<p>a</p><hr><p>b</p>" onChange={() => {}} />)
    curseurDans('b')
    expect(bouton('Ligne de séparation')).not.toHaveAttribute('aria-pressed')
  })

  it('est absent en lecture seule', () => {
    render(<RichEditor content="<p>a</p>" onChange={() => {}} readOnly />)
    expect(screen.queryByRole('button', { name: 'Ligne de séparation' })).not.toBeInTheDocument()
  })
})
