import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import App from './App'
import { setBackend } from './api'
import { AProposView } from './components/AProposView'

/**
 * Onglet « À propos » des Préférences : version courante, historique des
 * versions et sous-onglets de la vue.
 */

const INFO = {
  version: '2.3.4',
  releases: [
    {
      version: '2.3.4',
      date: '2026-09-25',
      important: 'Sauvegardez **avant** de mettre à jour',
      changes: [
        { type: 'ajout', level: 'mineur', text: 'Première modification' },
        { type: 'ajout', level: 'mineur', text: 'Une *deuxième* modification' },
        { type: 'ajout', level: 'mineur', text: 'Une **troisième** modification' },
      ],
    },
    { version: '2.0.0', date: '2026-01-05', changes: [{ type: 'correction', level: 'mineur', text: 'Ancienne modification' }] },
  ],
}

describe('AProposView', () => {
  it('affiche la version courante', async () => {
    setBackend({ GetAppInfo: async () => INFO as never })
    render(<AProposView />)
    expect(await screen.findByText('2.3.4', { selector: 'strong' })).toBeInTheDocument()
  })

  it('liste les versions de la plus récente à la plus ancienne, avec leurs modifications en puces', async () => {
    setBackend({ GetAppInfo: async () => INFO as never })
    render(<AProposView />)

    const articles = await screen.findAllByRole('article')
    expect(articles).toHaveLength(2)
    expect(within(articles[0]).getByRole('heading', { name: /Version 2\.3\.4/ })).toBeInTheDocument()
    expect(within(articles[1]).getByRole('heading', { name: /Version 2\.0\.0/ })).toBeInTheDocument()

    expect(within(articles[0]).getAllByRole('listitem')).toHaveLength(3)
    expect(within(articles[1]).getAllByRole('listitem')).toHaveLength(1)
    expect(within(articles[0]).getByText('25 septembre 2026')).toBeInTheDocument()
  })

  it('met en forme le gras et l’italique sans afficher les marqueurs', async () => {
    setBackend({ GetAppInfo: async () => INFO as never })
    render(<AProposView />)

    const articles = await screen.findAllByRole('article')
    const items = within(articles[0]).getAllByRole('listitem')
    expect(items[1].querySelector('em')?.textContent).toBe('deuxième')
    expect(items[2].querySelector('strong')?.textContent).toBe('troisième')
    expect(articles[0].textContent).not.toContain('*')
  })

  it('met en avant l’information importante, uniquement quand elle existe', async () => {
    setBackend({ GetAppInfo: async () => INFO as never })
    render(<AProposView />)

    const articles = await screen.findAllByRole('article')
    const note = within(articles[0]).getByRole('note')
    expect(note).toHaveTextContent('Sauvegardez avant de mettre à jour')
    expect(note.querySelector('strong + strong')?.textContent).toBe('avant')
    expect(within(articles[1]).queryByRole('note')).toBeNull()
  })

  it('n’interprète pas le HTML d’un texte', async () => {
    setBackend({
      GetAppInfo: async () =>
        ({ version: '1.0.0', releases: [{ version: '1.0.0', date: '2026-01-01', changes: [{ type: 'ajout', level: 'mineur', text: '<b>x</b>' }] }] }) as never,
    })
    render(<AProposView />)

    const article = await screen.findByRole('article')
    expect(article.querySelector('b')).toBeNull()
    expect(article).toHaveTextContent('<b>x</b>')
  })

  it('sépare les ajouts des corrections, les ajouts d’abord', async () => {
    setBackend({
      GetAppInfo: async () =>
        ({
          version: '3.0.0',
          releases: [
            {
              version: '3.0.0',
              date: '2026-10-07',
              changes: [
                { type: 'correction', level: 'mineur', text: 'Un bogue' },
                { type: 'ajout', level: 'mineur', text: 'Une nouveauté' },
                { type: 'ajout', level: 'mineur', text: 'Une autre nouveauté' },
              ],
            },
          ],
        }) as never,
    })
    render(<AProposView />)

    const article = await screen.findByRole('article')
    const rubriques = within(article).getAllByRole('heading', { level: 5 }).map((h) => h.textContent)
    expect(rubriques).toEqual(['Ajouts', 'Corrections'])
    expect(within(within(article).getByRole('list', { name: 'Ajouts' })).getAllByRole('listitem')).toHaveLength(2)
    const corrections = within(within(article).getByRole('list', { name: 'Corrections' })).getAllByRole('listitem')
    expect(corrections.map((li) => li.textContent)).toEqual(['Un bogue'])
  })

  it('n’affiche pas de rubrique « Ajouts » pour une version de corrections seules', async () => {
    setBackend({ GetAppInfo: async () => INFO as never })
    render(<AProposView />)

    const articles = await screen.findAllByRole('article')
    expect(within(articles[1]).getByRole('heading', { name: 'Corrections' })).toBeInTheDocument()
    expect(within(articles[1]).queryByRole('heading', { name: 'Ajouts' })).toBeNull()
    expect(within(articles[0]).queryByRole('heading', { name: 'Corrections' })).toBeNull()
  })

  it('place les changements majeurs en premier et les signale comme tels', async () => {
    setBackend({
      GetAppInfo: async () =>
        ({
          version: '3.0.0',
          releases: [
            {
              version: '3.0.0',
              date: '2026-10-07',
              changes: [
                { type: 'ajout', level: 'mineur', text: 'Petit ajout' },
                { type: 'ajout', level: 'majeur', text: 'Grand ajout' },
              ],
            },
          ],
        }) as never,
    })
    render(<AProposView />)

    const items = within(await screen.findByRole('list', { name: 'Ajouts' })).getAllByRole('listitem')
    expect(items[0]).toHaveTextContent('Grand ajout')
    // L'étiquette est un texte, pas seulement une couleur.
    expect(within(items[0]).getByText('Majeur')).toBeInTheDocument()
    expect(items[1]).toHaveTextContent('Petit ajout')
    expect(within(items[1]).queryByText('Majeur')).toBeNull()
  })

  it('signale une information indisponible sans planter', async () => {
    setBackend({ GetAppInfo: async () => Promise.reject(new Error('KO')) })
    render(<AProposView />)
    expect(await screen.findByText(/indisponibles/)).toBeInTheDocument()
  })
})

describe('Sous-onglets des Préférences', () => {
  async function ouvrirPreferences(user: ReturnType<typeof userEvent.setup>) {
    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: 'Préférences' }))
  }

  it('affiche « Paramétrage » par défaut, avec son contenu inchangé', async () => {
    const user = userEvent.setup()
    await ouvrirPreferences(user)

    const onglets = screen.getByRole('tablist', { name: 'Onglets des préférences' })
    expect(within(onglets).getByRole('tab', { name: 'Paramétrage' })).toHaveAttribute('aria-selected', 'true')
    expect(within(onglets).getByRole('tab', { name: 'À propos' })).toHaveAttribute('aria-selected', 'false')
    expect(await screen.findByText('Modèle sémantique')).toBeInTheDocument()
    expect(screen.getByText(/aucune requête réseau/i)).toBeInTheDocument()
    expect(screen.queryByRole('article')).toBeNull()
  })

  it('bascule sur « À propos » puis revient', async () => {
    const user = userEvent.setup()
    await ouvrirPreferences(user)

    await user.click(screen.getByRole('tab', { name: 'À propos' }))
    expect(await screen.findByRole('region', { name: 'Historique des versions' })).toBeInTheDocument()
    expect(screen.queryByText('Modèle sémantique')).toBeNull()

    await user.click(screen.getByRole('tab', { name: 'Paramétrage' }))
    expect(await screen.findByText('Modèle sémantique')).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Historique des versions' })).toBeNull()
  })
})
