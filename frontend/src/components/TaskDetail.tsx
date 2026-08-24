import { useEffect, useState } from 'react'
import type { domain, duedate } from '../../wailsjs/go/models'
import { useAutosave } from '../useAutosave'

const IMPORTANCES = ['Basse', 'Normale', 'Haute', 'Critique'] as const
type Importance = (typeof IMPORTANCES)[number]

const COULEUR: Record<Importance, string> = {
  Critique: 'var(--color-critique)',
  Haute: 'var(--color-haute)',
  Normale: 'var(--color-normale)',
  Basse: 'var(--color-basse)',
}

const RACCOURCIS: { kind: string; label: string }[] = [
  { kind: 'now', label: 'Tout de suite' },
  { kind: '10min', label: 'Dans 10 min' },
  { kind: '1h', label: 'Dans 1h' },
  { kind: 'journee', label: 'Journée' },
  { kind: 'lendemain9h', label: 'Lendemain 9h' },
]

export type TaskDetailProps = {
  task: domain.Task | null
  due: duedate.Info | undefined
  readOnly?: boolean
  onPatch: (patch: { name?: string; description?: string; importance?: string; clearDue?: boolean }) => void
  onQuickDue: (kind: string) => void
  onSetDue: (isoLocal: string) => void
}

/**
 * Panneau de détail d'une tâche (§2.2, §2.3).
 *
 * Il n'y a plus de boutons de déplacement ici : « Monter », « Descendre » et
 * consorts sont passés au menu contextuel en v2 (§2.2).
 */
export function TaskDetail({ task, due, readOnly, onPatch, onQuickDue, onSetDue }: TaskDetailProps) {
  const [nom, setNom] = useState('')
  const [description, setDescription] = useState('')

  useEffect(() => {
    setNom(task?.name ?? '')
    setDescription(task?.description ?? '')
  }, [task?.id, task?.name, task?.description])

  // Sauvegarde à 500 ms pour les tâches (§2.6), avec flush immédiat au blur.
  const nomSave = useAutosave((valeur: string) => onPatch({ name: valeur }), 500)
  const descSave = useAutosave((valeur: string) => onPatch({ description: valeur }), 500)

  if (!task) {
    return (
      <p className="p-4 text-sm text-neutral-500">
        Aucune tâche sélectionnée. Clic droit sur le fond de l’arbre pour en créer une.
      </p>
    )
  }

  // Après 17h, « Journée » (18h) n'a plus de sens : le bouton est désactivé et
  // renvoie vers « Lendemain 9h » (§2.3). C'est une règle d'affichage — le
  // domaine, lui, rend 18h sans condition.
  const apres17h = new Date().getHours() >= 17

  return (
    <div className="flex h-full flex-col gap-3 overflow-auto p-3">
      <label className="flex flex-col gap-1">
        <span className="text-xs font-medium text-neutral-600">Nom</span>
        <input
          value={nom}
          readOnly={readOnly}
          onChange={(e) => {
            setNom(e.target.value)
            nomSave.schedule(e.target.value)
          }}
          onBlur={nomSave.flush}
          className="rounded border border-neutral-300 px-2 py-1 text-sm read-only:bg-neutral-50"
        />
      </label>

      <fieldset className="flex flex-col gap-1">
        <legend className="text-xs font-medium text-neutral-600">Importance</legend>
        <div className="flex gap-1">
          {IMPORTANCES.map((imp) => (
            <button
              key={imp}
              type="button"
              disabled={readOnly}
              aria-pressed={task.importance === imp}
              onClick={() => onPatch({ importance: imp })}
              style={{ backgroundColor: COULEUR[imp] }}
              className={`rounded px-2 py-1 text-xs ${
                task.importance === imp ? 'border-2 border-neutral-900' : 'border border-neutral-300'
              }`}
            >
              {imp}
            </button>
          ))}
        </div>
      </fieldset>

      <fieldset className="flex flex-col gap-1">
        <legend className="text-xs font-medium text-neutral-600">Échéance</legend>

        <div className="flex flex-wrap items-center gap-1">
          {RACCOURCIS.map((r) => (
            <button
              key={r.kind}
              type="button"
              disabled={readOnly || (r.kind === 'journee' && apres17h)}
              title={r.kind === 'journee' && apres17h ? 'Après 17h, préférer « Lendemain 9h »' : undefined}
              onClick={() => onQuickDue(r.kind)}
              className="rounded border border-neutral-300 px-2 py-1 text-xs hover:bg-neutral-100 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {r.label}
            </button>
          ))}
        </div>

        <div className="mt-1 flex items-center gap-2">
          {/* Sélecteur libre, pour les échéances que les raccourcis ne couvrent
              pas (§2.3, nouveauté v2). */}
          <input
            type="datetime-local"
            disabled={readOnly}
            value={task.dueDate ? pourInputLocal(task.dueDate) : ''}
            onChange={(e) => e.target.value && onSetDue(e.target.value)}
            aria-label="Échéance précise"
            className="rounded border border-neutral-300 px-2 py-1 text-xs"
          />

          {due ? (
            <>
              <span className="text-xs text-neutral-600">
                <span aria-hidden>{due.urgent || due.overdue ? '⏰' : '🕐'}</span> {due.text}
              </span>
              {/* Croix rouge et non un lien « Retirer » (§2.3). */}
              <button
                type="button"
                disabled={readOnly}
                onClick={() => onPatch({ clearDue: true })}
                aria-label="Retirer l’échéance"
                className="flex h-5 w-5 items-center justify-center rounded-full bg-red-600 text-xs leading-none text-white hover:bg-red-700 disabled:opacity-40"
              >
                ✕
              </button>
            </>
          ) : null}
        </div>
      </fieldset>

      <label className="flex min-h-0 flex-1 flex-col gap-1">
        <span className="text-xs font-medium text-neutral-600">Description</span>
        <textarea
          value={description}
          readOnly={readOnly}
          onChange={(e) => {
            setDescription(e.target.value)
            descSave.schedule(e.target.value)
          }}
          onBlur={descSave.flush}
          className="min-h-24 flex-1 resize-none rounded border border-neutral-300 px-2 py-1 text-sm read-only:bg-neutral-50"
        />
      </label>
    </div>
  )
}

/**
 * Convertit une date ISO en valeur acceptée par `datetime-local`, qui attend
 * l'heure **locale** sans fuseau — lui passer un ISO en UTC décalerait
 * l'affichage de plusieurs heures.
 */
function pourInputLocal(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`
}
