import { useCallback, useEffect, useState } from 'react'
import { api } from '../api'
import type { domain } from '../../wailsjs/go/models'
import { Split } from './Split'
import { LigneTache, PastilleImportance } from './TaskRow'

/** Rythme de rafraîchissement des échéances (§2.3). */
const REFRESH_MS = 30_000

/**
 * Vue Priorités (§2.8).
 *
 * Changement majeur de la v2 : ce n'est plus un onglet sous un projet mais une
 * vue de premier niveau, pleine largeur et sans sidebar — elle balaie tous les
 * projets par nature, la rattacher à un seul n'aurait pas de sens.
 */
export function PrioritiesView({
  projects,
  onOpenTask,
}: {
  projects: domain.Project[]
  /** Double-clic : bascule sur la vue Projets et ouvre la tâche (§2.8). */
  onOpenTask: (task: domain.Task) => void
}) {
  const [critiques, setCritiques] = useState<domain.Task[]>([])
  const [echeances, setEcheances] = useState<domain.Task[]>([])
  const [selected, setSelected] = useState<domain.Task | null>(null)
  const [erreur, setErreur] = useState<string | null>(null)

  const nomProjet = useCallback(
    (id: string) => projects.find((p) => p.id === id)?.name ?? '',
    [projects],
  )

  const charger = useCallback(async () => {
    try {
      const p = await api.GetPriorityTasks()
      setCritiques(p.criticalOrHigh ?? [])
      setEcheances(p.dueSoon ?? [])
      setErreur(null)
    } catch (err) {
      setErreur(String(err))
    }
  }, [])

  useEffect(() => {
    charger()
    const id = setInterval(charger, REFRESH_MS)
    return () => clearInterval(id)
  }, [charger])

  return (
    <div className="flex h-full flex-col">
      {erreur ? (
        <p role="alert" className="border-b border-red-300 bg-red-50 px-3 py-1.5 text-xs text-red-800">
          {erreur}
        </p>
      ) : null}

      <Split
        initial={640}
        min={320}
        max={1100}
        className="h-full"
        first={
          <div className="flex flex-col gap-4 p-2">
            <Section
              titre={`Tâches Critiques & Hautes (${critiques.length})`}
              tasks={critiques}
              nomProjet={nomProjet}
              selectedId={selected?.id ?? null}
              onSelect={setSelected}
              onOpen={onOpenTask}
            />
            <Section
              titre={`Échéances à venir (${echeances.length})`}
              tasks={echeances}
              nomProjet={nomProjet}
              selectedId={selected?.id ?? null}
              onSelect={setSelected}
              onOpen={onOpenTask}
            />
          </div>
        }
        second={<DetailLectureSeule task={selected} nomProjet={nomProjet} onOpen={onOpenTask} />}
      />
    </div>
  )
}

function Section({
  titre,
  tasks,
  nomProjet,
  selectedId,
  onSelect,
  onOpen,
}: {
  titre: string
  tasks: domain.Task[]
  nomProjet: (id: string) => string
  selectedId: string | null
  onSelect: (t: domain.Task) => void
  onOpen: (t: domain.Task) => void
}) {
  return (
    <section>
      <h2 className="mb-1 px-1 text-sm font-semibold text-neutral-700">{titre}</h2>
      {tasks.length === 0 ? (
        <p className="px-1 text-xs text-neutral-500">Aucune tâche.</p>
      ) : (
        <ul aria-label={titre} className="flex flex-col">
          {tasks.map((task) => (
            <LigneTache
              key={task.id}
              task={task}
              projectName={nomProjet(task.projectId)}
              selected={task.id === selectedId}
              onSelect={() => onSelect(task)}
              onOpen={() => onOpen(task)}
            />
          ))}
        </ul>
      )}
    </section>
  )
}

/**
 * Aperçu en lecture seule (§2.8).
 *
 * Même structure visuelle que le panneau de détail de l'onglet Tâches, mais
 * aucun champ éditable : pour modifier, on ouvre la tâche dans son projet.
 */
function DetailLectureSeule({
  task,
  nomProjet,
  onOpen,
}: {
  task: domain.Task | null
  nomProjet: (id: string) => string
  onOpen: (t: domain.Task) => void
}) {
  if (!task) {
    return <p className="p-4 text-sm text-neutral-500">Sélectionne une tâche pour en voir le détail.</p>
  }
  return (
    <div className="flex h-full flex-col gap-3 p-3">
      <div className="flex items-center gap-2">
        <PastilleImportance importance={task.importance} />
        <h2 className="truncate text-sm font-semibold">{task.name}</h2>
      </div>

      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
        <dt className="text-neutral-500">Projet</dt>
        <dd>{nomProjet(task.projectId)}</dd>
        <dt className="text-neutral-500">Importance</dt>
        <dd>{task.importance}</dd>
        <dt className="text-neutral-500">Statut</dt>
        <dd>{task.cancelled ? 'Annulée' : task.completed ? 'Terminée' : 'Active'}</dd>
        <dt className="text-neutral-500">Échéance</dt>
        <dd>{task.dueDate ? new Date(task.dueDate).toLocaleString('fr-FR') : '—'}</dd>
      </dl>

      <div className="min-h-0 flex-1 overflow-auto rounded border border-neutral-200 bg-neutral-50 p-2 text-sm">
        {task.description || <span className="text-neutral-400">Aucune description.</span>}
      </div>

      <button
        type="button"
        onClick={() => onOpen(task)}
        className="self-start rounded bg-[var(--color-selection)] px-3 py-1 text-sm text-white"
      >
        Ouvrir dans Tâches
      </button>
    </div>
  )
}
