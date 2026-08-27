import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { dateInstance } from './dates'

/**
 * Ajouts d'ergonomie : compteur de priorités, format de date des instances,
 * recherche sur les noms de projets, création au double-clic sur le fond.
 */

afterEach(() => vi.unstubAllGlobals())

async function attendreLApplication() {
  await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())
}

describe('Compteur de priorités (§2.8)', () => {
  it('l’onglet annonce le nombre de tâches, sans compter deux fois', async () => {
    render(<App />)
    await attendreLApplication()

    // Le jeu de démonstration porte des tâches critiques datées, qui figurent
    // dans les deux sections : si le compteur les additionnait, il dépasserait
    // le nombre de lignes distinctes.
    const onglet = await screen.findByRole('tab', { name: /Priorités \(\d+\)/ })
    const annonce = Number(onglet.textContent?.match(/\((\d+)\)/)?.[1])

    const user = userEvent.setup()
    await user.click(onglet)

    const critiques = await screen.findByRole('list', { name: /Tâches Critiques/ })
    const echeances = await screen.findByRole('list', { name: /Échéances à venir/ })
    const ids = new Set(
      [...within(critiques).getAllByRole('listitem'), ...within(echeances).getAllByRole('listitem')].map(
        (li) => li.textContent,
      ),
    )

    expect(annonce).toBe(ids.size)
    // Et le total additionné est bien plus grand : le dédoublonnage sert.
    const additionne =
      within(critiques).getAllByRole('listitem').length + within(echeances).getAllByRole('listitem').length
    expect(additionne).toBeGreaterThan(annonce)
  })
})

describe('Format des dates d’instance (§2.7)', () => {
  it('rend « 25 Août 2026 17:43 »', () => {
    expect(dateInstance('2026-08-25T17:43:00')).toBe('25 Août 2026 17:43')
    // Un jour à un chiffre n'est pas complété, une heure si.
    expect(dateInstance('2026-01-03T09:05:00')).toBe('3 Janvier 2026 09:05')
  })

  it('rend une chaîne vide sur une date illisible', () => {
    // Mieux qu'« Invalid Date » au milieu d'une liste française.
    expect(dateInstance('pas une date')).toBe('')
  })

  it('la liste des instances utilise ce format', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    await user.click(screen.getByRole('tab', { name: 'Réunions' }))

    const instances = await screen.findByRole('list', { name: 'Instances' })
    const premiere = within(instances).getAllByRole('listitem')[0]
    expect(premiere.textContent).toMatch(
      /^\d{1,2} (Janvier|Février|Mars|Avril|Mai|Juin|Juillet|Août|Septembre|Octobre|Novembre|Décembre) \d{4} \d{2}:\d{2}$/,
    )
  })
})

describe('Recherche sur les noms de projets (§2.9)', () => {
  it('un projet remonte dans les résultats', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    await user.type(screen.getByRole('searchbox', { name: /Rechercher/ }), 'Migration')

    const resultats = await screen.findByRole('list', { name: 'Résultats' })
    // Le nom apparaît deux fois sur la ligne : comme titre, et comme étiquette
    // de projet — les deux sont le même texte pour un projet.
    expect(within(resultats).getAllByText('Migration 2026').length).toBeGreaterThan(0)
    // L'icône dossier distingue un projet d'une note ou d'une réunion.
    expect(within(resultats).getByLabelText('Projet')).toBeInTheDocument()
  })

  it('ouvrir un projet trouvé le sélectionne', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    await user.type(screen.getByRole('searchbox', { name: /Rechercher/ }), 'Refonte')
    const resultats = await screen.findByRole('list', { name: 'Résultats' })
    await user.dblClick(within(resultats).getAllByText('Refonte du portail')[0])

    // Retour sur la vue Projets, le projet trouvé sélectionné.
    const sidebar = await screen.findByRole('navigation', { name: 'Projets' })
    await waitFor(() =>
      expect(within(sidebar).getByRole('button', { name: /Refonte du portail/ })).toHaveAttribute(
        'aria-current',
        'true',
      ),
    )
  })
})

describe('Création au double-clic sur le fond (§2.1, §2.2, §2.7)', () => {
  it('sur le fond de la liste des projets', async () => {
    const user = userEvent.setup()
    vi.stubGlobal('prompt', () => 'Projet créé au double-clic')

    render(<App />)
    const sidebar = await screen.findByRole('navigation', { name: 'Projets' })

    await user.dblClick(sidebar)

    await waitFor(() =>
      expect(within(sidebar).getByRole('button', { name: /Projet créé au double-clic/ })).toBeInTheDocument(),
    )
  })

  it('sur le fond de l’arbre des tâches', async () => {
    const user = userEvent.setup()
    vi.stubGlobal('prompt', () => 'Tâche créée au double-clic')

    render(<App />)
    await attendreLApplication()
    const arbre = await screen.findByRole('tree', { name: 'Tâches' })

    // Le fond est le conteneur de l'arbre, pas l'arbre lui-même.
    await user.dblClick(arbre.parentElement!)

    await waitFor(() => expect(screen.getByText('Tâche créée au double-clic')).toBeInTheDocument())
  })

  it('sur le fond de la liste des réunions', async () => {
    const user = userEvent.setup()
    vi.stubGlobal('prompt', () => 'Réunion créée au double-clic')

    render(<App />)
    await attendreLApplication()
    await user.click(screen.getByRole('tab', { name: 'Réunions' }))

    const liste = await screen.findByRole('list', { name: 'Réunions' })
    await user.dblClick(liste)

    await waitFor(() =>
      expect(within(liste).getByRole('button', { name: 'Réunion créée au double-clic' })).toBeInTheDocument(),
    )
  })

  it('sur le fond de la liste des notes', async () => {
    const user = userEvent.setup()
    vi.stubGlobal('prompt', () => 'Note créée au double-clic')

    render(<App />)
    await attendreLApplication()
    await user.click(screen.getByRole('tab', { name: 'Notes' }))

    const liste = await screen.findByRole('list', { name: 'Notes' })
    await user.dblClick(liste)

    await waitFor(() =>
      expect(within(liste).getByRole('button', { name: 'Note créée au double-clic' })).toBeInTheDocument(),
    )
  })

  it('sur le fond de la liste des instances', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    await user.click(screen.getByRole('tab', { name: 'Réunions' }))

    const liste = await screen.findByRole('list', { name: 'Instances' })
    const avant = within(liste).getAllByRole('listitem').length

    await user.dblClick(liste)

    await waitFor(() => expect(within(liste).getAllByRole('listitem')).toHaveLength(avant + 1))
  })

  it('sans réunion sélectionnée, le fond des instances ne crée rien', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    // « Transverse / Divers » n'a aucune réunion : rien à quoi rattacher une
    // instance, donc le double-clic ne doit rien faire (§2.7).
    await user.click(screen.getByRole('tab', { name: 'Réunions' }))
    const liste = await screen.findByRole('list', { name: 'Instances' })

    await user.dblClick(liste)

    await waitFor(() => expect(within(liste).queryAllByRole('listitem')).toHaveLength(0))
  })

  it('un double-clic sur une ligne ne crée rien', async () => {
    const user = userEvent.setup()
    // Le double-clic sur un projet sert à le renommer : s'il créait aussi, le
    // moindre clic un peu vif ajouterait un projet.
    vi.stubGlobal('prompt', () => null)

    render(<App />)
    const sidebar = await screen.findByRole('navigation', { name: 'Projets' })
    const avant = within(sidebar).getAllByRole('button').length

    await user.dblClick(within(sidebar).getByRole('button', { name: /Migration 2026/ }))

    await waitFor(() => expect(within(sidebar).getAllByRole('button')).toHaveLength(avant))
  })
})
