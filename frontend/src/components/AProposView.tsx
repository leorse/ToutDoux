import { useEffect, useState } from 'react'
import { api } from '../api'
import { formaterTexte } from '../texteEnrichi'
import type { main } from '../../wailsjs/go/models'

/** « 2026-09-25 » → « 25 septembre 2026 ». UTC : la date n'a pas d'heure, un fuseau ne doit pas la décaler. */
function dateLongue(iso: string): string {
  const d = new Date(`${iso}T00:00:00Z`)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })
}

/**
 * Onglet « À propos » des Préférences : la version courante et l'historique des
 * versions, de la plus récente à la plus ancienne. Les données viennent du
 * backend, qui les lit dans wails.json et releases.json.
 */
export function AProposView() {
  const [info, setInfo] = useState<main.AppInfo | null>(null)
  const [indisponible, setIndisponible] = useState(false)

  useEffect(() => {
    let annule = false
    api.GetAppInfo().then(
      (i) => {
        if (!annule) setInfo(i)
      },
      () => {
        if (!annule) setIndisponible(true)
      },
    )
    return () => {
      annule = true
    }
  }, [])

  if (indisponible) {
    return <p className="text-sm text-neutral-500">Informations de version indisponibles.</p>
  }
  if (!info) return null

  return (
    <div className="flex flex-col gap-4">
      <section className="rounded border border-neutral-300 p-3">
        <h3 className="text-sm font-medium">Tout Doux</h3>
        <p className="mt-1 text-xs text-neutral-600">
          Version <strong>{info.version}</strong>
        </p>
      </section>

      <section aria-label="Historique des versions" className="flex flex-col gap-3">
        <h3 className="text-sm font-medium">Historique des versions</h3>
        {info.releases.map((release) => (
          <article key={release.version} className="rounded border border-neutral-300 p-3">
            <h4 className="flex items-baseline gap-2 text-sm font-medium">
              Version {release.version}
              <time dateTime={release.date} className="text-xs font-normal text-neutral-500">
                {dateLongue(release.date)}
              </time>
            </h4>

            {release.important ? (
              <p
                role="note"
                className="mt-2 rounded border-l-4 border-amber-500 bg-amber-50 px-3 py-2 text-xs text-amber-900"
              >
                <strong>Important : </strong>
                {formaterTexte(release.important)}
              </p>
            ) : null}

            <ul className="mt-2 list-disc pl-5 text-xs text-neutral-700">
              {release.changes.map((c, i) => (
                <li key={i}>{formaterTexte(c)}</li>
              ))}
            </ul>
          </article>
        ))}
      </section>
    </div>
  )
}
