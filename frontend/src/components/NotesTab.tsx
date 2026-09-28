import { useCallback, useEffect, useState } from 'react'
import { api } from '../api'
import { surLeFond } from '../fond'
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
 */
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
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [titre, setTitre] = useState('')
  const [contenu, setContenu] = useState('')
  const [statut, setStatut] = useState<string>('')
  const [menu, setMenu] = useState<MenuState>(null)

  const affichees = montrerCaches ? notes : notes.filter((n) => !n.hidden)
  const selected = affichees.find((n) => n.id === selectedId) ?? null

  const recharger = useCallback(async () => {
    const liste = await api.GetNotes(projectId)
    setNotes(liste)
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
      setNotes((prev) => prev.map((n) => (n.id === maj.id ? maj : n)))
      setStatut(`Note sauvegardée à ${new Date().toLocaleTimeString('fr-FR')}`)
      onDataChanged()
    },
    [selectedId],
  )

  // 2 s d'inactivité pour les notes (§2.6). Le vidage immédiat est déclenché
  // par le blur des champs et par le changement de note, via useAutosave.
  const save = useAutosave(enregistrer, 2000)

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
  }

  async function creer() {
    const titre = window.prompt('Titre de la note ?')
    if (!titre?.trim()) return
    const note = await api.CreateNote(projectId, titre.trim())
    await recharger()
    setSelectedId(note.id)
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
            >
              {affichees.map((note) => (
                <li key={note.id} data-ligne>
                  <button
                    type="button"
                    onClick={() => selectionner(note.id)}
                    onContextMenu={(e) => {
                      e.preventDefault()
                      e.stopPropagation()
                      setMenu({
                        x: e.clientX,
                        y: e.clientY,
                        items: [
                          { kind: 'action', label: '+ Nouvelle note', onSelect: creer },
                          { kind: 'separator' },
                          {
                            kind: 'action',
                            label: note.hidden ? 'Réafficher' : 'Cacher',
                            onSelect: () => basculerCache(note),
                          },
                          { kind: 'action', label: 'Supprimer', danger: true, onSelect: () => supprimer(note) },
                        ],
                      })
                    }}
                    className={`w-full truncate rounded px-2 py-1 text-left text-sm ${
                      note.id === selectedId ? 'bg-neutral-200' : 'hover:bg-neutral-100'
                    } ${note.hidden ? 'italic opacity-50' : ''}`}
                  >
                    {note.title || 'Sans titre'}
                  </button>
                </li>
              ))}
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
