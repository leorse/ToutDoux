import { useCallback, useEffect, useState } from 'react'
import { api } from '../api'
import type { domain } from '../../wailsjs/go/models'
import { useAutosave } from '../useAutosave'
import { ContextMenu, type MenuState } from './ContextMenu'
import { RichEditor } from './RichEditor'
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
}: {
  projectId: string
  /** Le compteur de notes de la sidebar dépend de ces écritures (§2.1). */
  onDataChanged: () => void
}) {
  const [notes, setNotes] = useState<domain.Note[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [titre, setTitre] = useState('')
  const [contenu, setContenu] = useState('')
  const [statut, setStatut] = useState<string>('')
  const [menu, setMenu] = useState<MenuState>(null)

  const selected = notes.find((n) => n.id === selectedId) ?? null

  const recharger = useCallback(async () => {
    const liste = await api.GetNotes(projectId)
    setNotes(liste)
    return liste
  }, [projectId])

  useEffect(() => {
    recharger().then((liste) => setSelectedId(liste[0]?.id ?? null))
  }, [recharger])

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
    const liste = await recharger()
    if (selectedId === note.id) setSelectedId(liste[0]?.id ?? null)
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
          <ul
            aria-label="Notes"
            className="h-full min-h-full p-1"
            onContextMenu={(e) => {
              e.preventDefault()
              setMenu({ x: e.clientX, y: e.clientY, items: [{ kind: 'action', label: '+ Nouvelle note', onSelect: creer }] })
            }}
          >
            {notes.map((note) => (
              <li key={note.id}>
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
                        { kind: 'action', label: 'Supprimer', danger: true, onSelect: () => supprimer(note) },
                      ],
                    })
                  }}
                  className={`w-full truncate rounded px-2 py-1 text-left text-sm ${
                    note.id === selectedId ? 'bg-neutral-200' : 'hover:bg-neutral-100'
                  }`}
                >
                  {note.title || 'Sans titre'}
                </button>
              </li>
            ))}
            {notes.length === 0 ? (
              <li className="p-2 text-xs text-neutral-500">
                Aucune note. Clic droit pour en créer une.
              </li>
            ) : null}
          </ul>
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
              <p aria-live="polite" className="border-t border-neutral-300 px-3 py-1 text-xs text-neutral-500">
                {statut}
              </p>
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
