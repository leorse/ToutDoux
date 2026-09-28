import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { formaterTexte } from './texteEnrichi'

/** Rend le texte dans un <p> et rend son HTML, pour comparer la structure. */
function html(texte: string): string {
  const { container } = render(<p>{formaterTexte(texte)}</p>)
  return container.innerHTML
}

describe('formaterTexte', () => {
  it('rend du texte simple tel quel', () => {
    expect(html('Rien de spécial')).toBe('<p>Rien de spécial</p>')
  })

  it('met en gras', () => {
    expect(html('un **mot** important')).toBe('<p>un <strong>mot</strong> important</p>')
  })

  it('met en italique', () => {
    expect(html('un *mot* important')).toBe('<p>un <em>mot</em> important</p>')
  })

  it('combine gras et italique côte à côte', () => {
    expect(html('**gras** et *italique*')).toBe('<p><strong>gras</strong> et <em>italique</em></p>')
  })

  it('accepte de l’italique dans du gras', () => {
    expect(html('**gras *italique* gras**')).toBe(
      '<p><strong>gras <em>italique</em> gras</strong></p>',
    )
  })

  it('laisse en texte les marqueurs non fermés', () => {
    expect(html('un **gras non fermé')).toBe('<p>un **gras non fermé</p>')
    expect(html('une * étoile seule')).toBe('<p>une * étoile seule</p>')
  })

  it('ne prend pas une multiplication pour de l’italique', () => {
    expect(html('2 * 3 * 4')).toBe('<p>2 * 3 * 4</p>')
  })

  it('n’interprète pas le HTML', () => {
    const { container } = render(<p>{formaterTexte('<b>x</b> et <img src=x onerror=alert(1)>')}</p>)
    expect(container.querySelector('b')).toBeNull()
    expect(container.querySelector('img')).toBeNull()
    expect(container.textContent).toBe('<b>x</b> et <img src=x onerror=alert(1)>')
  })

  it('rend une liste vide pour un texte vide', () => {
    expect(formaterTexte('')).toEqual([])
  })
})
