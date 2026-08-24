/**
 * Filtres de l'onglet Tâches (§2.4).
 *
 * Trois changements de fond par rapport à la v1 : le statut devient une
 * multi-sélection, la représentation passe du texte aux icônes et pastilles
 * pour tenir sur une seule ligne, et l'état actif se lit à l'épaisseur du
 * contour et non à la couleur de fond — la couleur, elle, porte déjà
 * l'importance et ne peut pas signifier deux choses à la fois.
 */

export type Statut = 'active' | 'completed' | 'cancelled'
export type Importance = 'Critique' | 'Haute' | 'Normale' | 'Basse'

export type Filtres = {
  status: Statut[]
  importance: Importance[]
  dueOnly: boolean
}

export const FILTRES_PAR_DEFAUT: Filtres = {
  status: ['active'],
  importance: ['Critique', 'Haute', 'Normale', 'Basse'],
  dueOnly: false,
}

const COULEUR_IMPORTANCE: Record<Importance, string> = {
  Critique: 'var(--color-critique)',
  Haute: 'var(--color-haute)',
  Normale: 'var(--color-normale)',
  Basse: 'var(--color-basse)',
}

export function TaskFilters({
  filtres,
  onChange,
}: {
  filtres: Filtres
  onChange: (f: Filtres) => void
}) {
  /**
   * Bascule un statut. Le dernier statut décoché se re-coche automatiquement :
   * un arbre vide sans explication passerait pour une panne (§2.4).
   */
  function basculerStatut(s: Statut) {
    const actif = filtres.status.includes(s)
    if (actif && filtres.status.length === 1) return
    onChange({
      ...filtres,
      status: actif ? filtres.status.filter((x) => x !== s) : [...filtres.status, s],
    })
  }

  function basculerImportance(i: Importance) {
    const actif = filtres.importance.includes(i)
    if (actif && filtres.importance.length === 1) return
    onChange({
      ...filtres,
      importance: actif ? filtres.importance.filter((x) => x !== i) : [...filtres.importance, i],
    })
  }

  return (
    <div
      role="group"
      aria-label="Filtres des tâches"
      className="flex flex-wrap items-center gap-3 border-b border-neutral-300 px-2 py-1.5"
    >
      <div className="flex items-center gap-1">
        <BoutonFiltre
          label="Actives"
          actif={filtres.status.includes('active')}
          onClick={() => basculerStatut('active')}
        >
          <span className="inline-block h-3.5 w-3.5 rounded-[2px] border border-neutral-700 bg-white" />
        </BoutonFiltre>

        <BoutonFiltre
          label="Complétées"
          actif={filtres.status.includes('completed')}
          onClick={() => basculerStatut('completed')}
        >
          <span className="inline-flex h-3.5 w-3.5 items-center justify-center rounded-[2px] border border-neutral-700 bg-green-600 text-[10px] leading-none text-white">
            ✓
          </span>
        </BoutonFiltre>

        <BoutonFiltre
          label="Annulées"
          actif={filtres.status.includes('cancelled')}
          onClick={() => basculerStatut('cancelled')}
        >
          <span className="inline-flex items-center gap-0.5">
            <span className="inline-block h-3.5 w-3.5 rounded-[2px] border border-neutral-700 bg-white" />
            <span className="text-[10px] leading-none text-neutral-600 line-through">abc</span>
          </span>
        </BoutonFiltre>
      </div>

      <span aria-hidden className="h-4 w-px bg-neutral-300" />

      <div className="flex items-center gap-1">
        {(Object.keys(COULEUR_IMPORTANCE) as Importance[]).map((imp) => (
          <BoutonFiltre
            key={imp}
            label={imp}
            actif={filtres.importance.includes(imp)}
            onClick={() => basculerImportance(imp)}
          >
            <span
              className="inline-block h-3.5 w-3.5 rounded-full border border-neutral-900"
              style={{ backgroundColor: COULEUR_IMPORTANCE[imp] }}
            />
          </BoutonFiltre>
        ))}
      </div>

      <span aria-hidden className="h-4 w-px bg-neutral-300" />

      <BoutonFiltre
        label="Avec échéance uniquement"
        actif={filtres.dueOnly}
        onClick={() => onChange({ ...filtres, dueOnly: !filtres.dueOnly })}
      >
        <span className="inline-flex h-3.5 w-3.5 items-center justify-center rounded-full border border-neutral-900 text-[9px] leading-none">
          🕐
        </span>
      </BoutonFiltre>
    </div>
  )
}

/**
 * Bouton de filtre : l'état actif se marque par l'épaisseur du contour, jamais
 * par la couleur de fond (§2.4).
 */
function BoutonFiltre({
  label,
  actif,
  onClick,
  children,
}: {
  label: string
  actif: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={actif}
      aria-label={label}
      title={label}
      className={`flex items-center justify-center rounded px-1.5 py-1 ${
        actif ? 'border-2 border-neutral-900' : 'border border-transparent opacity-50'
      }`}
    >
      {children}
    </button>
  )
}
