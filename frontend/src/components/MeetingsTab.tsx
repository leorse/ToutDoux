import { useCallback, useEffect, useState } from 'react'
import { api } from '../api'
import type { domain } from '../../wailsjs/go/models'
import { useAutosave } from '../useAutosave'
import { ContextMenu, type MenuState } from './ContextMenu'
import { RichEditor } from './RichEditor'
import { SemanticButton } from './SemanticButton'
import { Split } from './Split'

/**
 * Onglet Réunions (§2.7).
 *
 * Trois colonnes : réunions, instances, éditeur. Tout passe par le clic droit ;
 * il n'y a plus de boutons [+] [-] en v2.
 */
export function MeetingsTab({
  projectId,
  onDataChanged,
  cibleInstance,
}: {
  projectId: string
  onDataChanged: () => void
  /** Instance à ouvrir, quand on arrive depuis la recherche (§2.9). */
  cibleInstance?: { meetingId?: string; instanceId?: string }
}) {
  const [meetings, setMeetings] = useState<domain.Meeting[]>([])
  const [instances, setInstances] = useState<domain.MeetingInstance[]>([])
  const [meetingId, setMeetingId] = useState<string | null>(null)
  const [instanceId, setInstanceId] = useState<string | null>(null)
  const [notes, setNotes] = useState('')
  const [statut, setStatut] = useState('')
  const [menu, setMenu] = useState<MenuState>(null)

  const instance = instances.find((i) => i.id === instanceId) ?? null

  const chargerReunions = useCallback(async () => {
    const liste = await api.GetMeetings(projectId)
    setMeetings(liste)
    return liste
  }, [projectId])

  useEffect(() => {
    chargerReunions().then((liste) => {
      setMeetingId(cibleInstance?.meetingId ?? liste[0]?.id ?? null)
    })
  }, [chargerReunions, cibleInstance?.meetingId])

  useEffect(() => {
    if (!meetingId) {
      setInstances([])
      setInstanceId(null)
      return
    }
    api.GetInstances(meetingId).then((liste) => {
      setInstances(liste)
      // Sélectionner une réunion sélectionne sa dernière instance, pour éviter
      // un clic supplémentaire (§2.7).
      setInstanceId(cibleInstance?.instanceId ?? liste[0]?.id ?? null)
    })
  }, [meetingId, cibleInstance?.instanceId])

  useEffect(() => {
    setNotes(instance?.notes ?? '')
    setStatut('')
  }, [instance?.id])

  const enregistrer = useCallback(
    async (valeur: string) => {
      if (!instanceId) return
      const maj = await api.UpdateInstanceNotes(instanceId, valeur)
      setInstances((prev) => prev.map((i) => (i.id === maj.id ? maj : i)))
      setStatut(`Sauvegardé à ${new Date().toLocaleTimeString('fr-FR')}`)
      onDataChanged()
    },
    [instanceId, onDataChanged],
  )

  // 2 s d'inactivité, plus vidage immédiat au blur et au changement
  // d'instance, de réunion ou de projet (§2.7, même fiabilité qu'en §2.6).
  const save = useAutosave(enregistrer, 2000)

  function choisirReunion(id: string) {
    save.flush()
    setMeetingId(id)
  }

  function choisirInstance(id: string) {
    save.flush()
    setInstanceId(id)
  }

  async function creerReunion() {
    const titre = window.prompt('Titre de la réunion ?')
    if (!titre?.trim()) return
    const m = await api.CreateMeeting(projectId, titre.trim())
    await chargerReunions()
    setMeetingId(m.id)
    onDataChanged()
  }

  async function renommerReunion(m: domain.Meeting) {
    const titre = window.prompt('Nouveau titre ?', m.title)
    if (!titre?.trim() || titre === m.title) return
    await api.RenameMeeting(m.id, titre.trim())
    await chargerReunions()
  }

  async function supprimerReunion(m: domain.Meeting) {
    if (!window.confirm(`Supprimer « ${m.title} » et tout son historique ?`)) return
    await api.DeleteMeeting(m.id)
    const liste = await chargerReunions()
    if (meetingId === m.id) setMeetingId(liste[0]?.id ?? null)
    onDataChanged()
  }

  async function creerInstance() {
    if (!meetingId) return
    const i = await api.AddMeetingInstance(meetingId)
    const liste = await api.GetInstances(meetingId)
    setInstances(liste)
    setInstanceId(i.id)
  }

  async function supprimerInstance(i: domain.MeetingInstance) {
    if (!window.confirm('Supprimer cette instance ?')) return
    await api.DeleteInstance(i.id)
    const liste = await api.GetInstances(meetingId!)
    setInstances(liste)
    if (instanceId === i.id) setInstanceId(liste[0]?.id ?? null)
  }

  return (
    <>
      <Split
        initial={220}
        min={140}
        max={420}
        className="h-full"
        first={
          <ul
            aria-label="Réunions"
            className="h-full min-h-full p-1"
            onContextMenu={(e) => {
              e.preventDefault()
              setMenu({
                x: e.clientX,
                y: e.clientY,
                items: [{ kind: 'action', label: '+ Nouvelle réunion', onSelect: creerReunion }],
              })
            }}
          >
            {meetings.map((m) => (
              <li key={m.id}>
                <button
                  type="button"
                  onClick={() => choisirReunion(m.id)}
                  onContextMenu={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    choisirReunion(m.id)
                    setMenu({
                      x: e.clientX,
                      y: e.clientY,
                      items: [
                        { kind: 'action', label: '+ Nouvelle réunion', onSelect: creerReunion },
                        { kind: 'separator' },
                        { kind: 'action', label: 'Renommer', onSelect: () => renommerReunion(m) },
                        { kind: 'action', label: 'Supprimer', danger: true, onSelect: () => supprimerReunion(m) },
                      ],
                    })
                  }}
                  className={`w-full truncate rounded px-2 py-1 text-left text-sm ${
                    m.id === meetingId ? 'bg-neutral-200' : 'hover:bg-neutral-100'
                  }`}
                >
                  {m.title}
                </button>
              </li>
            ))}
            {meetings.length === 0 ? (
              <li className="p-2 text-xs text-neutral-500">Aucune réunion. Clic droit pour en créer une.</li>
            ) : null}
          </ul>
        }
        second={
          <Split
            initial={220}
            min={140}
            max={420}
            className="h-full"
            first={
              <ul
                aria-label="Instances"
                className="h-full min-h-full p-1"
                onContextMenu={(e) => {
                  e.preventDefault()
                  // Sans réunion sélectionnée, le clic droit ne fait rien :
                  // pas de menu du tout, précision de la v2 (§2.7).
                  if (!meetingId) return
                  setMenu({
                    x: e.clientX,
                    y: e.clientY,
                    items: [{ kind: 'action', label: '+ Nouvelle instance', onSelect: creerInstance }],
                  })
                }}
              >
                {instances.map((i) => (
                  <li key={i.id}>
                    <button
                      type="button"
                      onClick={() => choisirInstance(i.id)}
                      onContextMenu={(e) => {
                        e.preventDefault()
                        e.stopPropagation()
                        choisirInstance(i.id)
                        setMenu({
                          x: e.clientX,
                          y: e.clientY,
                          items: [
                            { kind: 'action', label: '+ Nouvelle instance', onSelect: creerInstance },
                            { kind: 'separator' },
                            { kind: 'action', label: 'Supprimer', danger: true, onSelect: () => supprimerInstance(i) },
                          ],
                        })
                      }}
                      className={`w-full truncate rounded px-2 py-1 text-left text-sm ${
                        i.id === instanceId ? 'bg-neutral-200' : 'hover:bg-neutral-100'
                      }`}
                    >
                      {new Date(i.timestamp).toLocaleString('fr-FR', {
                        day: '2-digit',
                        month: '2-digit',
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
                    </button>
                  </li>
                ))}
                {meetingId && instances.length === 0 ? (
                  <li className="p-2 text-xs text-neutral-500">Aucune instance. Clic droit pour en créer une.</li>
                ) : null}
              </ul>
            }
            second={
              instance ? (
                <div className="flex h-full flex-col">
                  <div className="min-h-0 flex-1">
                    <RichEditor
                      content={notes}
                      onChange={(html) => {
                        setNotes(html)
                        setStatut('Modifications en cours…')
                        save.schedule(html)
                      }}
                      onBlur={save.flush}
                    />
                  </div>
                  <div className="flex items-center gap-3 border-t border-neutral-300 px-3 py-1">
                    <p aria-live="polite" className="text-xs text-neutral-500">
                      {statut}
                    </p>
                    {/* Opt-in sémantique sur l'instance, pas sur la réunion :
                        c'est l'instance qui porte le compte rendu (§2.12). */}
                    <span className="ml-auto">
                      <SemanticButton key={instance.id} entityType="meeting" entityId={instance.id} />
                    </span>
                  </div>
                </div>
              ) : (
                <p className="p-4 text-sm text-neutral-500">Aucune instance sélectionnée.</p>
              )
            }
          />
        }
      />
      <ContextMenu state={menu} onClose={() => setMenu(null)} />
    </>
  )
}
