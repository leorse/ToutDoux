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
      changes: ['Première modification', 'Une *deuxième* modification', 'Une **troisième** modification'],
    },
    { version: '2.0.0', date: '2026-01-05', changes: ['Ancienne modification'] },
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
        ({ version: '1.0.0', releases: [{ version: '1.0.0', date: '2026-01-01', changes: ['<b>x</b>'] }] }) as never,
    })
    render(<AProposView />)

    const article = await screen.findByRole('article')
    expect(article.querySelector('b')).toBeNull()
    expect(article).toHaveTextContent('<b>x</b>')
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
