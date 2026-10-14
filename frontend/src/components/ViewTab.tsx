/**
 * Onglet d'une barre `role="tablist"` : l'onglet actif est plein, les autres discrets.
 *
 * Avec `onClose`, l'onglet porte une croix de fermeture. Elle est un bouton
 * frère de l'onglet et non un enfant : un bouton imbriqué dans un bouton n'est
 * pas du HTML valide, et l'onglet garde ainsi son seul libellé comme nom
 * accessible.
 */
export function ViewTab({
  label,
  active,
  onClick,
  onClose,
  closeLabel,
}: {
  label: string
  active: boolean
  onClick: () => void
  onClose?: () => void
  closeLabel?: string
}) {
  const couleurs = active ? 'bg-[var(--color-selection)] text-white' : 'text-neutral-700 hover:bg-neutral-100'

  if (!onClose) {
    return (
      <button type="button" role="tab" aria-selected={active} onClick={onClick} className={`rounded px-3 py-1 text-sm ${couleurs}`}>
        {label}
      </button>
    )
  }

  return (
    <span className={`inline-flex items-center rounded text-sm ${couleurs}`}>
      <button type="button" role="tab" aria-selected={active} onClick={onClick} className="py-1 pl-3 pr-1">
        {label}
      </button>
      <button
        type="button"
        onClick={(event) => {
          // La fermeture ne doit pas d'abord sélectionner l'onglet.
          event.stopPropagation()
          onClose()
        }}
        aria-label={closeLabel ?? `Fermer ${label}`}
        title={closeLabel ?? `Fermer ${label}`}
        className={`mr-1 rounded px-1 leading-none ${active ? 'hover:bg-white/20' : 'text-neutral-500 hover:bg-neutral-200 hover:text-neutral-900'}`}
      >
        ✕
      </button>
    </span>
  )
}
