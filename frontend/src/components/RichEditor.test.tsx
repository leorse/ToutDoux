import { act, fireEvent, render, screen } from '@testing-library/react'
import type { Editor } from '@tiptap/core'
import { describe, expect, it } from 'vitest'
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
