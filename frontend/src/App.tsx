import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import type { domain } from '../wailsjs/go/models'
import { MeetingsTab } from './components/MeetingsTab'
import { NotesTab } from './components/NotesTab'
import { PrioritiesView } from './components/PrioritiesView'
import { ProjectSidebar } from './components/ProjectSidebar'
import { SearchView, type CibleOuverture } from './components/SearchView'
import { Split } from './components/Split'
import { FILTRES_PAR_DEFAUT, TasksTab, type Filtres } from './components/TasksTab'

/** Vues de premier niveau (§2.11). */
type View = 'projects' | 'priorities'

/** Sous-onglets de la vue Projets (§2.11). */
type Tab = 'tasks' | 'notes' | 'meetings'

/**
 * Coquille de l'application (§2.11).
 *
 * Il n'y a volontairement pas de barre de titre « Tout Doux » en HTML : la
 * fenêtre Wails en fournit une, native. Celle du prototype n'existait que
 * parce qu'il tournait dans un onglet de navigateur.
 */
export default function App() {
  const [view, setView] = useState<View>('projects')
  const [tab, setTab] = useState<Tab>('tasks')
  const [query, setQuery] = useState('')
  const [projects, setProjects] = useState<domain.Project[]>([])
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Les filtres persistent d'un projet à l'autre : ils ne sont plus
  // réinitialisés au changement de projet, contrairement à la v1 (§2.4).
  const [filtres, setFiltres] = useState<Filtres>(FILTRES_PAR_DEFAUT)

  // Élément à ouvrir après une navigation depuis Priorités ou la Recherche.
  const [cible, setCible] = useState<CibleOuverture | null>(null)

  // Compteur de révision des données. Chaque écriture l'incrémente, et les vues
  // qui affichent des agrégats s'y abonnent — c'est ce qui met à jour les
  // compteurs et les couleurs de la sidebar au moment de l'action.
  const [revision, setRevision] = useState(0)
  const signalerChangement = useCallback(() => setRevision((n) => n + 1), [])

  const rechargerProjets = useCallback(() => {
    api
      .ListProjects()
      .then((loaded) => {
        setProjects(loaded)
        setError(null)
        setSelectedProjectId((current) => {
          // Le projet verrouillé arrive en tête : c'est la sélection par défaut,
          // et le repli quand le projet actif vient d'être supprimé (§2.1).
          if (current && loaded.some((p) => p.id === current)) return current
          return loaded[0]?.id ?? null
        })
      })
      .catch((err) => setError(String(err)))
  }, [])

  useEffect(rechargerProjets, [rechargerProjets])

  const searching = query.trim().length > 2

  // Échap ferme la recherche et revient à la vue précédente (§2.9).
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setQuery('')
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  /**
   * Navigation vers un élément depuis Priorités ou la Recherche (§2.8, §2.9).
   *
   * Ferme la recherche, bascule sur la vue Projets — tout ce que ces deux vues
   * peuvent ouvrir y vit —, sélectionne le projet et l'onglet natif de
   * l'élément, puis transmet la cible à l'onglet concerné.
   */
  const ouvrir = useCallback((c: CibleOuverture) => {
    setQuery('')
    setView('projects')
    setSelectedProjectId(c.projectId)
    setTab(c.noteId ? 'notes' : c.meetingId || c.instanceId ? 'meetings' : 'tasks')
    setCible(c)
  }, [])

  const ouvrirTache = useCallback(
    (task: domain.Task) => {
      // La navigation peut compléter les filtres pour que la tâche visée reste
      // visible, sans toucher au reste des préférences (§2.4, seule exception à
      // leur persistance).
      setFiltres((f) => {
        const statut = task.cancelled ? 'cancelled' : task.completed ? 'completed' : 'active'
        return {
          ...f,
          status: f.status.includes(statut as never) ? f.status : [...f.status, statut as never],
          importance: f.importance.includes(task.importance as never)
            ? f.importance
            : [...f.importance, task.importance as never],
          dueOnly: task.dueDate ? f.dueOnly : false,
        }
      })
      ouvrir({ projectId: task.projectId, taskId: task.id })
    },
    [ouvrir],
  )

  return (
    <div className="flex h-full flex-col bg-white text-neutral-900">
      <TransverseBar view={view} onView={setView} query={query} onQuery={setQuery} searching={searching} />

      {error ? (
        <p role="alert" className="border-b border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800">
          {error}
        </p>
      ) : null}

      <main className="min-h-0 flex-1">
        {searching ? (
          <SearchView query={query} onClose={() => setQuery('')} onOpen={ouvrir} />
        ) : view === 'priorities' ? (
          <PrioritiesView projects={projects} onOpenTask={ouvrirTache} />
        ) : (
          <Split
            initial={240}
            min={160}
            max={480}
            className="h-full"
            first={
              <ProjectSidebar
                projects={projects}
                selectedId={selectedProjectId}
                onSelect={setSelectedProjectId}
                onChanged={rechargerProjets}
                onError={setError}
                revision={revision}
              />
            }
            second={
              <ProjectView
                tab={tab}
                onTab={setTab}
                projectId={selectedProjectId}
                filtres={filtres}
                onFiltres={setFiltres}
                onDataChanged={signalerChangement}
                cible={cible}
              />
            }
          />
        )}
      </main>
    </div>
  )
}

/**
 * Bande transverse unique (§2.11) : les deux vues de premier niveau à gauche,
 * la recherche à droite. La v1 empilait la recherche au-dessus des onglets ;
 * la v2 les réunit sur une seule ligne.
 */
function TransverseBar({
  view,
  onView,
  query,
  onQuery,
  searching,
}: {
  view: View
  onView: (v: View) => void
  query: string
  onQuery: (q: string) => void
  searching: boolean
}) {
  return (
    <div className="flex items-center gap-4 border-b border-neutral-300 px-3 py-2">
      <div role="tablist" aria-label="Vues" className="flex gap-1">
        <ViewTab label="Projets" active={view === 'projects'} onClick={() => onView('projects')} />
        <ViewTab label="Priorités" active={view === 'priorities'} onClick={() => onView('priorities')} />
      </div>

      <div className="ml-auto flex min-w-0 flex-1 items-center justify-end gap-2">
        <label className="relative flex w-full max-w-md items-center">
          <span aria-hidden className="pointer-events-none absolute left-2 text-neutral-500">
            🔍
          </span>
          <input
            type="search"
            value={query}
            onChange={(event) => onQuery(event.target.value)}
            placeholder="Rechercher dans les tâches, notes, réunions…"
            aria-label="Rechercher dans les tâches, notes, réunions"
            className="w-full rounded border border-neutral-300 py-1 pl-8 pr-8 text-sm focus:border-[var(--color-selection)] focus:outline-none"
          />
          {/* La croix n'apparaît qu'une fois du texte saisi (§2.9). */}
          {query ? (
            <button
              type="button"
              onClick={() => onQuery('')}
              aria-label="Effacer la recherche"
              className="absolute right-2 text-neutral-500 hover:text-neutral-900"
            >
              ✕
            </button>
          ) : null}
        </label>
        {/* La recherche ne se déclenche qu'au-delà de 2 caractères (§2.9) :
            l'indiquer évite de laisser croire à une panne pendant la saisie. */}
        {query && !searching ? (
          <span className="shrink-0 text-xs text-neutral-500">3 caractères minimum</span>
        ) : null}
      </div>
    </div>
  )
}

function ViewTab({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={`rounded px-3 py-1 text-sm ${
        active ? 'bg-[var(--color-selection)] text-white' : 'text-neutral-700 hover:bg-neutral-100'
      }`}
    >
      {label}
    </button>
  )
}

/** Vue Projets : sous-onglets Tâches / Notes / Réunions (§2.11). */
function ProjectView({
  tab,
  onTab,
  projectId,
  filtres,
  onFiltres,
  onDataChanged,
  cible,
}: {
  tab: Tab
  onTab: (t: Tab) => void
  projectId: string | null
  filtres: Filtres
  onFiltres: (f: Filtres) => void
  onDataChanged: () => void
  cible: CibleOuverture | null
}) {
  return (
    <div className="flex h-full flex-col">
      <div role="tablist" aria-label="Onglets du projet" className="flex gap-1 border-b border-neutral-300 px-2 py-1">
        <ViewTab label="Tâches" active={tab === 'tasks'} onClick={() => onTab('tasks')} />
        <ViewTab label="Notes" active={tab === 'notes'} onClick={() => onTab('notes')} />
        <ViewTab label="Réunions" active={tab === 'meetings'} onClick={() => onTab('meetings')} />
      </div>

      <div className="min-h-0 flex-1">
        {!projectId ? (
          <p className="p-4 text-sm text-neutral-500">Sélectionne un projet dans la liste de gauche.</p>
        ) : tab === 'tasks' ? (
          // `key` force le remontage au changement de projet : c'est ce qui
          // déclenche le vidage des sauvegardes en attente et le recalcul du
          // dépliement par défaut (§2.2, §2.6).
          <TasksTab
            key={projectId}
            projectId={projectId}
            filtres={filtres}
            onFiltres={onFiltres}
            onDataChanged={onDataChanged}
            cibleTaskId={cible?.projectId === projectId ? cible.taskId : undefined}
          />
        ) : tab === 'notes' ? (
          <NotesTab key={projectId} projectId={projectId} onDataChanged={onDataChanged} />
        ) : (
          <MeetingsTab
            key={projectId}
            projectId={projectId}
            onDataChanged={onDataChanged}
            cibleInstance={
              cible?.projectId === projectId
                ? { meetingId: cible.meetingId, instanceId: cible.instanceId }
                : undefined
            }
          />
        )}
      </div>
    </div>
  )
}
