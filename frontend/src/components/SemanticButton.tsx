import { useCallback, useEffect, useState } from 'react'
import { api } from '../api'
import { ModelMissingDialog } from './ModelMissingDialog'

/** Types d'entités indexables sémantiquement (§2.12). */
export type SemanticEntityType = 'task' | 'note' | 'meeting'

/**
 * Bouton « Ajouter à la recherche sémantique » (§2.12).
 *
 * Présent sur les notes, les instances de réunion et les tâches. Il porte tout
 * l'opt-in de la fonctionnalité : rien n'entre dans l'index sémantique sans un
 * clic ici.
 *
 * Il gère lui-même la fenêtre « modèle absent » plutôt que de la remonter à
 * l'application. C'est ce qui lui permet d'être posé dans n'importe quel onglet
 * sans faire traverser trois composants à un état qui ne concerne que lui.
 */
export function SemanticButton({
  entityType,
  entityId,
}: {
  entityType: SemanticEntityType
  entityId: string
}) {
  const [indexe, setIndexe] = useState<boolean | null>(null)
  const [occupe, setOccupe] = useState(false)
  const [erreur, setErreur] = useState<string | null>(null)
  const [modeleAbsent, setModeleAbsent] = useState(false)

  const relireEtat = useCallback(() => {
    setErreur(null)
    api
      .IsInSemanticIndex(entityId)
      .then(setIndexe)
      .catch(() => setIndexe(false))
  }, [entityId])

  useEffect(relireEtat, [relireEtat])

  async function basculer() {
    setOccupe(true)
    setErreur(null)
    try {
      if (indexe) {
        await api.RemoveFromSemanticIndex(entityId)
        setIndexe(false)
      } else {
        // Le statut est relu à chaque clic plutôt que mémorisé : l'utilisateur
        // peut déposer le modèle pendant que l'application tourne, et un état
        // figé au démarrage lui refuserait indéfiniment la fonctionnalité.
        const statut = await api.GetSemanticStatus()
        if (!statut.modelAvailable) {
          setModeleAbsent(true)
          return
        }
        await api.AddToSemanticIndex(entityType, entityId)
        setIndexe(true)
      }
    } catch (err) {
      setErreur(String(err))
    } finally {
      setOccupe(false)
    }
  }

  // Tant que l'état n'est pas connu, on n'affiche rien plutôt qu'un bouton
  // dont le libellé changerait sous le curseur.
  if (indexe === null) return null

  return (
    <>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={basculer}
          disabled={occupe}
          aria-pressed={indexe}
          title={
            indexe
              ? 'Cette entité remonte dans la recherche sémantique. Ses modifications sont suivies automatiquement.'
              : 'Vectorise le texte pour qu’il remonte sur une reformulation ou malgré une faute de frappe.'
          }
          className={`rounded border px-2 py-1 text-xs disabled:opacity-50 ${
            indexe
              ? 'border-[var(--color-selection)] bg-[var(--color-selection)] text-white'
              : 'border-neutral-300 hover:bg-neutral-100'
          }`}
        >
          <span aria-hidden>🧠</span>{' '}
          {indexe ? 'Dans la recherche sémantique' : 'Ajouter à la recherche sémantique'}
        </button>
        {erreur ? (
          <span role="alert" className="text-xs text-red-700">
            {erreur}
          </span>
        ) : null}
      </div>

      {modeleAbsent ? (
        <ModelMissingDialog
          onClose={() => setModeleAbsent(false)}
          onAvailable={() => {
            // Les fichiers viennent d'apparaître : on referme et on laisse
            // l'utilisateur recliquer, plutôt que de vectoriser dans son dos.
            setModeleAbsent(false)
          }}
        />
      ) : null}
    </>
  )
}
