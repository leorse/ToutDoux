import { getSchema } from '@tiptap/core'
import Image from '@tiptap/extension-image'
import { DOMParser as PMDOMParser, type Node as PMNode } from '@tiptap/pm/model'
import StarterKit from '@tiptap/starter-kit'
import { describe, expect, it } from 'vitest'
import { fragmentToMarkdown } from './markdown'

const schema = getSchema([StarterKit, Image.configure({ allowBase64: true })])

function doc(html: string): PMNode {
  const conteneur = document.createElement('div')
  conteneur.innerHTML = html
  return PMDOMParser.fromSchema(schema).parse(conteneur)
}

const md = (html: string) => fragmentToMarkdown(doc(html).content)

/** Position juste avant la première occurrence de `mot` dans le document. */
function pos(d: PMNode, mot: string): number {
  let trouve = -1
  d.descendants((node, p) => {
    if (trouve >= 0 || !node.isText) return
    const i = node.text!.indexOf(mot)
    if (i >= 0) trouve = p + i
  })
  if (trouve < 0) throw new Error(`« ${mot} » introuvable`)
  return trouve
}

const IMAGE = 'data:image/png;base64,iVBORw0KGgo='

describe('fragmentToMarkdown', () => {
  it('liste à puces serrée', () => {
    expect(md('<ul><li><p>text1</p></li><li><p>text2</p></li><li><p>text3</p></li></ul>')).toBe(
      '* text1\n* text2\n* text3',
    )
  })

  it('titres par niveau, blocs séparés d’une ligne vide', () => {
    expect(md('<h2>Titre</h2><p>Texte</p>')).toBe('## Titre\n\nTexte')
    expect(md('<h1>A</h1><h3>B</h3>')).toBe('# A\n\n### B')
  })

  it('liste numérotée avec sous-liste à puces', () => {
    expect(md('<ol><li><p>Un</p><ul><li><p>a</p></li></ul></li><li><p>Deux</p></li></ol>')).toBe(
      '1. Un\n   * a\n2. Deux',
    )
  })

  it('liste numérotée commençant à 3', () => {
    expect(md('<ol start="3"><li><p>c</p></li><li><p>d</p></li></ol>')).toBe('3. c\n4. d')
  })

  it('trois niveaux d’imbrication', () => {
    expect(md('<ul><li><p>a</p><ul><li><p>b</p><ul><li><p>c</p></li></ul></li></ul></li></ul>')).toBe(
      '* a\n  * b\n    * c',
    )
  })

  it('image sans texte alternatif', () => {
    expect(md(`<p>Avant</p><img src="${IMAGE}">`)).toBe(`Avant\n\n![image](${IMAGE})`)
  })

  it('image avec texte alternatif', () => {
    expect(md(`<img src="${IMAGE}" alt="schéma">`)).toBe(`![schéma](${IMAGE})`)
  })

  it('bloc de code clôturé, contenu intact', () => {
    expect(md('<pre><code>a  *  b</code></pre>')).toBe('```\na  *  b\n```')
    expect(md('<pre><code class="language-go">x := 1</code></pre>')).toBe('```go\nx := 1\n```')
  })

  it('marques en ligne', () => {
    expect(md('<p><strong>g</strong> <em>i</em> <s>b</s> <code>c</code> <u>s</u></p>')).toBe(
      '**g** *i* ~~b~~ `c` s',
    )
  })

  it('une plage de gras sur des nœuds voisins n’est ouverte qu’une fois', () => {
    expect(md('<p><strong>a <em>b</em> c</strong></p>')).toBe('**a *b* c**')
  })

  it('lien au texte en gras', () => {
    expect(md('<p><a href="https://x.fr">voir <strong>ici</strong></a></p>')).toBe('[voir **ici**](https://x.fr)')
  })

  it('citation, ligne horizontale, retour à la ligne forcé', () => {
    expect(md('<blockquote><p>a</p><p>b</p></blockquote><hr><p>x<br>y</p>')).toBe('> a\n>\n> b\n\n---\n\nx\\\ny')
  })

  it('texte copié tel que tapé, sans échappement', () => {
    expect(md('<p>2*3 # _x_</p>')).toBe('2*3 # _x_')
  })

  it('les paragraphes vides ne laissent pas de lignes en trop', () => {
    expect(md('<ul><li><p>a</p></li></ul><p></p>')).toBe('* a')
    expect(md('<p>a</p><p></p><p></p><p>b</p>')).toBe('a\n\nb')
  })

  it('fragment vide', () => {
    expect(md('')).toBe('')
  })

  it('sélection au milieu d’un paragraphe : pas de lignes vides autour', () => {
    const d = doc('<p>Ceci est très <strong>important</strong> ici</p>')
    const tranche = d.slice(pos(d, 'très'), pos(d, 'important') + 'important'.length)
    expect(fragmentToMarkdown(tranche.content)).toBe('très **important**')
  })

  it('sélection à cheval sur deux éléments de liste', () => {
    const d = doc('<ul><li><p>text1</p></li><li><p>text2</p></li><li><p>text3</p></li></ul>')
    const tranche = d.slice(pos(d, 'xt1'), pos(d, 'text2') + 5)
    expect(fragmentToMarkdown(tranche.content)).toBe('* xt1\n* text2')
  })
})
