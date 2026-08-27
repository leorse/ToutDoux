import { useCallback, useEffect, useState } from 'react'
import { api } from '../api'
import type { main, model } from '../../wailsjs/go/models'

/**
 * Fenêtre « modèle absent » (§2.12).
 *
 * Elle ne s'affiche qu'au moment où l'utilisateur demande quelque chose que le
 * modèle seul peut faire. Elle donne tout ce qu'il faut pour agir : le chemin
 * exact, les noms de fichiers exacts, le lien, et un bouton Réessayer.
 *
 * **Aucune tentative automatique.** Ni au montage, ni sur Réessayer : le bouton
 * relit le disque, rien d'autre. L'application n'accède jamais au réseau (§6),
 * et c'est ici que la tentation serait la plus grande.
 */
export function ModelMissingDialog({
  onClose,
  onAvailable,
}: {
  onClose: () => void
  /** Appelé quand la recherche sémantique devient réellement utilisable. */
  onAvailable?: () => void
}) {
  const { fichiers, semantique, verifier } = useModelStatus(onAvailable)

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Recherche sémantique indisponible"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
    >
      <div className="max-h-full w-full max-w-2xl overflow-auto rounded border border-neutral-300 bg-white p-4 shadow-lg">
        <h2 className="text-base font-semibold">La recherche sémantique n’est pas disponible</h2>
        <Explication fichiers={fichiers} semantique={semantique} />
        <ModelDetails fichiers={fichiers} />

        <div className="mt-4 flex items-center gap-2">
          <button
            type="button"
            onClick={verifier}
            className="rounded bg-[var(--color-selection)] px-3 py-1 text-sm text-white"
          >
            Réessayer
          </button>
          <button
            type="button"
            onClick={onClose}
            className="rounded border border-neutral-300 px-3 py-1 text-sm hover:bg-neutral-100"
          >
            Fermer
          </button>
        </div>
      </div>
    </div>
  )
}

/**
 * Lit les deux statuts et les tient à jour.
 *
 * Ils sont distincts et le restent : les fichiers peuvent être déposés sans que
 * la vectorisation soit possible. Les confondre enverrait l'utilisateur
 * retélécharger 120 Mo qu'il a déjà.
 */
export function useModelStatus(onAvailable?: () => void) {
  const [fichiers, setFichiers] = useState<model.Status | null>(null)
  const [semantique, setSemantique] = useState<main.SemanticStatus | null>(null)

  const verifier = useCallback(() => {
    api.GetModelStatus().then(setFichiers, () => setFichiers(null))
    api.GetSemanticStatus().then(
      (s) => {
        setSemantique(s)
        if (s.modelAvailable) onAvailable?.()
      },
      () => setSemantique(null),
    )
  }, [onAvailable])

  useEffect(verifier, [verifier])
  return { fichiers, semantique, verifier }
}

/** Dit quoi faire, selon ce qui manque réellement. */
function Explication({
  fichiers,
  semantique,
}: {
  fichiers: model.Status | null
  semantique: main.SemanticStatus | null
}) {
  if (semantique?.modelAvailable) {
    return <p className="mt-2 text-sm text-green-700">La recherche sémantique est prête.</p>
  }

  // Modèle déposé mais moteur absent : ce n'est pas le dépôt qui est en cause,
  // et le dire évite de retélécharger 120 Mo pour rien.
  if (fichiers?.available && !fichiers.runtime.present) {
    return (
      <p className="mt-2 text-sm text-neutral-700">
        Les trois fichiers du modèle sont bien déposés. Il manque{' '}
        <strong>{fichiers.runtime.name}</strong>, la bibliothèque qui exécute le modèle. Elle est
        normalement livrée avec l’application, à côté de <code>toutdoux.exe</code>.
      </p>
    )
  }

  return (
    <p className="mt-2 text-sm text-neutral-700">
      La recherche sémantique a besoin d’un modèle de langue d’environ 120 Mo. Il n’est pas fourni
      avec l’application et <strong>n’est jamais téléchargé automatiquement</strong> : Tout Doux
      n’accède pas à Internet. Récupère les fichiers depuis un poste qui a accès au réseau, puis
      dépose-les dans le dossier ci-dessous.
    </p>
  )
}

/**
 * Détail des fichiers attendus, partagé par la fenêtre du §2.12 et la vue
 * Préférences du §2.13.
 *
 * C'est le pendant côté interface du choix d'une seule commande
 * `GetModelStatus` côté backend : une seule source, un seul rendu, donc aucun
 * risque que les deux écrans se contredisent.
 */
export function ModelDetails({ fichiers }: { fichiers: model.Status | null }) {
  const [copie, setCopie] = useState<string | null>(null)

  if (!fichiers) {
    return <p className="mt-3 text-sm text-neutral-500">Statut du modèle indisponible.</p>
  }

  function copier(valeur: string) {
    // La copie est un confort : si le presse-papiers est refusé, le texte reste
    // sélectionnable à la main, et l'écran n'a pas besoin de le signaler.
    navigator.clipboard?.writeText(valeur).then(
      () => setCopie(valeur),
      () => setCopie(null),
    )
  }

  return (
    <div className="mt-3 flex flex-col gap-3">
      <Champ label="Dossier attendu" valeur={fichiers.directory} copie={copie} onCopier={copier} />

      <div>
        <p className="text-xs font-medium text-neutral-600">Fichiers attendus</p>
        <ul aria-label="Fichiers du modèle" className="mt-1 flex flex-col gap-1">
          {fichiers.files.map((f) => (
            <li key={f.name} className="flex items-baseline gap-2 text-sm">
              <span aria-hidden>{f.present ? '✅' : '❌'}</span>
              <code className="rounded bg-neutral-100 px-1">{f.name}</code>
              <span className="text-xs text-neutral-500">
                {f.present ? `${Math.round(f.size / 1024)} Ko` : `à prendre dans ${f.source}`}
              </span>
            </li>
          ))}
        </ul>
        {/* Le seul fichier dont le nom change entre la source et la
            destination : le dépôt publie plusieurs quantifications. */}
        <p className="mt-1 text-xs text-neutral-500">
          Le fichier <code>model_int8.onnx</code> doit être renommé en <code>model.onnx</code>{' '}
          après téléchargement.
        </p>
      </div>

      {/* Le runtime ne vit pas dans le dossier des modèles : c'est une
          bibliothèque d'exécution livrée avec l'application (§3.5). Il figure
          ici parce qu'un utilisateur qui ne voit que la moitié du problème
          retéléchargera le modèle pour rien. */}
      <div>
        <p className="text-xs font-medium text-neutral-600">Moteur d’inférence</p>
        <p className="mt-1 flex items-baseline gap-2 text-sm">
          <span aria-hidden>{fichiers.runtime.present ? '✅' : '❌'}</span>
          <code className="rounded bg-neutral-100 px-1">{fichiers.runtime.name}</code>
          <span className="text-xs text-neutral-500">
            {fichiers.runtime.present
              ? `${Math.round(fichiers.runtime.size / 1024 / 1024)} Mo — livré avec l’application`
              : `à prendre dans ${fichiers.runtime.source}`}
          </span>
        </p>
      </div>

      <Champ
        label="Lien de téléchargement — à ouvrir depuis un poste connecté"
        valeur={fichiers.downloadUrl}
        copie={copie}
        onCopier={copier}
      />
    </div>
  )
}

/**
 * Valeur copiable.
 *
 * Le lien est affiché en texte et non en `<a href>` : un clic dans la WebView
 * déclencherait une navigation sortante, ce que le §6 interdit. L'utilisateur
 * copie et ouvre où il veut.
 */
function Champ({
  label,
  valeur,
  copie,
  onCopier,
}: {
  label: string
  valeur: string
  copie: string | null
  onCopier: (v: string) => void
}) {
  return (
    <div>
      <p className="text-xs font-medium text-neutral-600">{label}</p>
      <div className="mt-1 flex items-center gap-2">
        <code
          className="min-w-0 flex-1 truncate rounded bg-neutral-100 px-2 py-1 text-xs"
          title={valeur}
        >
          {valeur}
        </code>
        <button
          type="button"
          onClick={() => onCopier(valeur)}
          className="shrink-0 rounded border border-neutral-300 px-2 py-1 text-xs hover:bg-neutral-100"
        >
          {copie === valeur ? 'Copié' : 'Copier'}
        </button>
      </div>
    </div>
  )
}
