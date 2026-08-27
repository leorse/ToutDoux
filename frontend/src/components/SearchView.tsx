import { useEffect, useState } from 'react'
import { api } from '../api'
import type { main } from '../../wailsjs/go/models'
import { Split } from './Split'
import { PastilleImportance } from './TaskRow'

/** Anti-rebond de la saisie (§2.9). */
const DEBOUNCE_MS = 300

export type CibleOuverture = {
  projectId: string
  taskId?: string
  noteId?: string
  meetingId?: string
  instanceId?: string
}

/**
 * Vue des résultats de recherche (§2.9).
 *
 * Elle remplace tout le contenu principal, quelle que soit la vue affichée
 * avant — Projets ou Priorités. La recherche est toujours globale : le mode
 * « Projet » de la v1 a été retiré.
 */
export function SearchView({
  query,
  semantique = false,
  onClose,
  onOpen,
}: {
  query: string
  /** Mode sémantique (§2.12), piloté par la bascule 🧠 de la bande transverse. */
  semantique?: boolean
  onClose: () => void
  onOpen: (cible: CibleOuverture) => void
}) {
  const [results, setResults] = useState<main.SearchResult[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [erreur, setErreur] = useState<string | null>(null)
  const [indexVide, setIndexVide] = useState(false)

  const selected = results.find((r) => r.id === selectedId) ?? null

  useEffect(() => {
    // Anti-rebond : sans lui, chaque frappe déclencherait une requête, et les
    // réponses pourraient revenir dans le désordre (§2.9).
    const timer = setTimeout(() => {
      // Les deux modes ne sont jamais fusionnés : l'utilisateur en choisit un,
      // et la liste vient d'une seule source (§2.12).
      const recherche = semantique ? api.SearchSemantic(query) : api.SearchGlobal(query)
      recherche
        .then((r) => {
          setResults(r)
          setSelectedId((current) => (r.some((x) => x.id === current) ? current : (r[0]?.id ?? null)))
          setErreur(null)
        })
        .catch((err) => setErreur(String(err)))
    }, DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [query, semantique])

  // Un index sémantique vide se dit, il ne s'affiche pas comme « aucun
  // résultat » : la nuance est tout sauf cosmétique, elle sépare « rien ne
  // correspond » de « rien n'a été indexé » (§2.12).
  useEffect(() => {
    if (!semantique) {
      setIndexVide(false)
      return
    }
    api.GetSemanticStatus().then(
      (s) => setIndexVide(s.indexedCount === 0),
      () => setIndexVide(false),
    )
  }, [semantique, results])

  function ouvrir(r: main.SearchResult) {
    // Chaque type retourne dans son onglet natif ; la bascule vers la vue
    // Projets est faite par l'appelant, puisque tout ce que la recherche peut
    // ouvrir y vit (§2.9).
    onOpen({
      projectId: r.projectId,
      taskId: r.task?.id,
      noteId: r.note?.id,
      meetingId: r.meeting?.id,
      instanceId: r.instance?.id,
    })
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 border-b border-neutral-300 px-3 py-1.5">
        <span className="text-sm text-neutral-700">
          {results.length} résultat{results.length > 1 ? 's' : ''}
        </span>
        {semantique ? (
          <span className="rounded bg-[var(--color-selection)] px-1.5 py-0.5 text-xs text-white">
            🧠 sémantique
          </span>
        ) : null}
        <button
          type="button"
          onClick={onClose}
          className="ml-auto rounded border border-neutral-300 px-2 py-0.5 text-sm hover:bg-neutral-100"
        >
          Fermer ⊗
        </button>
      </div>

      {erreur ? (
        <p role="alert" className="border-b border-red-300 bg-red-50 px-3 py-1.5 text-xs text-red-800">
          {erreur}
        </p>
      ) : null}

      <div className="min-h-0 flex-1">
        <Split
          initial={480}
          min={260}
          max={900}
          className="h-full"
          first={
            results.length === 0 ? (
              <p className="p-4 text-sm text-neutral-500">
                {semantique && indexVide
                  ? 'L’index sémantique est vide. Ajoute des notes, des comptes rendus ou des tâches avec leur bouton 🧠 : seules les entités ajoutées peuvent remonter ici.'
                  : 'Aucun résultat.'}
              </p>
            ) : (
              <ul aria-label="Résultats" className="flex flex-col p-1">
                {results.map((r) => (
                  <li key={r.id}>
                    <button
                      type="button"
                      onClick={() => setSelectedId(r.id)}
                      onDoubleClick={() => ouvrir(r)}
                      aria-current={r.id === selectedId ? 'true' : undefined}
                      title={
                        semantique
                          ? 'Similarité avec la requête. Comparer les lignes entre elles : ' +
                            'sur ce modèle, les valeurs tiennent toutes entre 0,79 et 0,88.'
                          : undefined
                      }
                      className={`flex w-full items-center gap-2 rounded border px-2 py-1 text-left text-sm ${
                        r.id === selectedId
                          ? 'border-2 border-[var(--color-selection)] bg-neutral-100'
                          : 'border-transparent hover:bg-neutral-50'
                      }`}
                    >
                      <Icone result={r} />
                      <span className="truncate">{r.title || 'Sans titre'}</span>
                      {/* Score de similarité, en mode sémantique seulement.
                          Affiché brut (0,873) et non en pourcentage : « 87 % »
                          se lirait comme une confiance, alors que c'est un
                          nombre à comparer aux lignes voisines — l'écart entre
                          deux résultats est ce qui a du sens, pas la valeur
                          absolue (§2.12). */}
                      {semantique ? (
                        <span className="ml-auto shrink-0 font-mono text-xs tabular-nums text-neutral-500">
                          {r.score.toFixed(3)}
                        </span>
                      ) : null}
                      <span
                        className={`shrink-0 rounded bg-neutral-200 px-1.5 py-0.5 text-xs text-neutral-700 ${
                          semantique ? '' : 'ml-auto'
                        }`}
                      >
                        {r.projectName}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )
          }
          second={<Apercu result={selected} onOpen={ouvrir} />}
        />
      </div>
    </div>
  )
}

/**
 * Icône du résultat selon son type (§2.9).
 *
 * Une tâche affiche une vraie case à cocher reflétant son état, plus une
 * pastille d'importance — le simple « ✓ » de la v1 ne disait ni si la tâche
 * était faite, ni si elle était critique.
 */
function Icone({ result }: { result: main.SearchResult }) {
  if (result.task) {
    return (
      <span className="flex shrink-0 items-center gap-1">
        <input
          type="checkbox"
          checked={result.task.completed}
          readOnly
          tabIndex={-1}
          aria-label={result.task.completed ? 'Tâche terminée' : 'Tâche active'}
          className="pointer-events-none"
        />
        <PastilleImportance importance={result.task.importance} />
      </span>
    )
  }
  const emoji = result.type === 'note' ? '📝' : result.type === 'project' ? '📁' : '📞'
  const label = result.type === 'note' ? 'Note' : result.type === 'project' ? 'Projet' : 'Réunion'
  return (
    <span aria-label={label} title={label} className="shrink-0">
      {emoji}
    </span>
  )
}

/** Aperçu du résultat sélectionné, avec l'extrait surligné (§2.9). */
function Apercu({
  result,
  onOpen,
}: {
  result: main.SearchResult | null
  onOpen: (r: main.SearchResult) => void
}) {
  if (!result) {
    return <p className="p-4 text-sm text-neutral-500">Sélectionne un résultat.</p>
  }
  return (
    <div className="flex h-full flex-col gap-3 p-3">
      <h2 className="text-sm font-semibold">{result.title || 'Sans titre'}</h2>
      <p className="text-xs text-neutral-500">
        Projet : {result.projectName}
        {result.score > 0 ? <> · Similarité {result.score.toFixed(3)}</> : null}
      </p>

      <div className="min-h-0 flex-1 overflow-auto rounded border border-neutral-200 bg-neutral-50 p-2 text-sm">
        {result.snippet ? (
          <p>
            {result.snippet.leadingEllipsis ? '…' : ''}
            {result.snippet.parts.map((part, i) =>
              // Le domaine a marqué les fragments correspondants ; le frontend
              // se contente de les rendre en surbrillance jaune (§2.9).
              part.match ? (
                <mark key={i} className="bg-yellow-200">
                  {part.text}
                </mark>
              ) : (
                <span key={i}>{part.text}</span>
              ),
            )}
            {result.snippet.trailingEllipsis ? '…' : ''}
          </p>
        ) : (
          <span className="text-neutral-400">Aucun extrait.</span>
        )}
      </div>

      <button
        type="button"
        onClick={() => onOpen(result)}
        className="self-start rounded bg-[var(--color-selection)] px-3 py-1 text-sm text-white"
      >
        Ouvrir
      </button>
    </div>
  )
}
