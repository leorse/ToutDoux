import { useState } from 'react'
import { AProposView } from './AProposView'
import { ModelDetails, useModelStatus } from './ModelMissingDialog'
import { ViewTab } from './ViewTab'

type Onglet = 'preferences' | 'apropos'

/**
 * Vue Préférences (§2.13).
 *
 * Troisième vue de premier niveau, accessible par l'icône ⚙️ de la bande
 * transverse. Elle affiche exactement les mêmes informations que la fenêtre du
 * §2.12, mais de façon permanente : on doit pouvoir y revenir des semaines plus
 * tard pour retrouver le chemin ou le lien, sans avoir à reproduire la
 * condition d'erreur qui déclenche la fenêtre.
 *
 * Deux onglets : « Paramétrage », affiché par défaut, et « À propos » (version
 * et historique des versions).
 */
export function PreferencesView() {
  const { fichiers, semantique, verifier } = useModelStatus()
  const [onglet, setOnglet] = useState<Onglet>('preferences')

  return (
    <div className="flex h-full flex-col">
      <div role="tablist" aria-label="Onglets des préférences" className="flex gap-1 border-b border-neutral-300 px-2 py-1">
        <ViewTab label="Paramétrage" active={onglet === 'preferences'} onClick={() => setOnglet('preferences')} />
        <ViewTab label="À propos" active={onglet === 'apropos'} onClick={() => setOnglet('apropos')} />
      </div>

      <div className="min-h-0 flex-1 overflow-auto p-4">
        <div className="mx-auto max-w-2xl">
          {onglet === 'apropos' ? (
            <AProposView />
          ) : (
            <div className="flex flex-col gap-4">
              <section className="rounded border border-neutral-300 p-3">
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-medium">Modèle sémantique</h3>
                  <Etat pret={semantique?.modelAvailable} fichiersPresents={fichiers?.available} />
                  <button
                    type="button"
                    onClick={verifier}
                    className="ml-auto rounded border border-neutral-300 px-2 py-1 text-xs hover:bg-neutral-100"
                  >
                    Vérifier
                  </button>
                </div>

                <p className="mt-2 text-xs text-neutral-600">
                  La recherche sémantique retrouve un contenu reformulé — « client en colère » sur une
                  recherche « client mécontent » — et tolère les fautes de frappe. Elle est optionnelle :
                  tout le reste de Tout Doux fonctionne sans elle.
                </p>

                <ModelDetails fichiers={fichiers} />

                <p className="mt-3 text-xs text-neutral-500">
                  {semantique
                    ? `${semantique.indexedCount} élément${semantique.indexedCount > 1 ? 's' : ''} dans l’index sémantique.`
                    : null}{' '}
                  Rien n’y entre automatiquement : chaque note, instance de réunion ou tâche doit être
                  ajoutée à la main par son bouton 🧠.
                </p>
              </section>

              <section className="rounded border border-neutral-300 p-3">
                <h3 className="text-sm font-medium">Confidentialité</h3>
                <p className="mt-2 text-xs text-neutral-600">
                  Tout Doux n’émet <strong>aucune requête réseau</strong> : pas de téléchargement, pas de
                  vérification de mise à jour, pas de télémétrie. Les données restent dans le dossier
                  indiqué ci-dessus, sur ce poste.
                </p>
              </section>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

/** Pastille d'état, qui distingue « fichiers absents » de « moteur absent ». */
function Etat({ pret, fichiersPresents }: { pret?: boolean; fichiersPresents?: boolean }) {
  if (pret) {
    return <span className="rounded bg-green-100 px-2 py-0.5 text-xs text-green-800">Prêt</span>
  }
  if (fichiersPresents) {
    return (
      <span className="rounded bg-amber-100 px-2 py-0.5 text-xs text-amber-800">
        Fichiers déposés, moteur non intégré
      </span>
    )
  }
  return <span className="rounded bg-neutral-200 px-2 py-0.5 text-xs text-neutral-700">Absent</span>
}
