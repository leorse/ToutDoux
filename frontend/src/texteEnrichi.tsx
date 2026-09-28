import type { ReactNode } from 'react'

// Le marqueur ouvrant doit être suivi, et le fermant précédé, d'un caractère
// non blanc : « 2 * 3 * 4 » reste du texte, comme en Markdown.
const MARQUES = /\*\*(?=\S)(.+?)(?<=\S)\*\*|\*(?=\S)(.+?)(?<=\S)\*/g

/**
 * Met en forme le texte des notes de version : `**gras**` et `*italique*`.
 *
 * Le résultat est une liste de nœuds React, jamais une chaîne HTML : tout le
 * reste, y compris une balise `<b>`, s'affiche tel quel et n'est pas interprété.
 */
export function formaterTexte(texte: string): ReactNode[] {
  const noeuds: ReactNode[] = []
  let dernier = 0
  for (const m of texte.matchAll(MARQUES)) {
    const debut = m.index
    if (debut > dernier) noeuds.push(texte.slice(dernier, debut))
    noeuds.push(
      m[1] !== undefined ? (
        <strong key={debut}>{formaterTexte(m[1])}</strong>
      ) : (
        <em key={debut}>{formaterTexte(m[2])}</em>
      ),
    )
    dernier = debut + m[0].length
  }
  if (dernier < texte.length) noeuds.push(texte.slice(dernier))
  return noeuds
}
