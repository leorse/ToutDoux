import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'

/**
 * Vues transverses et recherche (§2.7, §2.8, §2.9).
 */

afterEach(() => vi.unstubAllGlobals())

describe('Vue Priorités (§2.8)', () => {
  it('est pleine largeur, sans sidebar projets', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())

    await user.click(screen.getByRole('tab', { name: /Priorités/ }))

    // La sidebar disparaît : la vue balaie tous les projets par nature.
    expect(screen.queryByRole('navigation', { name: 'Projets' })).not.toBeInTheDocument()
    await waitFor(() =>
      expect(screen.getByRole('list', { name: /Tâches Critiques/ })).toBeInTheDocument(),
    )
    expect(screen.getByRole('list', { name: /Échéances à venir/ })).toBeInTheDocument()
  })

  it('un simple clic affiche le détail en lecture seule', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(screen.getByRole('tab', { name: /Priorités/ }))

    const section = await screen.findByRole('list', { name: /Tâches Critiques/ })
    await user.click(within(section).getAllByRole('button')[0])

    // Le panneau reprend la structure du détail Tâches, mais sans champ éditable.
    expect(await screen.findByRole('button', { name: 'Ouvrir dans Tâches' })).toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
  })

  it('un double-clic bascule sur la vue Projets et ouvre la tâche', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(screen.getByRole('tab', { name: /Priorités/ }))

    const section = await screen.findByRole('list', { name: /Tâches Critiques/ })
    await user.dblClick(within(section).getAllByRole('button')[0])

    // Retour sur Projets, avec la sidebar et l'arbre.
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())
    expect(screen.getByRole('tab', { name: 'Projets' })).toHaveAttribute('aria-selected', 'true')
    await waitFor(() => expect(screen.getByRole('tree', { name: 'Tâches' })).toBeInTheDocument())
  })
})

describe('Onglet Réunions (§2.7)', () => {
  it('présente trois colonnes séparées par deux poignées', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    await user.click(screen.getByRole('tab', { name: 'Réunions' }))

    await waitFor(() => expect(screen.getByRole('list', { name: 'Réunions' })).toBeInTheDocument())
    expect(screen.getByRole('list', { name: 'Instances' })).toBeInTheDocument()

    // Sidebar↔contenu, réunions↔instances, instances↔éditeur : trois poignées,
    // toutes verticales.
    const orientations = screen.getAllByRole('separator').map((s) => s.getAttribute('aria-orientation'))
    expect(orientations).toEqual(['vertical', 'vertical', 'vertical'])
  })

  it('sélectionner une réunion sélectionne sa dernière instance', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    await user.click(screen.getByRole('tab', { name: 'Réunions' }))

    const instances = await screen.findByRole('list', { name: 'Instances' })
    await waitFor(() => expect(within(instances).getAllByRole('button').length).toBe(2))

    // Deux instances au jeu de démonstration. La plus récente doit être
    // sélectionnée d'office, sans clic supplémentaire (§2.7) : son compte rendu
    // s'affiche donc dans l'éditeur, et non le message « aucune instance ».
    await waitFor(() =>
      expect(screen.getByText(/Échéance repoussée, revoir le périmètre/)).toBeInTheDocument(),
    )
    expect(screen.queryByText('Aucune instance sélectionnée.')).not.toBeInTheDocument()
  })

  // Sans réunion sélectionnée, le clic droit sur la colonne Instances ne doit
  // afficher aucun menu — précision de la v2 (§2.7).
  it('le clic droit sur les instances ne fait rien sans réunion sélectionnée', async () => {
    const user = userEvent.setup()
    render(<App />)
    // « Refonte du portail » n'a aucune réunion.
    await user.click(await screen.findByRole('button', { name: /Refonte du portail/ }))
    await user.click(screen.getByRole('tab', { name: 'Réunions' }))

    const instances = await screen.findByRole('list', { name: 'Instances' })
    fireEvent.contextMenu(instances)
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })
})

describe('Recherche globale (§2.9)', () => {
  it('remplace tout le contenu principal, quelle que soit la vue active', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => expect(screen.getByRole('tree', { name: 'Tâches' })).toBeInTheDocument())

    await user.type(screen.getByRole('searchbox'), 'réunion')

    await waitFor(() => expect(screen.getByRole('list', { name: 'Résultats' })).toBeInTheDocument())
    // L'arbre et la sidebar cèdent la place aux résultats.
    expect(screen.queryByRole('tree', { name: 'Tâches' })).not.toBeInTheDocument()
    expect(screen.queryByRole('navigation', { name: 'Projets' })).not.toBeInTheDocument()
  })

  it('ne se déclenche qu’au-delà de deux caractères', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.type(screen.getByRole('searchbox'), 'ré')

    expect(screen.getByText('3 caractères minimum')).toBeInTheDocument()
    expect(screen.queryByRole('list', { name: 'Résultats' })).not.toBeInTheDocument()
  })

  it('surligne le terme cherché dans l’extrait', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.type(screen.getByRole('searchbox'), 'réunion')

    const resultats = await screen.findByRole('list', { name: 'Résultats' })
    await user.click(within(resultats).getAllByRole('button')[0])

    const marque = await waitFor(() => document.querySelector('mark'))
    expect(marque).not.toBeNull()
    expect(marque!.textContent?.toLowerCase()).toBe('réunion')
  })

  it('Échap ferme la recherche et rend la vue précédente', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.type(screen.getByRole('searchbox'), 'réunion')
    await waitFor(() => expect(screen.getByRole('list', { name: 'Résultats' })).toBeInTheDocument())

    await user.keyboard('{Escape}')

    await waitFor(() => expect(screen.getByRole('tree', { name: 'Tâches' })).toBeInTheDocument())
    expect(screen.queryByRole('list', { name: 'Résultats' })).not.toBeInTheDocument()
  })

  it('le bouton Fermer rend aussi la vue précédente', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.type(screen.getByRole('searchbox'), 'réunion')
    await user.click(await screen.findByRole('button', { name: /Fermer/ }))

    await waitFor(() => expect(screen.getByRole('tree', { name: 'Tâches' })).toBeInTheDocument())
  })

  it('double-clic sur un résultat bascule sur la vue Projets', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.type(screen.getByRole('searchbox'), 'réunion')

    const resultats = await screen.findByRole('list', { name: 'Résultats' })
    await user.dblClick(within(resultats).getAllByRole('button')[0])

    await waitFor(() => expect(screen.queryByRole('list', { name: 'Résultats' })).not.toBeInTheDocument())
    expect(screen.getByRole('tab', { name: 'Projets' })).toHaveAttribute('aria-selected', 'true')
  })
})
