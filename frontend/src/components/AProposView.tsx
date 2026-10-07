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

/** Rubriques d'une version, dans l'ordre d'affichage : les ajouts d'abord. */
const RUBRIQUES = [
  { type: 'ajout', titre: 'Ajouts' },
  { type: 'correction', titre: 'Corrections' },
]

/** Changements d'une nature donnée, les majeurs d'abord, l'ordre du fichier conservé pour le reste. */
function changements(release: main.Release, type: string): main.Change[] {
  const lignes = release.changes.filter((c) => c.type === type)
  return [...lignes.filter((c) => c.level === 'majeur'), ...lignes.filter((c) => c.level !== 'majeur')]
}

/**
 * Onglet « À propos » des Préférences : la version courante et l'historique des
 * versions, de la plus récente à la plus ancienne. Les données viennent du
 * backend, qui les lit dans wails.json et releases.json.
 *
 * Chaque version sépare ses ajouts de ses corrections ; dans chaque rubrique,
 * les changements majeurs passent devant et portent une étiquette « Majeur ».
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

            {RUBRIQUES.map(({ type, titre }) => {
              const lignes = changements(release, type)
              if (lignes.length === 0) return null
              return (
                <div key={type}>
                  <h5 className="mt-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">{titre}</h5>
                  <ul aria-label={titre} className="mt-1 list-disc pl-5 text-xs text-neutral-700">
                    {lignes.map((c, i) => (
                      <li key={i} className={c.level === 'majeur' ? 'font-medium text-neutral-900' : undefined}>
                        {c.level === 'majeur' ? (
                          <span className="mr-1.5 rounded bg-neutral-800 px-1 py-px text-[10px] font-semibold uppercase text-white">
                            Majeur
                          </span>
                        ) : null}
                        {formaterTexte(c.text)}
                      </li>
                    ))}
                  </ul>
                </div>
              )
            })}
          </article>
        ))}
      </section>
    </div>
  )
}
