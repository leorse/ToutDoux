import { useCallback, useEffect, useRef } from 'react'

/**
 * Sauvegarde automatique avec vidage immédiat (§2.6, §2.7).
 *
 * Le délai seul ne suffit pas : la v1 perdait des frappes quand l'utilisateur
 * cliquait ailleurs avant son expiration. La correction de fond de la v2 est
 * que la sauvegarde part **aussi** immédiatement sur perte de focus, changement
 * de sélection ou changement de projet.
 *
 * `schedule` arme le délai, `flush` écrit tout de suite ce qui est en attente.
 * Un flush sans rien en attente ne déclenche aucun appel : sans cette garde,
 * chaque clic dans l'application produirait une écriture inutile.
 */
export function useAutosave<T>(save: (valeur: T) => void, delaiMs: number) {
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const enAttente = useRef<{ valeur: T } | null>(null)

  // `save` est souvent une lambda recréée à chaque rendu ; la garder dans une
  // ref évite que `flush` capture une version périmée.
  const saveRef = useRef(save)
  useEffect(() => {
    saveRef.current = save
  }, [save])

  const flush = useCallback(() => {
    if (timer.current) {
      clearTimeout(timer.current)
      timer.current = null
    }
    if (!enAttente.current) return
    const { valeur } = enAttente.current
    enAttente.current = null
    saveRef.current(valeur)
  }, [])

  const schedule = useCallback(
    (valeur: T) => {
      enAttente.current = { valeur }
      if (timer.current) clearTimeout(timer.current)
      timer.current = setTimeout(flush, delaiMs)
    },
    [delaiMs, flush],
  )

  // Au démontage — changement de note, de tâche ou de projet — ce qui est en
  // attente doit partir. C'est le cas que la v1 perdait.
  useEffect(() => () => flush(), [flush])

  return { schedule, flush }
}
