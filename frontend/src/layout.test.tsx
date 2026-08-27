import { render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import App from './App'

/**
 * Contrat de disposition (§2.5, §2.6, §2.7, §2.11).
 *
 * Ces tests ne valident pas que la disposition est *la bonne* au sens produit —
 * ça, c'est la spec qui le dit, et une spec peut se tromper. Ils garantissent
 * qu'une disposition une fois arrêtée ne change plus à l'insu de tout le monde
 * au fil des itérations.
 *
 * jsdom ne calcule pas de mise en page : tous les rectangles y valent zéro, et
 * comparer des coordonnées à l'écran n'a donc aucun sens ici. On vérifie à la
 * place la structure qui *produit* la disposition — l'orientation déclarée du
 * séparateur — qui est justement ce qu'une régression modifierait.
 */

function orientationsDesSeparateurs(): string[] {
  return screen
    .getAllByRole('separator')
    .map((s) => s.getAttribute('aria-orientation') ?? '')
}

describe('Bande transverse', () => {
  it('§2.11 — porte les deux vues de premier niveau et la recherche', () => {
    render(<App />)
    expect(screen.getByRole('tab', { name: 'Projets' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /Priorités/ })).toBeInTheDocument()
    expect(screen.getByRole('searchbox', { name: /rechercher dans les tâches/i })).toBeInTheDocument()
  })

  it('§2.11 — les trois sous-onglets du projet sont présents', () => {
    render(<App />)
    for (const nom of ['Tâches', 'Notes', 'Réunions']) {
      expect(screen.getByRole('tab', { name: nom })).toBeInTheDocument()
    }
  })
})

describe('Onglet Tâches', () => {
  it('§2.5 — l’arbre et le détail sont côte à côte, jamais empilés', async () => {
    render(<App />)
    await waitFor(() => expect(screen.getByRole('tree', { name: 'Tâches' })).toBeInTheDocument())

    // Deux séparateurs sur cet écran : sidebar↔contenu et arbre↔détail. Un
    // « horizontal » ici signalerait un retour à des panneaux empilés.
    expect(orientationsDesSeparateurs()).toEqual(['vertical', 'vertical'])
  })

  it('§2.4 — les filtres tiennent sur une ligne, en icônes et pastilles', async () => {
    render(<App />)
    await waitFor(() => expect(screen.getByRole('group', { name: 'Filtres des tâches' })).toBeInTheDocument())

    // Statut : trois entrées multi-sélectionnables, dont seule « Actives » est
    // active par défaut.
    expect(screen.getByRole('button', { name: 'Actives' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Complétées' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Annulées' })).toHaveAttribute('aria-pressed', 'false')

    // Importance : les quatre pastilles, toutes actives par défaut.
    for (const imp of ['Critique', 'Haute', 'Normale', 'Basse']) {
      expect(screen.getByRole('button', { name: imp })).toHaveAttribute('aria-pressed', 'true')
    }

    expect(screen.getByRole('button', { name: 'Avec échéance uniquement' })).toHaveAttribute(
      'aria-pressed',
      'false',
    )
  })
})

describe('Onglet Notes', () => {
  it('§2.6 — la liste et l’éditeur sont côte à côte', async () => {
    const { default: userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('tab', { name: 'Notes' }))
    await waitFor(() => expect(screen.getByRole('list', { name: 'Notes' })).toBeInTheDocument())
    expect(orientationsDesSeparateurs()).toEqual(['vertical', 'vertical'])
  })
})
