import { useEffect, useRef } from 'react'

export type MenuItem =
  | { kind: 'action'; label: string; onSelect: () => void; disabled?: boolean; danger?: boolean }
  | { kind: 'separator' }

export type MenuState = { x: number; y: number; items: MenuItem[] } | null

/**
 * Menu contextuel (§2.2).
 *
 * En v2, toutes les actions sur les tâches, les notes et les réunions passent
 * par le clic droit : il n'y a plus aucun bouton de barre d'outils. Ce composant
 * est donc le seul chemin vers la création, la suppression et le déplacement.
 */
export function ContextMenu({ state, onClose }: { state: MenuState; onClose: () => void }) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!state) return
    const onPointer = (event: MouseEvent) => {
      if (!ref.current?.contains(event.target as Node)) onClose()
    }
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
    }
    // `capture` pour fermer avant qu'un autre gestionnaire n'ouvre un second
    // menu : sans cela, un clic droit sur une autre tâche en laisserait deux.
    window.addEventListener('mousedown', onPointer, true)
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('mousedown', onPointer, true)
      window.removeEventListener('keydown', onKey)
    }
  }, [state, onClose])

  if (!state) return null

  return (
    <div
      ref={ref}
      role="menu"
      // Le menu est positionné au curseur, et ramené dans la fenêtre s'il
      // déborderait — un menu ouvert en bas d'écran serait sinon inatteignable.
      style={{
        left: Math.min(state.x, window.innerWidth - 220),
        top: Math.min(state.y, window.innerHeight - state.items.length * 32 - 16),
      }}
      className="fixed z-50 min-w-52 rounded border border-neutral-300 bg-white py-1 shadow-lg"
    >
      {state.items.map((item, i) =>
        item.kind === 'separator' ? (
          <hr key={i} className="my-1 border-neutral-200" />
        ) : (
          <button
            key={i}
            role="menuitem"
            type="button"
            disabled={item.disabled}
            onClick={() => {
              item.onSelect()
              onClose()
            }}
            className={`block w-full px-3 py-1.5 text-left text-sm disabled:cursor-not-allowed disabled:text-neutral-400 ${
              item.danger ? 'text-red-700 hover:bg-red-50' : 'hover:bg-neutral-100'
            }`}
          >
            {item.label}
          </button>
        ),
      )}
    </div>
  )
}
