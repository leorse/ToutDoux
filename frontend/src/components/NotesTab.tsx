import { useCallback, useEffect, useRef, useState, type CSSProperties, type DragEvent, type MouseEvent } from 'react'
import { api } from '../api'
import { teinte } from '../couleurs'
import { surLeFond } from '../fond'
import { blocs, memeDisposition, moveGroup, moveNote, toLayout, type Cible, type Position } from '../notesLayout'
import type { domain } from '../../wailsjs/go/models'
import { useAutosave } from '../useAutosave'
import { ContextMenu, type MenuState } from './ContextMenu'
import { HiddenToggle } from './HiddenToggle'
import { RichEditor } from './RichEditor'
import { SemanticButton } from './SemanticButton'
import { Split } from './Split'

/**
 * Onglet Notes (§2.6).
 *
 * Création, renommage et suppression passent exclusivement par le clic droit :
 * il n'y a plus de boutons [+] [-] [Renommer] en v2.
 *
 * Depuis la v1.2.0, la liste est rangée à la main par glisser-déposer et les
 * notes peuvent être rassemblées en groupes nommés. Un groupe est un cadre
 * coiffé d'un bandeau qui porte son nom ; c'est par ce bandeau qu'on le
 * déplace, qu'on le renomme et qu'on le dissout.
 */

/** Ce qu'on est en train de glisser : une note, ou un groupe pris par son bandeau. */
type Glisse = { kind: 'note' | 'group'; id: string } | null

/** Moitié de l'élément survolé : la haute insère avant, la basse après. */
function moitie(e: DragEvent<HTMLElement>): Position {
  const r = e.currentTarget.getBoundingClientRect()
  return e.clientY < r.top + r.height / 2 ? 'before' : 'after'
}

/**
 * Trait d'insertion, au-dessus ou au-dessous de l'élément survolé. Une ombre
 * plutôt qu'une bordure : la liste ne doit pas bouger pendant qu'on la vise.
 */
function traitInsertion(position: Position | undefined): CSSProperties | undefined {
  if (!position) return undefined
  return { boxShadow: `0 ${position === 'before' ? '-2px' : '2px'} 0 0 #0ea5e9` }
}
export function NotesTab({
  projectId,
  onDataChanged,
  montrerCaches,
  onToggleCaches,
}: {
  projectId: string
  /** Le compteur de notes de la sidebar dépend de ces écritures (§2.1). */
  onDataChanged: () => void
  /** Œil de la liste des notes (§2.6) : montre ou non les notes cachées. */
  montrerCaches: boolean
  onToggleCaches: () => void
}) {
  const [notes, setNotes] = useState<domain.Note[]>([])
  const [groupes, setGroupes] = useState<domain.NoteGroup[]>([])
  // `selectedId` est la note ouverte dans l'éditeur, toujours unique.
  // `selection` est la sélection multiple, qui ne sert qu'à « Regrouper ».
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [selection, setSelection] = useState<Set<string>>(new Set())
  const [glisse, setGlisse] = useState<Glisse>(null)
  const [cible, setCible] = useState<Cible | null>(null)
  const [titre, setTitre] = useState('')
  const [contenu, setContenu] = useState('')
  const [statut, setStatut] = useState<string>('')
  const [menu, setMenu] = useState<MenuState>(null)

  const affichees = montrerCaches ? notes : notes.filter((n) => !n.hidden)
  const selected = affichees.find((n) => n.id === selectedId) ?? null

  // La sélection ne garde que des notes affichées ; vide, elle retombe sur la
  // note ouverte.
  const encoreAffichees = affichees.filter((n) => selection.has(n.id)).map((n) => n.id)
  const selectionnees = new Set(encoreAffichees.length > 0 ? encoreAffichees : selectedId ? [selectedId] : [])

  const recharger = useCallback(async () => {
    const [liste, groupesDuProjet] = await Promise.all([api.GetNotes(projectId), api.GetNoteGroups(projectId)])
    setNotes(liste)
    setGroupes(groupesDuProjet)
    return liste
  }, [projectId])

  useEffect(() => {
    recharger()
  }, [recharger])

  // Si la note sélectionnée sort de la liste affichée — supprimée, cachée, ou
  // œil qui se referme dessus —, la sélection retombe sur la première note
  // affichée (§2.6). Couvre aussi la sélection initiale au premier chargement.
  useEffect(() => {
    setSelectedId((current) => {
      if (current && affichees.some((n) => n.id === current)) return current
      return affichees[0]?.id ?? null
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [affichees])

  useEffect(() => {
    setTitre(selected?.title ?? '')
    setContenu(selected?.content ?? '')
    setStatut('')
  }, [selected?.id])

  const enregistrer = useCallback(
    async ({ title, content }: { title: string; content: string }) => {
      if (!selectedId) return
      const maj = await api.UpdateNote(selectedId, title, content)
      // Seuls le titre, le contenu et la date sont repris : la réponse peut
      // avoir croisé un glisser-déposer, et sa place dans la liste être périmée.
      setNotes((prev) =>
        prev.map((n) =>
          n.id === maj.id ? ({ ...n, title: maj.title, content: maj.content, updatedAt: maj.updatedAt } as domain.Note) : n,
        ),
      )
      setStatut(`Note sauvegardée à ${new Date().toLocaleTimeString('fr-FR')}`)
      onDataChanged()
    },
    [selectedId],
  )

  // 2 s d'inactivité pour les notes (§2.6). Le vidage immédiat est déclenché
  // par le blur des champs et par le changement de note, via useAutosave.
  //
  // La promesse de la dernière écriture est gardée : réordonner ou regrouper
  // attend qu'elle soit finie avant d'écrire la disposition.
  const ecriture = useRef<Promise<void>>(Promise.resolve())
  const save = useAutosave((valeur: { title: string; content: string }) => {
    ecriture.current = enregistrer(valeur)
  }, 2000)

  async function ecrireEnAttente() {
    save.flush()
    await ecriture.current.catch(() => {})
  }

  function modifier(champs: { title?: string; content?: string }) {
    const suivant = { title: champs.title ?? titre, content: champs.content ?? contenu }
    if (champs.title !== undefined) setTitre(champs.title)
    if (champs.content !== undefined) setContenu(champs.content)
    setStatut('Modifications en cours…')
    save.schedule(suivant)
  }

  // Changer de note doit écrire ce qui est en attente avant de basculer,
  // sinon la frappe en cours part avec la note qu'on quitte (§2.6).
  function selectionner(id: string) {
    save.flush()
    setSelectedId(id)
    setSelection(new Set([id]))
  }

  // Ctrl+clic ajoute ou retire une note de la sélection, Maj+clic sélectionne
  // la plage depuis la note ouverte. Ni l'un ni l'autre ne change la note
  // ouverte dans l'éditeur.
  function clic(note: domain.Note, e: MouseEvent) {
    if (e.ctrlKey || e.metaKey) {
      const suivante = new Set(selectionnees)
      if (suivante.has(note.id)) suivante.delete(note.id)
      else suivante.add(note.id)
      setSelection(suivante)
      return
    }
    if (e.shiftKey && selectedId) {
      const a = affichees.findIndex((n) => n.id === selectedId)
      const b = affichees.findIndex((n) => n.id === note.id)
      if (a >= 0 && b >= 0) {
        setSelection(new Set(affichees.slice(Math.min(a, b), Math.max(a, b) + 1).map((n) => n.id)))
        return
      }
    }
    selectionner(note.id)
  }

  async function creer() {
    const titre = window.prompt('Titre de la note ?')
    if (!titre?.trim()) return
    const note = await api.CreateNote(projectId, titre.trim())
    await recharger()
    setSelectedId(note.id)
    setSelection(new Set([note.id]))
    onDataChanged()
  }

  async function supprimer(note: domain.Note) {
    if (!window.confirm(`Supprimer la note « ${note.title} » ?`)) return
    await api.DeleteNote(note.id)
    // La sélection retombe automatiquement sur la première note affichée
    // (effet ci-dessus) : la note supprimée disparaît de `notes`.
    await recharger()
    onDataChanged()
  }

  async function basculerCache(note: domain.Note) {
    await api.SetNoteHidden(note.id, !note.hidden)
    await recharger()
    onDataChanged()
  }

  // La couleur est un attribut d'affichage : seule la couleur revient dans
  // l'état, pour ne rien écraser de ce qui a pu changer entre-temps.
  async function changerCouleur(couleur: string) {
    if (!selectedId) return
    await ecrireEnAttente()
    const maj = await api.SetNoteColor(selectedId, couleur)
    setNotes((prev) => prev.map((n) => (n.id === maj.id ? ({ ...n, color: maj.color } as domain.Note) : n)))
  }

  async function regrouper(ids: string[]) {
    const nom = window.prompt('Nom du groupe ?')
    if (!nom?.trim()) return
    await ecrireEnAttente()
    await api.GroupNotes(projectId, nom.trim(), ids)
    await recharger()
  }

  async function renommerGroupe(groupe: domain.NoteGroup) {
    const nom = window.prompt('Nouveau nom ?', groupe.name)
    if (!nom?.trim()) return
    await api.RenameNoteGroup(groupe.id, nom.trim())
    await recharger()
  }

  // Les notes restent en place et redeviennent des notes hors groupe : rien
  // n'est perdu, donc pas de confirmation.
  async function dissoudreGroupe(groupe: domain.NoteGroup) {
    await ecrireEnAttente()
    await api.DissolveNoteGroup(groupe.id)
    await recharger()
  }

  function menuNote(e: MouseEvent, note: domain.Note) {
    e.preventDefault()
    e.stopPropagation()
    // Clic droit hors de la sélection : la note visée devient la seule
    // sélectionnée, et « Regrouper » ne porte que sur elle.
    const dansLaSelection = selectionnees.has(note.id)
    const aRegrouper = dansLaSelection ? affichees.filter((n) => selectionnees.has(n.id)).map((n) => n.id) : [note.id]
    if (!dansLaSelection) setSelection(new Set([note.id]))
    setMenu({
      x: e.clientX,
      y: e.clientY,
      items: [
        { kind: 'action', label: '+ Nouvelle note', onSelect: creer },
        { kind: 'separator' },
        { kind: 'action', label: 'Regrouper', onSelect: () => regrouper(aRegrouper) },
        {
          kind: 'action',
          label: note.hidden ? 'Réafficher' : 'Cacher',
          onSelect: () => basculerCache(note),
        },
        { kind: 'action', label: 'Supprimer', danger: true, onSelect: () => supprimer(note) },
      ],
    })
  }

  function menuGroupe(e: MouseEvent, groupe: domain.NoteGroup) {
    e.preventDefault()
    e.stopPropagation()
    setMenu({
      x: e.clientX,
      y: e.clientY,
      items: [
        { kind: 'action', label: 'Renommer le groupe', onSelect: () => renommerGroupe(groupe) },
        { kind: 'action', label: 'Dissoudre le groupe', onSelect: () => dissoudreGroupe(groupe) },
      ],
    })
  }

  /* ---- Glisser-déposer ---- */

  function commencer(e: DragEvent<HTMLElement>, quoi: NonNullable<Glisse>) {
    e.dataTransfer.setData(quoi.kind === 'note' ? 'text/note-id' : 'text/note-group-id', quoi.id)
    e.dataTransfer.effectAllowed = 'move'
    setGlisse(quoi)
  }

  function terminer() {
    setGlisse(null)
    setCible(null)
  }

  // Sur une note : la note glissée se pose avant ou après elle. Un groupe
  // glissé au-dessus d'une note d'un autre groupe ne vise pas cette note — rien
  // n'est rendu, et l'événement remonte au cadre, qui vise le groupe entier.
  function cibleSurNote(e: DragEvent<HTMLElement>, note: domain.Note): Cible | null {
    if (glisse?.kind === 'group' && note.groupId) return null
    return { type: 'note', noteId: note.id, position: moitie(e) }
  }

  // Sur le cadre d'un groupe, bandeau compris : une note y entre, un autre
  // groupe se pose avant ou après lui, jamais dedans.
  function cibleSurGroupe(e: DragEvent<HTMLElement>, groupId: string): Cible {
    if (glisse?.kind === 'group') return { type: 'group', groupId, position: moitie(e) }
    return { type: 'group', groupId }
  }

  function survoler(e: DragEvent<HTMLElement>, visee: Cible | null) {
    if (!glisse || !visee) return
    e.preventDefault()
    e.stopPropagation()
    setCible((precedente) => (JSON.stringify(precedente) === JSON.stringify(visee) ? precedente : visee))
  }

  async function deposer(e: DragEvent<HTMLElement>, visee: Cible | null) {
    if (!glisse || !visee) return
    e.preventDefault()
    e.stopPropagation()
    const quoi = glisse
    terminer()

    const actuelle = toLayout(notes)
    const suivante = quoi.kind === 'note' ? moveNote(actuelle, quoi.id, visee) : moveGroup(actuelle, quoi.id, visee)
    if (memeDisposition(actuelle, suivante)) return

    // La frappe en attente part d'abord : elle ne doit pas être perdue, ni
    // revenir écraser la disposition qu'on s'apprête à écrire.
    await ecrireEnAttente()
    try {
      await api.SetNotesLayout(projectId, suivante)
    } finally {
      await recharger()
    }
  }

  function ligne(note: domain.Note) {
    const ouverte = note.id === selectedId
    const selectionnee = selectionnees.has(note.id)
    // Une note colorée garde sa couleur quel que soit son état : la note
    // ouverte et la sélection se lisent donc sur un liseré, pas sur le fond.
    const fond = teinte(note.color)
    const insertion = cible?.type === 'note' && cible.noteId === note.id && glisse?.id !== note.id ? cible.position : undefined
    return (
      <li key={note.id} data-ligne data-insertion={insertion} style={traitInsertion(insertion)}>
        <button
          type="button"
          draggable
          aria-current={ouverte ? 'true' : undefined}
          aria-pressed={selectionnee}
          onClick={(e) => clic(note, e)}
          onContextMenu={(e) => menuNote(e, note)}
          onDragStart={(e) => commencer(e, { kind: 'note', id: note.id })}
          onDragEnd={terminer}
          onDragOver={(e) => survoler(e, cibleSurNote(e, note))}
          onDrop={(e) => deposer(e, cibleSurNote(e, note))}
          style={{ backgroundColor: fond }}
          className={`w-full truncate rounded px-2 py-1 text-left text-sm ${
            fond ? '' : ouverte ? 'bg-neutral-200' : selectionnee ? 'bg-sky-100' : 'hover:bg-neutral-100'
          } ${
            ouverte
              ? 'ring-2 ring-inset ring-[var(--color-selection)]'
              : selectionnee
                ? 'ring-2 ring-inset ring-sky-400'
                : ''
          } ${note.hidden ? 'italic opacity-50' : ''}`}
        >
          {note.title || 'Sans titre'}
        </button>
      </li>
    )
  }

  function cadre(groupId: string, notesDuGroupe: domain.Note[]) {
    const groupe = groupes.find((g) => g.id === groupId)
    // Le groupe arrive par un second appel : le temps qu'il soit là, ses notes
    // s'affichent déjà, dans un cadre encore sans nom.
    const nom = groupe?.name ?? ''
    const visee = cible?.type === 'group' && cible.groupId === groupId && glisse?.id !== groupId ? cible : null
    const accueille = !!visee && !visee.position
    const idBandeau = `groupe-de-notes-${groupId}`
    return (
      <li key={groupId} data-ligne className="my-1">
        <div
          role="group"
          aria-labelledby={idBandeau}
          data-insertion={visee?.position}
          data-accueille={accueille || undefined}
          style={traitInsertion(visee?.position)}
          onDragOver={(e) => survoler(e, cibleSurGroupe(e, groupId))}
          onDrop={(e) => deposer(e, cibleSurGroupe(e, groupId))}
          className={`overflow-hidden rounded border-2 ${accueille ? 'border-sky-500' : 'border-neutral-400'}`}
        >
          <div
            id={idBandeau}
            draggable
            title={nom}
            onDragStart={(e) => commencer(e, { kind: 'group', id: groupId })}
            onDragEnd={terminer}
            onContextMenu={(e) => groupe && menuGroupe(e, groupe)}
            className={`cursor-grab truncate px-2 py-1 text-xs font-semibold text-white ${
              accueille ? 'bg-sky-500' : 'bg-neutral-400'
            }`}
          >
            {nom}
          </div>
          <ul aria-label={`Notes du groupe ${nom}`} className="p-0.5">
            {notesDuGroupe.map(ligne)}
          </ul>
        </div>
      </li>
    )
  }

  return (
    <>
      <Split
        initial={260}
        min={160}
        max={520}
        className="h-full"
        first={
          <div className="flex h-full flex-col">
            <div className="flex items-center justify-end border-b border-neutral-200 px-2 py-1">
              <HiddenToggle montrerCaches={montrerCaches} onToggle={onToggleCaches} />
            </div>
            <ul
              aria-label="Notes"
              className="min-h-0 flex-1 overflow-auto p-1"
              onContextMenu={(e) => {
                e.preventDefault()
                setMenu({ x: e.clientX, y: e.clientY, items: [{ kind: 'action', label: '+ Nouvelle note', onSelect: creer }] })
              }}
              // Double-clic sur le fond : création directe (§2.11). Le filtre
              // écarte les double-clics tombés sur une note.
              onDoubleClick={(e) => {
                if (surLeFond(e)) creer()
              }}
              // Déposer sur le fond : en fin de liste, hors de tout groupe.
              onDragOver={(e) => survoler(e, { type: 'end' })}
              onDrop={(e) => deposer(e, { type: 'end' })}
              onDragLeave={(e) => {
                if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setCible(null)
              }}
            >
              {blocs(affichees, true).map((bloc) =>
                bloc.type === 'note' ? ligne(bloc.note) : cadre(bloc.groupId, bloc.notes),
              )}
              {cible?.type === 'end' ? <li aria-hidden data-insertion="end" className="h-0.5 rounded bg-sky-500" /> : null}
              {affichees.length === 0 ? (
                <li className="p-2 text-xs text-neutral-500">
                  Aucune note. Double-clic ou clic droit pour en créer une.
                </li>
              ) : null}
            </ul>
          </div>
        }
        second={
          selected ? (
            <div className="flex h-full flex-col">
              <input
                value={titre}
                aria-label="Titre de la note"
                onChange={(e) => modifier({ title: e.target.value })}
                onBlur={save.flush}
                className="border-b border-neutral-300 px-3 py-2 text-sm font-medium focus:outline-none"
              />
              <div className="min-h-0 flex-1">
                <RichEditor
                  content={contenu}
                  onChange={(html) => modifier({ content: html })}
                  onBlur={save.flush}
                  color={selected.color}
                  onColorChange={changerCouleur}
                />
              </div>
              <div className="flex items-center gap-3 border-t border-neutral-300 px-3 py-1">
                <p aria-live="polite" className="text-xs text-neutral-500">
                  {statut}
                </p>
                {/* Opt-in sémantique (§2.12). Remonté par sa clé : changer de
                    note doit relire l'état, pas le garder de la précédente. */}
                <span className="ml-auto">
                  <SemanticButton key={selected.id} entityType="note" entityId={selected.id} />
                </span>
              </div>
            </div>
          ) : (
            <p className="p-4 text-sm text-neutral-500">Aucune note sélectionnée.</p>
          )
        }
      />
      <ContextMenu state={menu} onClose={() => setMenu(null)} />
    </>
  )
}
