/**
 * Disposition de la liste des notes d'un projet (v1.2.0).
 *
 * Les notes d'un projet ont un ordre unique, dans lequel celles d'un même
 * groupe sont contiguës : un groupe n'a pas de position propre, il se trouve là
 * où sont ses notes. Ce module calcule la disposition qui résulte d'un
 * glisser-déposer ; le backend la vérifie et l'enregistre (`SetNotesLayout`).
 *
 * Tout se calcule sur la liste **complète**, notes cachées comprises : c'est ce
 * qui laisse une note cachée à sa place quand on réordonne œil fermé.
 */

/** Une note à sa place dans la liste. `groupId` est vide hors groupe. */
export type Place = { noteId: string; groupId: string }

/** Ce qu'il faut savoir d'une note pour la disposer. */
type NoteDisposable = { id: string; groupId?: string | null; hidden?: boolean }

export type Position = 'before' | 'after'

/**
 * Endroit où l'on dépose.
 * - `note` : avant ou après une note ; la note déposée prend son groupe.
 * - `group` sans position : dans le groupe, en dernier. Avec position : avant
 *   ou après le groupe entier (déplacement d'un groupe).
 * - `end` : en fin de liste, hors groupe.
 */
export type Cible =
  | { type: 'note'; noteId: string; position: Position }
  | { type: 'group'; groupId: string; position?: Position }
  | { type: 'end' }

export function toLayout(notes: NoteDisposable[]): Place[] {
  return notes.map((n) => ({ noteId: n.id, groupId: n.groupId ?? '' }))
}

export function memeDisposition(a: Place[], b: Place[]): boolean {
  return a.length === b.length && a.every((p, i) => p.noteId === b[i].noteId && p.groupId === b[i].groupId)
}

/** Indice qui suit la dernière note du groupe, ou -1 si le groupe est absent. */
function finDuGroupe(layout: Place[], groupId: string): number {
  for (let i = layout.length - 1; i >= 0; i--) if (layout[i].groupId === groupId) return i + 1
  return -1
}

function inserer(layout: Place[], indice: number, places: Place[]): Place[] {
  return [...layout.slice(0, indice), ...places, ...layout.slice(indice)]
}

/**
 * Déplace une note. Déposée sur une note d'un groupe ou sur un groupe, elle
 * y entre ; déposée ailleurs, elle sort du sien. Rend la disposition
 * inchangée si la cible n'existe pas (ou est la note elle-même).
 */
export function moveNote(layout: Place[], noteId: string, cible: Cible): Place[] {
  if (!layout.some((p) => p.noteId === noteId)) return layout
  const reste = layout.filter((p) => p.noteId !== noteId)

  if (cible.type === 'end') return [...reste, { noteId, groupId: '' }]

  if (cible.type === 'group') {
    const fin = finDuGroupe(reste, cible.groupId)
    if (fin < 0) return layout
    return inserer(reste, fin, [{ noteId, groupId: cible.groupId }])
  }

  const indice = reste.findIndex((p) => p.noteId === cible.noteId)
  if (indice < 0) return layout
  const place = { noteId, groupId: reste[indice].groupId }
  return inserer(reste, cible.position === 'before' ? indice : indice + 1, [place])
}

/**
 * Déplace un groupe entier, ses notes gardant leur ordre. Un groupe ne se
 * dépose jamais dans un autre : visé par une de ses notes, le groupe cible est
 * pris en entier.
 */
export function moveGroup(layout: Place[], groupId: string, cible: Cible): Place[] {
  const bloc = layout.filter((p) => p.groupId === groupId)
  if (bloc.length === 0) return layout
  const reste = layout.filter((p) => p.groupId !== groupId)

  if (cible.type === 'end') return [...reste, ...bloc]

  let groupeCible = cible.type === 'group' ? cible.groupId : ''
  const position = cible.position ?? 'after'
  if (cible.type === 'note') {
    const indice = reste.findIndex((p) => p.noteId === cible.noteId)
    if (indice < 0) return layout
    if (!reste[indice].groupId) return inserer(reste, position === 'before' ? indice : indice + 1, bloc)
    groupeCible = reste[indice].groupId
  }

  const debut = reste.findIndex((p) => p.groupId === groupeCible)
  if (debut < 0) return layout
  return inserer(reste, position === 'before' ? debut : finDuGroupe(reste, groupeCible), bloc)
}

/**
 * Rassemble des notes dans un nouveau groupe, à la place de la première
 * d'entre elles. Pendant de `notelayout.Group` côté Go, pour le backend de
 * démonstration.
 */
export function groupNotes(layout: Place[], noteIds: string[], groupId: string): Place[] {
  const choisies = new Set(noteIds)
  const reste = layout.filter((p) => !choisies.has(p.noteId))
  const prises = layout.filter((p) => choisies.has(p.noteId)).map((p) => ({ noteId: p.noteId, groupId }))
  if (prises.length === 0) return layout

  let indice = layout.findIndex((p) => choisies.has(p.noteId))
  indice = layout.slice(0, indice).filter((p) => !choisies.has(p.noteId)).length
  // Jamais au milieu d'un groupe qui garde des notes de part et d'autre.
  while (indice > 0 && indice < reste.length && reste[indice - 1].groupId && reste[indice - 1].groupId === reste[indice].groupId) {
    indice++
  }
  return inserer(reste, indice, prises)
}

/** Sort toutes les notes d'un groupe, sans les déplacer. */
export function dissolve(layout: Place[], groupId: string): Place[] {
  return layout.map((p) => (p.groupId === groupId ? { ...p, groupId: '' } : p))
}

/** Ce que la liste affiche : une note hors groupe, ou un groupe et ses notes. */
export type Bloc<N> = { type: 'note'; note: N } | { type: 'group'; groupId: string; notes: N[] }

/**
 * Découpe la liste ordonnée en blocs d'affichage. Œil fermé, les notes cachées
 * sont écartées, et un groupe dont aucune note n'est affichée ne l'est pas.
 */
export function blocs<N extends NoteDisposable>(notes: N[], montrerCaches: boolean): Bloc<N>[] {
  const resultat: Bloc<N>[] = []
  for (const note of notes) {
    if (note.hidden && !montrerCaches) continue
    const dernier = resultat[resultat.length - 1]
    if (!note.groupId) {
      resultat.push({ type: 'note', note })
    } else if (dernier?.type === 'group' && dernier.groupId === note.groupId) {
      dernier.notes.push(note)
    } else {
      resultat.push({ type: 'group', groupId: note.groupId, notes: [note] })
    }
  }
  return resultat
}
