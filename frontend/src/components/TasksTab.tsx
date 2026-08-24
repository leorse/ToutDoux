import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '../api'
import type { domain, duedate } from '../../wailsjs/go/models'
import { ContextMenu, type MenuItem, type MenuState } from './ContextMenu'
import { Split } from './Split'
import { TaskDetail } from './TaskDetail'
import { FILTRES_PAR_DEFAUT, TaskFilters, type Filtres } from './TaskFilters'
import { TaskTree } from './TaskTree'

/** Rythme de rafraîchissement des libellés d'échéance (§2.3). */
const REFRESH_MS = 30_000

export function TasksTab({
  projectId,
  filtres,
  onFiltres,
  onDataChanged,
  cibleTaskId,
}: {
  projectId: string
  /** Les filtres vivent au-dessus de l'onglet : ils ne sont plus réinitialisés
   *  au changement de projet, contrairement à la v1 (§2.4). */
  filtres: Filtres
  onFiltres: (f: Filtres) => void
  /** Signale à la coquille qu'une écriture a eu lieu, pour que la sidebar
   *  recharge ses compteurs et ses couleurs (§2.1). */
  onDataChanged: () => void
  /** Tâche à ouvrir quand on arrive depuis Priorités ou la Recherche (§2.8, §2.9). */
  cibleTaskId?: string
}) {
  const [tasks, setTasks] = useState<domain.Task[]>([])
  const [visible, setVisible] = useState<Set<string>>(new Set())
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [due, setDue] = useState<Record<string, duedate.Info>>({})
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [menu, setMenu] = useState<MenuState>(null)
  const [erreur, setErreur] = useState<string | null>(null)

  const selected = tasks.find((t) => t.id === selectedId) ?? null

  /**
   * Recharge la vue. `replierSelonDefaut` n'est vrai qu'au chargement et au
   * changement de projet : recalculer le dépliement en continu replierait une
   * branche sous les yeux de l'utilisateur pendant qu'il travaille (§2.2).
   */
  const recharger = useCallback(
    async (replierSelonDefaut: boolean) => {
      try {
        const vue = await api.GetTaskView(projectId, filtres)
        setTasks(vue.tasks)
        setVisible(new Set(vue.visible))
        setDue(vue.due ?? {})
        if (replierSelonDefaut) setExpanded(new Set(vue.defaultExpanded))
        setErreur(null)
      } catch (err) {
        setErreur(String(err))
      }
    },
    [projectId, filtres],
  )

  useEffect(() => {
    recharger(true)
    setSelectedId(null)
  }, [projectId])

  useEffect(() => {
    recharger(false)
  }, [filtres])

  // Arrivée depuis Priorités, la Recherche ou la barre système : on sélectionne
  // la tâche visée et on déplie sa chaîne d'ancêtres, sinon elle resterait
  // invisible dans une branche repliée (§2.8, §2.9, §2.10).
  //
  // La cible n'est appliquée QU'UNE FOIS par valeur. Cet effet dépend de
  // `tasks`, qui change à chaque écriture — or la sauvegarde du panneau de
  // détail écrit toutes les 500 ms pendant la frappe. Sans ce garde-fou, chaque
  // caractère tapé re-sélectionnait la tâche d'où l'on venait, arrachant le
  // focus vers une autre ligne de l'arbre.
  const cibleAppliquee = useRef<string | undefined>(undefined)
  useEffect(() => {
    if (!cibleTaskId || tasks.length === 0) return
    if (cibleAppliquee.current === cibleTaskId) return
    cibleAppliquee.current = cibleTaskId
    setSelectedId(cibleTaskId)

    // Remontée de la chaîne parent par parent. Ce n'est pas une règle métier
    // mais un simple parcours de données, d'où son maintien ici plutôt qu'un
    // aller-retour vers le backend.
    setExpanded((prev) => {
      const next = new Set(prev)
      const parDefaut = new Map(tasks.map((t) => [t.id, t]))
      let courant = parDefaut.get(cibleTaskId)?.parentId ?? null
      const vus = new Set<string>()
      while (courant && !vus.has(courant)) {
        vus.add(courant)
        next.add(courant)
        courant = parDefaut.get(courant)?.parentId ?? null
      }
      return next
    })
  }, [cibleTaskId, tasks])

  // Rafraîchissement périodique : « dans 12 min » doit devenir « dans 11 min »
  // sans intervention (§2.3).
  useEffect(() => {
    const id = setInterval(() => recharger(false), REFRESH_MS)
    return () => clearInterval(id)
  }, [recharger])

  async function agir(action: () => Promise<unknown>) {
    try {
      await action()
      await recharger(false)
      // Le compteur et la couleur du projet dépendent des tâches : toute
      // écriture doit les rafraîchir, pas seulement l'arbre.
      onDataChanged()
      setErreur(null)
    } catch (err) {
      // Les refus du domaine — anti-cycle en tête — doivent être montrés, pas
      // avalés : la spec exige un message explicite (§2.2).
      setErreur(String(err))
    }
  }

  const deplier = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })

  function toutEtendre() {
    setExpanded(new Set(tasks.map((t) => t.id)))
  }

  async function creer(parentId: string) {
    const nom = window.prompt(parentId ? 'Nom de la sous-tâche ?' : 'Nom de la tâche ?')
    if (!nom?.trim()) return
    await agir(async () => {
      const res = await api.CreateTask(projectId, parentId, nom.trim())
      if (parentId) setExpanded((prev) => new Set(prev).add(parentId))
      setSelectedId(res.task.id)
    })
  }

  async function supprimer(task: domain.Task) {
    const n = await api.TaskDeletionSummary(task.id)
    const message =
      n > 0
        ? `Supprimer « ${task.name} » et ses ${n} sous-tâche(s) ?`
        : `Supprimer « ${task.name} » ?`
    if (!window.confirm(message)) return
    await agir(async () => {
      await api.DeleteTask(task.id)
      if (selectedId === task.id) setSelectedId(null)
    })
  }

  const menuPourTache = (task: domain.Task, x: number, y: number): MenuState => ({
    x,
    y,
    items: [
      { kind: 'action', label: '+ Sous-tâche', onSelect: () => creer(task.id) },
      { kind: 'action', label: '+ Nouvelle tâche', onSelect: () => creer('') },
      { kind: 'separator' },
      { kind: 'action', label: 'Envoyer au début', onSelect: () => agir(() => api.MoveTask(task.id, 'top')) },
      { kind: 'action', label: 'Monter', onSelect: () => agir(() => api.MoveTask(task.id, 'up')) },
      { kind: 'action', label: 'Descendre', onSelect: () => agir(() => api.MoveTask(task.id, 'down')) },
      { kind: 'action', label: 'Envoyer à la fin', onSelect: () => agir(() => api.MoveTask(task.id, 'bottom')) },
      { kind: 'separator' },
      {
        kind: 'action',
        label: task.cancelled ? 'Réactiver' : 'Annuler',
        onSelect: () => agir(() => api.ToggleTaskCancelled(task.id)),
      },
      { kind: 'separator' },
      { kind: 'action', label: 'Tout étendre', onSelect: toutEtendre },
      { kind: 'action', label: 'Tout réduire', onSelect: () => setExpanded(new Set()) },
      { kind: 'separator' },
      { kind: 'action', label: 'Supprimer', danger: true, onSelect: () => supprimer(task) },
    ] satisfies MenuItem[],
  })

  const menuPourFond = (x: number, y: number): MenuState => ({
    x,
    y,
    items: [
      { kind: 'action', label: '+ Nouvelle tâche', onSelect: () => creer('') },
      { kind: 'separator' },
      { kind: 'action', label: 'Tout étendre', onSelect: toutEtendre },
      { kind: 'action', label: 'Tout réduire', onSelect: () => setExpanded(new Set()) },
    ],
  })

  return (
    <div className="flex h-full flex-col">
      <TaskFilters filtres={filtres} onChange={onFiltres} />

      {erreur ? (
        <p role="alert" className="border-b border-red-300 bg-red-50 px-3 py-1.5 text-xs text-red-800">
          {erreur}
        </p>
      ) : null}

      <div className="min-h-0 flex-1">
        <Split
          initial={520}
          min={240}
          max={1000}
          className="h-full"
          first={
            <TaskTree
              tasks={tasks}
              visible={visible}
              expanded={expanded}
              due={due}
              selectedId={selectedId}
              onSelect={setSelectedId}
              onToggleExpand={deplier}
              onToggleCompleted={(id) => agir(() => api.ToggleTaskCompleted(id))}
              onReparent={(draggedId, targetId) =>
                agir(() => api.ReparentTask(draggedId, targetId ?? ''))
              }
              onMenu={setMenu}
              menuPourTache={menuPourTache}
              menuPourFond={menuPourFond}
            />
          }
          second={
            <TaskDetail
              task={selected}
              due={selectedId ? due[selectedId] : undefined}
              onPatch={(patch) => agir(() => api.UpdateTask(selectedId!, patch as never))}
              onQuickDue={(kind) => agir(() => api.SetTaskDueDateQuick(selectedId!, kind))}
              onSetDue={(local) =>
                agir(() =>
                  api.UpdateTask(selectedId!, { dueDate: new Date(local).toISOString() } as never),
                )
              }
            />
          }
        />
      </div>

      <ContextMenu state={menu} onClose={() => setMenu(null)} />
    </div>
  )
}

export { FILTRES_PAR_DEFAUT }
export type { Filtres }
