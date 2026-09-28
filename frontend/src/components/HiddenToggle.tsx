/**
 * Bascule icône seule qui montre ou cache les éléments masqués d'une liste
 * (projets, notes ou réunions) (§2.1, §2.6, §2.7).
 *
 * Œil ouvert (par défaut) : les éléments cachés ne s'affichent pas. Œil fermé :
 * ils s'affichent, en plus discret. Il n'y a pas de texte visible — seulement un
 * `aria-label` — car l'icône porte l'état à elle seule, comme les filtres de
 * tâches.
 */
export function HiddenToggle({ montrerCaches, onToggle }: { montrerCaches: boolean; onToggle: () => void }) {
  const label = montrerCaches ? 'Masquer les éléments cachés' : 'Afficher les éléments cachés'
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-pressed={montrerCaches}
      aria-label={label}
      title={label}
      className={`flex items-center justify-center rounded px-1.5 py-1 ${
        montrerCaches ? 'border-2 border-neutral-900' : 'border border-transparent opacity-50'
      }`}
    >
      {montrerCaches ? <OeilFerme /> : <OeilOuvert />}
    </button>
  )
}

function OeilOuvert() {
  return (
    <svg aria-hidden viewBox="0 0 20 20" className="h-3.5 w-3.5" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M1 10s3.5-6 9-6 9 6 9 6-3.5 6-9 6-9-6-9-6Z" strokeLinejoin="round" />
      <circle cx="10" cy="10" r="2.5" />
    </svg>
  )
}

function OeilFerme() {
  return (
    <svg aria-hidden viewBox="0 0 20 20" className="h-3.5 w-3.5" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M1 10s3.5-6 9-6 9 6 9 6-3.5 6-9 6-9-6-9-6Z" strokeLinejoin="round" />
      <path d="M2 2l16 16" strokeLinecap="round" />
    </svg>
  )
}
