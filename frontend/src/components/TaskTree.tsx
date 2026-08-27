import { useMemo, useState } from 'react'
import type { domain, duedate } from '../../wailsjs/go/models'
import { surLeFond } from '../fond'
import type { MenuState } from './ContextMenu'

export type TaskTreeProps = {
  tasks: domain.Task[]
  visible: Set<string>
  expanded: Set<string>
  due: Record<string, duedate.Info>
  selectedId: string | null
  onSelect: (id: string) => void
  onToggleExpand: (id: string) => void
  onToggleCompleted: (id: string) => void
  onReparent: (draggedId: string, targetId: string | null) => void
  onMenu: (state: MenuState) => void
  menuPourTache: (task: domain.Task, x: number, y: number) => MenuState
  menuPourFond: (x: number, y: number) => MenuState
  /** Double-clic sur le fond : création d'une tâche racine (§2.2). */
  onCreerRacine: () => void
}

/**
 * Arbre hiérarchique des tâches (§2.2, §2.5).
 *
 * Le composant ne calcule aucune règle : la visibilité sous filtre, le
 * dépliement par défaut et les libellés d'échéance arrivent déjà résolus depuis
 * le backend. Il se contente de les dessiner — c'est ce qui permet de le tester
 * avec des fixtures, sans backend (§3.10).
 */
export function TaskTree(props: TaskTreeProps) {
  const { tasks, visible, onMenu, menuPourFond, onReparent, onCreerRacine } = props

  // Regroupement par parent, chaque fratrie triée par orderIndex.
  const enfants = useMemo(() => {
    const map = new Map<string, domain.Task[]>()
    for (const t of tasks) {
      const cle = t.parentId ?? '__racine__'
      map.set(cle, [...(map.get(cle) ?? []), t])
    }
    for (const [, liste] of map) liste.sort((a, b) => a.orderIndex - b.orderIndex)
    return map
  }, [tasks])

  const racines = enfants.get('__racine__') ?? []

  return (
    <div
      className="h-full min-h-full p-1"
      // Clic droit sur le fond : création d'une tâche racine (§2.2).
      onContextMenu={(e) => {
        e.preventDefault()
        onMenu(menuPourFond(e.clientX, e.clientY))
      }}
      // Double-clic sur le fond : même création, sans passer par le menu.
      // Le filtre écarte les double-clics tombés sur une tâche, qui sert à
      // déplier et à sélectionner.
      onDoubleClick={(e) => {
        if (surLeFond(e)) onCreerRacine()
      }}
      // Déposer sur le fond ramène la tâche à la racine du projet.
      onDragOver={(e) => e.preventDefault()}
      onDrop={(e) => {
        const id = e.dataTransfer.getData('text/task-id')
        if (id) onReparent(id, null)
      }}
    >
      <ul role="tree" aria-label="Tâches" className="flex flex-col">
        {racines
          .filter((t) => visible.has(t.id))
          .map((t) => (
            <Noeud key={t.id} task={t} profondeur={0} enfants={enfants} {...props} />
          ))}
      </ul>
    </div>
  )
}

function Noeud({
  task,
  profondeur,
  enfants,
  visible,
  expanded,
  due,
  selectedId,
  onSelect,
  onToggleExpand,
  onToggleCompleted,
  onReparent,
  onMenu,
  menuPourTache,
  ...reste
}: TaskTreeProps & { task: domain.Task; profondeur: number; enfants: Map<string, domain.Task[]> }) {
  const [survol, setSurvol] = useState(false)
  const mesEnfants = (enfants.get(task.id) ?? []).filter((t) => visible.has(t.id))
  const deplie = expanded.has(task.id)
  const info = due[task.id]

  return (
    <li role="treeitem" data-ligne aria-expanded={mesEnfants.length ? deplie : undefined}>
      <div
        draggable
        onDragStart={(e) => {
          e.dataTransfer.setData('text/task-id', task.id)
          e.dataTransfer.effectAllowed = 'move'
        }}
        onDragOver={(e) => {
          e.preventDefault()
          setSurvol(true)
        }}
        onDragLeave={() => setSurvol(false)}
        onDrop={(e) => {
          e.preventDefault()
          e.stopPropagation()
          setSurvol(false)
          const id = e.dataTransfer.getData('text/task-id')
          // Le refus d'un cycle est tranché par le backend, qui rend une erreur
          // explicite : le décider aussi ici dupliquerait la règle du §2.2.
          if (id && id !== task.id) onReparent(id, task.id)
        }}
        onClick={() => onSelect(task.id)}
        onContextMenu={(e) => {
          e.preventDefault()
          e.stopPropagation()
          onSelect(task.id)
          onMenu(menuPourTache(task, e.clientX, e.clientY))
        }}
        style={{ paddingLeft: profondeur * 18 + 4, backgroundColor: fond(task) }}
        className={`flex cursor-default items-center gap-1.5 rounded border py-1 pr-2 text-sm ${
          selectedId === task.id ? 'border-2 border-[var(--color-selection)]' : 'border-transparent'
        } ${survol ? 'outline-2 outline-dashed outline-[var(--color-selection)]' : ''}`}
      >
        <button
          type="button"
          aria-label={deplie ? 'Replier' : 'Déplier'}
          onClick={(e) => {
            e.stopPropagation()
            onToggleExpand(task.id)
          }}
          className={`w-4 shrink-0 text-xs text-neutral-600 ${mesEnfants.length ? '' : 'invisible'}`}
        >
          {deplie ? '▾' : '▸'}
        </button>

        <input
          type="checkbox"
          checked={task.completed}
          aria-label={`Terminer ${task.name}`}
          onClick={(e) => e.stopPropagation()}
          onChange={() => onToggleCompleted(task.id)}
          className="shrink-0"
        />

        <span className={`truncate ${task.cancelled ? 'text-neutral-500 line-through' : ''}`}>
          {task.name}
        </span>

        {info ? (
          <span className="ml-1 shrink-0 whitespace-nowrap text-xs text-neutral-600">
            {/* ⏰ pour une échéance imminente OU dépassée ; 🕐 sinon (§2.3). */}
            <span aria-hidden>{info.urgent || info.overdue ? '⏰' : '🕐'}</span> ({info.text})
          </span>
        ) : null}
      </div>

      {deplie && mesEnfants.length ? (
        <ul role="group" className="flex flex-col">
          {mesEnfants.map((enfant) => (
            <Noeud
              key={enfant.id}
              task={enfant}
              profondeur={profondeur + 1}
              enfants={enfants}
              visible={visible}
              expanded={expanded}
              due={due}
              selectedId={selectedId}
              onSelect={onSelect}
              onToggleExpand={onToggleExpand}
              onToggleCompleted={onToggleCompleted}
              onReparent={onReparent}
              onMenu={onMenu}
              menuPourTache={menuPourTache}
              {...reste}
            />
          ))}
        </ul>
      ) : null}
    </li>
  )
}

/** Couleur de fond d'une tâche (§2.2). Le statut prime sur l'importance. */
function fond(task: domain.Task): string {
  if (task.cancelled) return 'var(--color-annulee)'
  if (task.completed) return 'var(--color-completee)'
  switch (task.importance) {
    case 'Critique':
      return 'var(--color-critique)'
    case 'Haute':
      return 'var(--color-haute)'
    case 'Basse':
      return 'var(--color-basse)'
    default:
      return 'var(--color-normale)'
  }
}
