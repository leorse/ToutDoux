import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'
import { resetBackend } from './api'
import { resetDemoData } from './demoBackend'

// jsdom ne fait aucune mise en page : ni `getClientRects` sur les Range, ni
// `elementFromPoint`. ProseMirror s'en sert pour faire défiler jusqu'au curseur
// après un `focus()` ; sans ces bouchons, tout test qui pilote l'éditeur lève.
const rectVide = () => ({ x: 0, y: 0, top: 0, left: 0, right: 0, bottom: 0, width: 0, height: 0, toJSON: () => ({}) }) as DOMRect
for (const proto of [Range.prototype, Element.prototype]) {
  if (!proto.getClientRects) proto.getClientRects = () => [] as unknown as DOMRectList
  if (!proto.getBoundingClientRect) proto.getBoundingClientRect = rectVide
}
if (!document.elementFromPoint) document.elementFromPoint = () => null

// Chaque test repart d'un DOM vide, du backend par défaut et du jeu de
// démonstration intact. Sans la remise à zéro des données, un test qui crée une
// tâche fausserait les compteurs attendus par le suivant.
afterEach(() => {
  cleanup()
  resetBackend()
  resetDemoData()
})
