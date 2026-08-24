import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'

/**
 * Mise à jour en direct des agrégats de la sidebar (§2.1).
 *
 * Les compteurs et la couleur d'un projet dérivent de ses tâches et de ses
 * notes. Ils étaient chargés une seule fois, au montage : une tâche ajoutée ou
 * une criticité modifiée ne se voyaient qu'après un aller-retour par l'onglet
 * Priorités, qui démontait puis remontait la sidebar. Ces tests fixent la
 * correction — toute écriture rafraîchit la sidebar sur-le-champ.
 */

afterEach(() => vi.unstubAllGlobals())

/** Rend la ligne de sidebar du projet nommé. */
function ligneProjet(nom: string | RegExp): HTMLElement {
  return screen.getByRole('button', { name: new RegExp(typeof nom === 'string' ? nom : nom.source) })
}

describe('Sidebar en direct', () => {
  it('§2.1 — ajouter une tâche incrémente le compteur sans changer de vue', async () => {
    vi.stubGlobal('prompt', () => 'Tâche ajoutée par le test')
    render(<App />)

    // « Transverse / Divers » porte une tâche active dans le jeu de démonstration.
    await waitFor(() => expect(ligneProjet('Transverse')).toHaveTextContent('1'))
    await waitFor(() => expect(screen.getByRole('tree', { name: 'Tâches' })).toBeInTheDocument())

    // Création par le clic droit sur le fond de l'arbre, seul chemin en v2 (§2.2).
    fireEvent.contextMenu(screen.getByRole('tree', { name: 'Tâches' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: '+ Nouvelle tâche' }))

    // Le compteur doit suivre immédiatement, sans passage par Priorités.
    await waitFor(() => expect(ligneProjet('Transverse')).toHaveTextContent('2'))
  })

  it('§2.1 — passer une tâche en Critique recolore le projet sans changer de vue', async () => {
    render(<App />)

    // « Migration 2026 » a déjà une critique ; on travaille sur « Refonte du
    // portail », qui n'en a pas et démarre donc sans fond rouge.
    const refonte = await screen.findByRole('button', { name: /Refonte du portail/ })
    expect(refonte.className).not.toContain('color-critique')

    fireEvent.click(refonte)
    await waitFor(() => expect(screen.getByText('Maquette de la page d’accueil')).toBeInTheDocument())

    fireEvent.click(screen.getByText('Maquette de la page d’accueil'))
    fireEvent.click(await screen.findByRole('button', { name: 'Critique', pressed: false }))

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Refonte du portail/ }).className).toContain(
        'color-critique',
      ),
    )
  })

  it('§2.1 — créer une note incrémente le compteur de notes', async () => {
    vi.stubGlobal('prompt', () => 'Note ajoutée par le test')
    render(<App />)

    await waitFor(() => expect(ligneProjet('Transverse')).toHaveTextContent('1 notes'))

    fireEvent.click(screen.getByRole('tab', { name: 'Notes' }))
    const liste = await screen.findByRole('list', { name: 'Notes' })
    fireEvent.contextMenu(liste)
    fireEvent.click(await screen.findByRole('menuitem', { name: '+ Nouvelle note' }))

    await waitFor(() => expect(ligneProjet('Transverse')).toHaveTextContent('2 notes'))
  })

  it('§2.1 — supprimer une tâche décrémente le compteur', async () => {
    vi.stubGlobal('confirm', () => true)
    render(<App />)

    await waitFor(() => expect(ligneProjet('Transverse')).toHaveTextContent('1'))
    const arbre = await screen.findByRole('tree', { name: 'Tâches' })

    const tache = within(arbre).getByText('Commander le matériel')
    fireEvent.contextMenu(tache)
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Supprimer' }))

    await waitFor(() => expect(ligneProjet('Transverse')).toHaveTextContent('0'))
  })
})
