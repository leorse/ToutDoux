import { describe, expect, it } from 'vitest'
import { blocs, dissolve, groupNotes, memeDisposition, moveGroup, moveNote, toLayout, type Place } from './notesLayout'

/**
 * Disposition de la liste des notes (v1.2.0) : ordre manuel et groupes.
 *
 * Les dispositions s'écrivent « A B:g C:g D » : une note par mot, son groupe
 * éventuel après les deux-points.
 */
function lire(texte: string): Place[] {
  return texte
    .split(' ')
    .filter(Boolean)
    .map((mot) => {
      const [noteId, groupId = ''] = mot.split(':')
      return { noteId, groupId }
    })
}

function ecrire(layout: Place[]): string {
  return layout.map((p) => (p.groupId ? `${p.noteId}:${p.groupId}` : p.noteId)).join(' ')
}

describe('moveNote — réordonner', () => {
  it('remonte une note déposée sur la moitié haute d’une autre', () => {
    expect(ecrire(moveNote(lire('A B C'), 'C', { type: 'note', noteId: 'A', position: 'before' }))).toBe('C A B')
  })

  it('place la note après celle sur la moitié basse de laquelle elle est déposée', () => {
    expect(ecrire(moveNote(lire('A B C'), 'A', { type: 'note', noteId: 'B', position: 'after' }))).toBe('B A C')
  })

  it('envoie en fin de liste une note déposée sur le fond', () => {
    expect(ecrire(moveNote(lire('A B C'), 'A', { type: 'end' }))).toBe('B C A')
  })

  it('ne change rien quand la note est déposée là où elle est déjà', () => {
    const depart = lire('A B C')
    expect(memeDisposition(depart, moveNote(depart, 'B', { type: 'note', noteId: 'B', position: 'after' }))).toBe(true)
    expect(memeDisposition(depart, moveNote(depart, 'B', { type: 'note', noteId: 'A', position: 'after' }))).toBe(true)
    expect(memeDisposition(depart, moveNote(depart, 'B', { type: 'note', noteId: 'C', position: 'before' }))).toBe(true)
  })

  it('laisse une note cachée à sa place : la note se pose par rapport à sa cible', () => {
    // Ordre complet A, H, B, C — H cachée et non affichée ; C glissée sur le haut de B.
    expect(ecrire(moveNote(lire('A H B C'), 'C', { type: 'note', noteId: 'B', position: 'before' }))).toBe('A H C B')
  })
})

describe('moveNote — entrer dans un groupe et en sortir', () => {
  it('ajoute en dernier une note déposée sur le bandeau du groupe', () => {
    expect(ecrire(moveNote(lire('A:g B:g C'), 'C', { type: 'group', groupId: 'g' }))).toBe('A:g B:g C:g')
  })

  it('insère la note à l’endroit visé dans le groupe', () => {
    expect(ecrire(moveNote(lire('A:g B:g C'), 'C', { type: 'note', noteId: 'B', position: 'before' }))).toBe('A:g C:g B:g')
  })

  it('fait passer une note d’un groupe à l’autre', () => {
    expect(ecrire(moveNote(lire('A:g1 B:g1 C:g2'), 'A', { type: 'group', groupId: 'g2' }))).toBe('B:g1 C:g2 A:g2')
  })

  it('réordonne à l’intérieur d’un groupe', () => {
    expect(ecrire(moveNote(lire('A:g B:g C:g'), 'C', { type: 'note', noteId: 'A', position: 'before' }))).toBe('C:g A:g B:g')
  })

  it('sort du groupe une note déposée sur une note hors groupe', () => {
    expect(ecrire(moveNote(lire('A:g B:g C'), 'B', { type: 'note', noteId: 'C', position: 'after' }))).toBe('A:g C B')
  })

  it('sort du groupe une note déposée sur le fond de la liste', () => {
    expect(ecrire(moveNote(lire('A:g B:g C'), 'A', { type: 'end' }))).toBe('B:g C A')
  })

  it('ne change rien quand la seule note d’un groupe est déposée sur son propre bandeau', () => {
    const depart = lire('A:g B')
    expect(moveNote(depart, 'A', { type: 'group', groupId: 'g' })).toBe(depart)
  })
})

describe('moveGroup', () => {
  it('pose le groupe sous une note, ses notes gardant leur ordre', () => {
    expect(ecrire(moveGroup(lire('X:g Y:g A B'), 'g', { type: 'note', noteId: 'B', position: 'after' }))).toBe('A B X:g Y:g')
  })

  it('pose le groupe au-dessus d’une note', () => {
    expect(ecrire(moveGroup(lire('A B X:g'), 'g', { type: 'note', noteId: 'A', position: 'before' }))).toBe('X:g A B')
  })

  it('pose le groupe après un autre groupe, jamais dedans', () => {
    expect(ecrire(moveGroup(lire('X:g1 A:g2 B:g2 C'), 'g1', { type: 'group', groupId: 'g2', position: 'after' }))).toBe('A:g2 B:g2 X:g1 C')
    expect(ecrire(moveGroup(lire('A:g2 B:g2 C X:g1'), 'g1', { type: 'group', groupId: 'g2', position: 'before' }))).toBe('X:g1 A:g2 B:g2 C')
  })

  it('prend le groupe entier pour cible quand on vise une de ses notes', () => {
    expect(ecrire(moveGroup(lire('X:g1 A:g2 B:g2'), 'g1', { type: 'note', noteId: 'A', position: 'after' }))).toBe('A:g2 B:g2 X:g1')
  })

  it('envoie le groupe en fin de liste', () => {
    expect(ecrire(moveGroup(lire('X:g Y:g A'), 'g', { type: 'end' }))).toBe('A X:g Y:g')
  })

  it('emporte les notes cachées du groupe', () => {
    expect(ecrire(moveGroup(lire('X:g H:g A'), 'g', { type: 'note', noteId: 'A', position: 'after' }))).toBe('A X:g H:g')
  })
})

describe('groupNotes et dissolve', () => {
  it('rassemble la sélection à la place de sa première note', () => {
    expect(ecrire(groupNotes(lire('A B C D'), ['D', 'A', 'C'], 'n'))).toBe('A:n C:n D:n B')
  })

  it('retire d’un autre groupe les notes regroupées', () => {
    expect(ecrire(groupNotes(lire('A:g1 B:g1 C'), ['B', 'C'], 'n'))).toBe('A:g1 B:n C:n')
  })

  it('ne coupe pas en deux le groupe d’où vient la première note', () => {
    expect(ecrire(groupNotes(lire('A:g1 B:g1 C:g1 X'), ['B', 'X'], 'n'))).toBe('A:g1 C:g1 B:n X:n')
  })

  it('dissout un groupe en laissant ses notes en place', () => {
    expect(ecrire(dissolve(lire('A B:g C:g D'), 'g'))).toBe('A B C D')
  })
})

describe('blocs', () => {
  const notes = [
    { id: 'A' },
    { id: 'B', groupId: 'g' },
    { id: 'H', groupId: 'g', hidden: true },
    { id: 'C' },
    { id: 'K', groupId: 'h', hidden: true },
  ]

  it('range les notes contiguës d’un même groupe dans un seul bloc', () => {
    const resultat = blocs(notes, true)
    expect(resultat.map((b) => (b.type === 'note' ? b.note.id : `${b.groupId}[${b.notes.map((n) => n.id)}]`))).toEqual([
      'A',
      'g[B,H]',
      'C',
      'h[K]',
    ])
  })

  it('œil fermé, filtre les notes cachées et n’affiche pas un groupe entièrement caché', () => {
    const resultat = blocs(notes, false)
    expect(resultat.map((b) => (b.type === 'note' ? b.note.id : `${b.groupId}[${b.notes.map((n) => n.id)}]`))).toEqual([
      'A',
      'g[B]',
      'C',
    ])
  })

  it('dérive la disposition complète des notes', () => {
    expect(ecrire(toLayout(notes))).toBe('A B:g H:g C K:h')
  })
})
