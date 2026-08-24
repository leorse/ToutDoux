import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'
import { resetBackend } from './api'
import { resetDemoData } from './demoBackend'

// Chaque test repart d'un DOM vide, du backend par défaut et du jeu de
// démonstration intact. Sans la remise à zéro des données, un test qui crée une
// tâche fausserait les compteurs attendus par le suivant.
afterEach(() => {
  cleanup()
  resetBackend()
  resetDemoData()
})
