import type { Fragment, Mark, Node as PMNode } from '@tiptap/pm/model'

/**
 * Rendu Markdown d'un contenu de l'éditeur riche (v1.2.1).
 *
 * C'est la forme texte brut du presse-papier : coller une note dans VS Code
 * doit rendre une liste à puces, pas trois lignes séparées de blancs. La forme
 * HTML, elle, reste celle de ProseMirror.
 *
 * Le jeu de nœuds est celui de StarterKit plus les images, donc fermé : un
 * parcours écrit à la main suffit, sans dépendance. Le texte est copié tel que
 * tapé, sans échappement — la lisibilité du collage prime.
 */
export function fragmentToMarkdown(fragment: Fragment): string {
  if (fragment.childCount === 0) return ''
  // Une sélection à l'intérieur d'un paragraphe peut arriver sous forme de
  // contenu en ligne seul.
  if (fragment.firstChild!.isInline) return enLigne(fragment)
  return blocs(fragment)
}

/**
 * Blocs successifs, séparés par une ligne vide — sauf des éléments de liste
 * voisins, qui arrivent sans leur liste quand la sélection est à cheval sur
 * plusieurs d'entre eux, et restent serrés. Un bloc vide (le paragraphe que
 * l'éditeur garde en fin de document, une ligne laissée blanche) ne produit
 * rien : en Markdown, des lignes vides en plus ne disent rien de plus.
 */
function blocs(fragment: Fragment): string {
  let sortie = ''
  let precedent: string | null = null
  fragment.forEach((node) => {
    const rendu = bloc(node)
    if (!rendu) return
    if (precedent !== null) sortie += precedent === 'listItem' && node.type.name === 'listItem' ? '\n' : '\n\n'
    sortie += rendu
    precedent = node.type.name
  })
  return sortie
}

function bloc(node: PMNode): string {
  switch (node.type.name) {
    case 'paragraph':
      return enLigne(node.content)
    case 'heading':
      return `${'#'.repeat(node.attrs.level ?? 1)} ${enLigne(node.content)}`
    case 'bulletList':
      return liste(node, () => '* ')
    case 'orderedList': {
      const debut = Number(node.attrs.start ?? 1)
      return liste(node, (i) => `${debut + i}. `)
    }
    case 'listItem':
      // Élément isolé de sa liste : rendu comme une puce.
      return element(node, '* ')
    case 'blockquote':
      return blocs(node.content)
        .split('\n')
        .map((ligne) => (ligne ? `> ${ligne}` : '>'))
        .join('\n')
    case 'codeBlock':
      return '```' + (node.attrs.language ?? '') + '\n' + node.textContent + '\n```'
    case 'horizontalRule':
      return '---'
    case 'image':
      return image(node)
    default:
      // Nœud inconnu : son contenu, plutôt que rien.
      return node.isTextblock ? enLigne(node.content) : blocs(node.content)
  }
}

/** Liste serrée : aucun blanc entre les éléments. */
function liste(node: PMNode, marqueur: (i: number) => string): string {
  const elements: string[] = []
  node.forEach((item, _offset, i) => elements.push(element(item, marqueur(i))))
  return elements.join('\n')
}

/**
 * Élément de liste : le marqueur en tête, les lignes suivantes indentées de la
 * largeur du marqueur — c'est ce qui rattache une sous-liste à son parent.
 */
function element(item: PMNode, marqueur: string): string {
  const parties: string[] = []
  item.forEach((enfant, _offset, i) => {
    const rendu = bloc(enfant)
    if (i === 0) parties.push(rendu)
    else parties.push((estListe(enfant) ? '\n' : '\n\n') + rendu)
  })
  const retrait = ' '.repeat(marqueur.length)
  return (
    marqueur +
    parties
      .join('')
      .split('\n')
      .map((ligne, i) => (i === 0 || !ligne ? ligne : retrait + ligne))
      .join('\n')
  )
}

function estListe(node: PMNode) {
  return node.type.name === 'bulletList' || node.type.name === 'orderedList'
}

function image(node: PMNode): string {
  return `![${node.attrs.alt || 'image'}](${node.attrs.src ?? ''})`
}

/**
 * Marques rendues, de la plus extérieure à la plus intérieure. Le code est en
 * dernier : rien ne s'emboîte dans un `code`. Le souligné n'a pas de forme
 * Markdown et reste du texte simple.
 */
const ORDRE_MARQUES = ['link', 'bold', 'italic', 'strike', 'code']

function ouvrir(m: Mark): string {
  switch (m.type.name) {
    case 'link':
      return '['
    case 'bold':
      return '**'
    case 'italic':
      return '*'
    case 'strike':
      return '~~'
    default:
      return '`'
  }
}

function fermer(m: Mark): string {
  return m.type.name === 'link' ? `](${m.attrs.href ?? ''})` : ouvrir(m)
}

/**
 * Contenu en ligne. Les marques sont ouvertes et fermées par plages maximales :
 * deux nœuds texte voisins en gras donnent `**a b**`, pas `**a** **b**`.
 */
function enLigne(contenu: Fragment): string {
  let sortie = ''
  const pile: Mark[] = []
  contenu.forEach((node) => {
    const marques = node.marks
      .filter((m) => ORDRE_MARQUES.includes(m.type.name))
      .sort((a, b) => ORDRE_MARQUES.indexOf(a.type.name) - ORDRE_MARQUES.indexOf(b.type.name))
    let communes = 0
    while (communes < pile.length && communes < marques.length && pile[communes].eq(marques[communes])) communes++
    while (pile.length > communes) sortie += fermer(pile.pop()!)
    for (const m of marques.slice(communes)) {
      sortie += ouvrir(m)
      pile.push(m)
    }
    if (node.isText) sortie += node.text
    else if (node.type.name === 'hardBreak') sortie += '\\\n'
    else if (node.type.name === 'image') sortie += image(node)
    else sortie += node.textContent
  })
  while (pile.length) sortie += fermer(pile.pop()!)
  return sortie
}
