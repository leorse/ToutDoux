import * as wails from '../wailsjs/go/main/App'
import { demoBackend } from './demoBackend'

/**
 * Contrat du backend, dérivé des méthodes Go exposées par Wails.
 *
 * Le type se déduit des bindings générés : il n'y a donc rien à maintenir en
 * parallèle, et ajouter une méthode Go la rend automatiquement typée ici.
 */
export type Backend = typeof wails

/**
 * Indique si l'application tourne dans la WebView Wails.
 *
 * Les bindings générés font `window['go']['main']['App'][...]` **dans le corps**
 * de chaque fonction. Hors de Wails, `window.go` est absent et l'appel lève de
 * façon synchrone — avant même de rendre une promesse, donc sans qu'aucun
 * `.catch()` puisse l'intercepter. C'est ce qui faisait planter toute
 * l'application au lieu d'afficher une erreur.
 */
function dansWails(): boolean {
  return typeof window !== 'undefined' && 'go' in window
}

let impl: Backend = dansWails() ? wails : (demoBackend as unknown as Backend)

/**
 * Remplace le backend. Réservé aux tests, qui injectent leurs propres doubles.
 */
export function setBackend(next: Partial<Backend>): void {
  impl = { ...impl, ...next } as Backend
}

/**
 * Rétablit le backend réel (ou la démo hors Wails). À appeler entre deux tests.
 */
export function resetBackend(): void {
  impl = dansWails() ? wails : (demoBackend as unknown as Backend)
}

/**
 * S'abonne à un événement émis par le backend (§2.10).
 *
 * Comme les bindings, `window.runtime` n'existe que dans la WebView Wails.
 * Hors de là — navigateur, jsdom — la fonction ne fait rien et rend un
 * désabonnement inerte, plutôt que de faire planter le composant qui s'abonne.
 */
export function onEvent(nom: string, handler: (payload: never) => void): () => void {
  const runtime = (window as unknown as { runtime?: Record<string, unknown> }).runtime
  const abonner = runtime?.EventsOn as
    | ((n: string, h: (p: never) => void) => () => void)
    | undefined
  if (typeof abonner !== 'function') return () => {}
  return abonner(nom, handler)
}

/**
 * Point d'accès unique au backend.
 *
 * Les composants passent par ici et n'importent jamais `wailsjs` directement.
 * C'est ce qui permet à l'interface de s'afficher — et donc d'être testée et
 * capturée — en dehors de l'application compilée (§3.10).
 *
 * Le proxy relaie au backend courant à chaque appel, et non à l'import : sans
 * cela, un test qui remplace le backend après le premier rendu n'aurait aucun
 * effet.
 */
export const api: Backend = new Proxy({} as Backend, {
  get(_target, prop: string) {
    return (...args: unknown[]) => {
      const fn = (impl as unknown as Record<string, (...a: unknown[]) => unknown>)[prop]
      if (typeof fn !== 'function') {
        return Promise.reject(new Error(`Méthode backend inconnue : ${prop}`))
      }
      // L'appel est enveloppé pour qu'une exception synchrone devienne une
      // promesse rejetée : les appelants n'ont ainsi qu'un seul mode d'échec
      // à traiter.
      try {
        return Promise.resolve(fn(...args))
      } catch (err) {
        return Promise.reject(err)
      }
    }
  },
})
