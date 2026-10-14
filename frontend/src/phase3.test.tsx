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
  it('s’affiche dans l’onglet Recherche, à la place du contenu principal', async () => {
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

/** Lance une recherche et attend la liste de ses résultats. */
async function chercher(user: ReturnType<typeof userEvent.setup>, q = 'réunion') {
  await user.type(screen.getByRole('searchbox'), q)
  return screen.findByRole('list', { name: 'Résultats' })
}

/** Ouvre le premier résultat et attend la bascule sur la vue Projets. */
async function ouvrirPremier(user: ReturnType<typeof userEvent.setup>, resultats: HTMLElement) {
  await user.dblClick(within(resultats).getAllByRole('button')[0])
  await waitFor(() => expect(screen.getByRole('tab', { name: 'Projets' })).toHaveAttribute('aria-selected', 'true'))
}

describe('Onglet Recherche', () => {
  it('n’existe pas sans recherche', async () => {
    render(<App />)
    await waitFor(() => expect(screen.getByRole('tree', { name: 'Tâches' })).toBeInTheDocument())
    expect(screen.queryByRole('tab', { name: 'Recherche' })).not.toBeInTheDocument()
  })

  it('apparaît après Priorités et devient la vue active dès la recherche lancée', async () => {
    const user = userEvent.setup()
    render(<App />)
    await chercher(user)

    const onglets = screen.getAllByRole('tab').map((t) => t.textContent)
    expect(onglets.slice(0, 3)).toEqual(['Projets', expect.stringMatching(/^Priorités/), 'Recherche'])
    expect(screen.getByRole('tab', { name: 'Recherche' })).toHaveAttribute('aria-selected', 'true')
  })

  it('disparaît quand la requête repasse sous trois caractères, et rend la vue d’avant', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(screen.getByRole('tab', { name: /Priorités/ }))
    await chercher(user)

    await user.type(screen.getByRole('searchbox'), '{Backspace}{Backspace}{Backspace}{Backspace}{Backspace}')

    expect(screen.getByRole('searchbox')).toHaveValue('ré')
    expect(screen.queryByRole('tab', { name: 'Recherche' })).not.toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /Priorités/ })).toHaveAttribute('aria-selected', 'true')
  })

  it('n’a plus de bouton Fermer dans l’en-tête : seulement une croix sur l’onglet', async () => {
    const user = userEvent.setup()
    render(<App />)
    await chercher(user)

    expect(screen.queryByRole('button', { name: /Fermer ⊗/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Fermer la recherche' })).toHaveTextContent(/^✕$/)
  })

  it('ouvrir une note garde la requête et l’onglet', async () => {
    const user = userEvent.setup()
    render(<App />)
    await ouvrirPremier(user, await chercher(user, 'repoussée'))

    await waitFor(() => expect(screen.getByRole('tab', { name: 'Notes' })).toHaveAttribute('aria-selected', 'true'))
    expect(screen.getByRole('searchbox')).toHaveValue('repoussée')
    expect(screen.getByRole('tab', { name: 'Recherche' })).toHaveAttribute('aria-selected', 'false')
  })

  it('revenir sur l’onglet retrouve le résultat sélectionné', async () => {
    const user = userEvent.setup()
    render(<App />)
    const resultats = await chercher(user)
    const boutons = within(resultats).getAllByRole('button')
    expect(boutons.length).toBeGreaterThan(1)
    const dernier = boutons[boutons.length - 1]
    const libelle = dernier.textContent

    await user.dblClick(dernier)
    await waitFor(() => expect(screen.getByRole('tab', { name: 'Projets' })).toHaveAttribute('aria-selected', 'true'))

    await user.click(screen.getByRole('tab', { name: 'Recherche' }))
    const liste = await screen.findByRole('list', { name: 'Résultats' })
    const courant = within(liste)
      .getAllByRole('button')
      .find((b) => b.getAttribute('aria-current') === 'true')
    expect(courant?.textContent).toBe(libelle)
  })

  it('passer sur Priorités garde l’onglet', async () => {
    const user = userEvent.setup()
    render(<App />)
    await chercher(user)
    await user.click(screen.getByRole('tab', { name: /Priorités/ }))

    await waitFor(() => expect(screen.getByRole('list', { name: /Tâches Critiques/ })).toBeInTheDocument())
    expect(screen.getByRole('tab', { name: 'Recherche' })).toBeInTheDocument()
  })

  it('une nouvelle requête remplace la recherche dans un seul onglet', async () => {
    const user = userEvent.setup()
    render(<App />)
    await ouvrirPremier(user, await chercher(user))

    await user.clear(screen.getByRole('searchbox'))
    await user.type(screen.getByRole('searchbox'), 'repoussée')

    expect(screen.getAllByRole('tab', { name: 'Recherche' })).toHaveLength(1)
    expect(screen.getByRole('tab', { name: 'Recherche' })).toHaveAttribute('aria-selected', 'true')
    const liste = await screen.findByRole('list', { name: 'Résultats' })
    await waitFor(() => expect(within(liste).getByText('Compte rendu du 12/08')).toBeInTheDocument())
  })

  it('la croix de l’onglet ferme la recherche et rend la vue d’avant', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(screen.getByRole('tab', { name: /Priorités/ }))
    await chercher(user)

    await user.click(screen.getByRole('button', { name: 'Fermer la recherche' }))

    expect(screen.queryByRole('tab', { name: 'Recherche' })).not.toBeInTheDocument()
    expect(screen.getByRole('searchbox')).toHaveValue('')
    expect(screen.getByRole('tab', { name: /Priorités/ })).toHaveAttribute('aria-selected', 'true')
  })

  it('fermer depuis Projets laisse Projets et la note ouverte', async () => {
    const user = userEvent.setup()
    render(<App />)
    await ouvrirPremier(user, await chercher(user, 'repoussée'))
    await waitFor(() => expect(screen.getByRole('tab', { name: 'Notes' })).toHaveAttribute('aria-selected', 'true'))

    await user.click(screen.getByRole('button', { name: 'Fermer la recherche' }))

    expect(screen.queryByRole('tab', { name: 'Recherche' })).not.toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Projets' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: 'Notes' })).toHaveAttribute('aria-selected', 'true')
  })

  it('Échap sur l’onglet Recherche ferme la recherche et rend la vue d’avant', async () => {
    const user = userEvent.setup()
    render(<App />)
    await chercher(user)

    await user.keyboard('{Escape}')

    await waitFor(() => expect(screen.getByRole('tree', { name: 'Tâches' })).toBeInTheDocument())
    expect(screen.queryByRole('tab', { name: 'Recherche' })).not.toBeInTheDocument()
  })

  it('Échap ailleurs ne ferme pas la recherche', async () => {
    const user = userEvent.setup()
    render(<App />)
    await ouvrirPremier(user, await chercher(user))

    fireEvent.keyDown(window, { key: 'Escape' })

    expect(screen.getByRole('searchbox')).toHaveValue('réunion')
    expect(screen.getByRole('tab', { name: 'Recherche' })).toBeInTheDocument()
  })
})
