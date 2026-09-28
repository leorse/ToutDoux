/** Onglet d'une barre `role="tablist"` : l'onglet actif est plein, les autres discrets. */
export function ViewTab({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
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
