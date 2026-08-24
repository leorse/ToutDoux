import { useCallback, useEffect, useRef, useState } from 'react'

type Props = {
  /** Orientation de la séparation. */
  direction?: 'horizontal' | 'vertical'
  /** Taille initiale du premier panneau, en pixels. */
  initial: number
  min?: number
  max?: number
  first: React.ReactNode
  second: React.ReactNode
  className?: string
}

/**
 * Sépare deux panneaux par une poignée redimensionnable au glisser-déposer.
 *
 * Toutes les séparations de l'application passent par ce composant (§2.11) :
 * sidebar ↔ contenu, arbre ↔ détail, liste ↔ éditeur, et les deux séparations
 * de la vue Réunions. Les centraliser ici est ce qui rend le comportement
 * cohérent partout, plutôt que réimplémenté trois fois avec trois inerties
 * légèrement différentes.
 */
export function Split({
  direction = 'horizontal',
  initial,
  min = 140,
  max = 900,
  first,
  second,
  className = '',
}: Props) {
  const [size, setSize] = useState(initial)
  const [dragging, setDragging] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const horizontal = direction === 'horizontal'

  const onMove = useCallback(
    (event: MouseEvent) => {
      const box = containerRef.current?.getBoundingClientRect()
      if (!box) return
      const raw = horizontal ? event.clientX - box.left : event.clientY - box.top
      setSize(Math.min(max, Math.max(min, raw)))
    },
    [horizontal, min, max],
  )

  useEffect(() => {
    if (!dragging) return
    const stop = () => setDragging(false)
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', stop)

    // Pendant le glissement, le curseur doit rester celui du redimensionnement
    // même s'il sort de la poignée, et le texte ne doit pas se sélectionner —
    // sinon l'utilisateur surligne la moitié de l'écran en déplaçant la barre.
    const previousCursor = document.body.style.cursor
    document.body.style.cursor = horizontal ? 'col-resize' : 'row-resize'
    document.body.style.userSelect = 'none'

    return () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', stop)
      document.body.style.cursor = previousCursor
      document.body.style.userSelect = ''
    }
  }, [dragging, onMove, horizontal])

  return (
    <div
      ref={containerRef}
      className={`flex min-h-0 min-w-0 ${horizontal ? 'flex-row' : 'flex-col'} ${className}`}
    >
      <div
        className="min-h-0 min-w-0 overflow-auto"
        style={horizontal ? { width: size, flexShrink: 0 } : { height: size, flexShrink: 0 }}
      >
        {first}
      </div>

      <div
        role="separator"
        aria-orientation={horizontal ? 'vertical' : 'horizontal'}
        tabIndex={0}
        onMouseDown={() => setDragging(true)}
        onKeyDown={(event) => {
          // Le clavier doit pouvoir faire ce que fait la souris (§6,
          // accessibilité) : une poignée qui n'est atteignable qu'au pointeur
          // rend le panneau impossible à redimensionner sans souris.
          const step = event.shiftKey ? 40 : 10
          const decrease = horizontal ? 'ArrowLeft' : 'ArrowUp'
          const increase = horizontal ? 'ArrowRight' : 'ArrowDown'
          if (event.key === decrease) setSize((s) => Math.max(min, s - step))
          if (event.key === increase) setSize((s) => Math.min(max, s + step))
        }}
        className={`shrink-0 bg-neutral-200 transition-colors hover:bg-neutral-400 focus:bg-neutral-400 focus:outline-none dark:bg-neutral-700 dark:hover:bg-neutral-500 ${
          horizontal ? 'w-1 cursor-col-resize' : 'h-1 cursor-row-resize'
        } ${dragging ? 'bg-neutral-400 dark:bg-neutral-500' : ''}`}
      />

      <div className="min-h-0 min-w-0 flex-1 overflow-auto">{second}</div>
    </div>
  )
}
