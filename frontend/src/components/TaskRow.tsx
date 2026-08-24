import type { domain } from '../../wailsjs/go/models'

const COULEUR_IMPORTANCE: Record<string, string> = {
  Critique: 'var(--color-critique)',
  Haute: 'var(--color-haute)',
  Normale: 'var(--color-normale)',
  Basse: 'var(--color-basse)',
}

/**
 * Pastille colorée d'importance (§2.8, §2.9).
 *
 * La v1 affichait « [Crit] » et « [Haute] » en texte ; la v2 utilise la même
 * teinte que le fond des tâches, ce qui permet de reconnaître la criticité sans
 * lire.
 */
export function PastilleImportance({ importance }: { importance: string }) {
  return (
    <span
      aria-label={`Importance ${importance}`}
      title={importance}
      className="inline-block h-3 w-3 shrink-0 rounded-full border border-neutral-900"
      style={{ backgroundColor: COULEUR_IMPORTANCE[importance] ?? 'var(--color-normale)' }}
    />
  )
}

/**
 * Ligne de tâche en lecture seule, partagée par la vue Priorités et la vue
 * Recherche (§2.8, §2.9).
 *
 * La case à cocher reflète l'état réel de la tâche et n'est pas manipulable :
 * ces deux vues montrent, elles ne modifient pas.
 */
export function LigneTache({
  task,
  projectName,
  echeance,
  selected,
  onSelect,
  onOpen,
}: {
  task: domain.Task
  projectName?: string
  echeance?: string
  selected: boolean
  onSelect: () => void
  onOpen: () => void
}) {
  return (
    <li>
      <button
        type="button"
        onClick={onSelect}
        onDoubleClick={onOpen}
        aria-current={selected ? 'true' : undefined}
        className={`flex w-full items-center gap-2 rounded border px-2 py-1 text-left text-sm ${
          selected ? 'border-2 border-[var(--color-selection)] bg-neutral-100' : 'border-transparent hover:bg-neutral-50'
        }`}
      >
        <input
          type="checkbox"
          checked={task.completed}
          readOnly
          tabIndex={-1}
          aria-label={task.completed ? 'Terminée' : 'Active'}
          className="pointer-events-none shrink-0"
        />
        <PastilleImportance importance={task.importance} />
        <span className={`truncate ${task.cancelled ? 'text-neutral-500 line-through' : ''}`}>
          {task.name}
        </span>
        {projectName ? (
          <span className="ml-auto shrink-0 rounded bg-neutral-200 px-1.5 py-0.5 text-xs text-neutral-700">
            {projectName}
          </span>
        ) : null}
        {echeance ? (
          <span className="shrink-0 whitespace-nowrap text-xs text-neutral-600">{echeance}</span>
        ) : null}
      </button>
    </li>
  )
}
