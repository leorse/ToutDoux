import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import App from './App'

/**
 * Masquage des projets, notes et réunions (§2.1, §2.6, §2.7).
 *
 * Le jeu de démonstration (demoBackend.ts) porte volontairement un élément
 * caché par défaut dans chacune des trois listes — le projet « Ancien
 * portail », la note « Ancienne version du cahier des charges » et la réunion
 * « Ancien comité, abandonné » — pour que ces tests n'aient rien à préparer.
 */

async function attendreLApplication() {
  await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())
}

async function clicDroit(element: HTMLElement, user: ReturnType<typeof userEvent.setup>) {
  await user.pointer({ keys: '[MouseRight]', target: element })
}

describe('Liste des projets (§2.1)', () => {
  it('cache le projet « Ancien portail » par défaut, et le montre œil ouvert', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    const projets = screen.getByRole('navigation', { name: 'Projets' })
    expect(within(projets).queryByText('Ancien portail')).toBeNull()

    // Le bouton-œil est un frère de <nav>, pas un descendant.
    const oeil = screen.getByRole('button', { name: 'Afficher les éléments cachés' })
    await user.click(oeil)

    const ligne = within(projets).getByText('Ancien portail')
    expect(ligne).toHaveClass('italic')
    expect(ligne.closest('button')).toHaveClass('opacity-50')
    expect(screen.getByRole('button', { name: 'Masquer les éléments cachés' })).toBeInTheDocument()
  })

  it('affiche les compteurs de notes et réunions sous la forme N(J)', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    // Migration 2026 : 2 notes visibles + 1 cachée, 2 réunions visibles + 1 cachée.
    const ligne = await screen.findByRole('button', { name: /Migration 2026/ })
    await waitFor(() => expect(ligne).toHaveTextContent('2(1) notes · 2(1) réunions'))

    // Rester ainsi que l'œil des projets soit ouvert ou fermé (§2.1).
    await user.click(screen.getByRole('button', { name: 'Afficher les éléments cachés' }))
    expect(screen.getByRole('button', { name: /Migration 2026/ })).toHaveTextContent('2(1) notes · 2(1) réunions')
  })

  it('désactive « Cacher » sur le projet verrouillé', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    const ligne = await screen.findByRole('button', { name: /Transverse \/ Divers/ })
    await clicDroit(ligne, user)

    const entree = await screen.findByRole('menuitem', { name: 'Cacher' })
    expect(entree).toBeDisabled()
  })

  it('cacher le projet sélectionné fait retomber la sélection sur le premier projet affiché', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    const ligne = await screen.findByRole('button', { name: /Migration 2026/ })
    await user.click(ligne)
    expect(ligne).toHaveAttribute('aria-current', 'true')

    await clicDroit(ligne, user)
    await user.click(await screen.findByRole('menuitem', { name: 'Cacher' }))

    // Divers, verrouillé, reste en tête et devient la sélection de repli.
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Transverse \/ Divers/ })).toHaveAttribute('aria-current', 'true'),
    )
    expect(screen.queryByRole('button', { name: /Migration 2026/ })).toBeNull()
  })

  it('ouvrir une tâche d’un projet caché depuis les Priorités rouvre l’œil des projets', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    await user.click(await screen.findByRole('tab', { name: /Priorités/ }))
    const critiques = await screen.findByRole('list', { name: /Tâches Critiques/ })
    await user.dblClick(within(critiques).getByText('Tâche du projet caché'))

    await attendreLApplication()
    expect(screen.getByRole('button', { name: 'Masquer les éléments cachés' })).toBeInTheDocument()
    const ligne = await screen.findByRole('button', { name: /Ancien portail/ })
    expect(ligne).toHaveAttribute('aria-current', 'true')
  })
})

describe('Liste des notes (§2.6)', () => {
  async function allerDansNotes(user: ReturnType<typeof userEvent.setup>) {
    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    await user.click(screen.getByRole('tab', { name: 'Notes' }))
    return screen.findByRole('list', { name: 'Notes' })
  }

  it('cache la note par défaut, et la montre œil ouvert avec Cacher/Réafficher au menu', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()
    const liste = await allerDansNotes(user)

    expect(within(liste).queryByText('Ancienne version du cahier des charges')).toBeNull()

    const conteneur = liste.closest('.flex.h-full.flex-col') as HTMLElement
    await user.click(within(conteneur).getByRole('button', { name: 'Afficher les éléments cachés' }))

    const bouton = within(liste).getByText('Ancienne version du cahier des charges').closest('button')!
    expect(bouton).toHaveClass('italic', 'opacity-50')

    await clicDroit(bouton, user)
    expect(await screen.findByRole('menuitem', { name: 'Réafficher' })).toBeInTheDocument()
  })

  it('cacher la note sélectionnée fait retomber la sélection sur la première note affichée', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()
    const liste = await allerDansNotes(user)

    const premiere = within(liste).getAllByRole('button')[0]
    await user.click(premiere)
    await clicDroit(premiere, user)
    await user.click(await screen.findByRole('menuitem', { name: 'Cacher' }))

    await waitFor(() => expect(within(liste).queryByText(premiere.textContent!)).toBeNull())
    // Le titre affiché dans l'éditeur ne doit plus être celui qu'on vient de cacher.
    const titreInput = await screen.findByRole('textbox', { name: 'Titre de la note' })
    expect(titreInput).not.toHaveValue(premiere.textContent)
  })
})

describe('Liste des réunions (§2.7)', () => {
  async function allerDansReunions(user: ReturnType<typeof userEvent.setup>) {
    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    await user.click(screen.getByRole('tab', { name: 'Réunions' }))
    return screen.findByRole('list', { name: 'Réunions' })
  }

  it('cache la réunion par défaut, et la montre œil ouvert', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()
    const liste = await allerDansReunions(user)

    expect(within(liste).queryByText('Ancien comité, abandonné')).toBeNull()

    const conteneur = liste.closest('.flex.h-full.flex-col') as HTMLElement
    await user.click(within(conteneur).getByRole('button', { name: 'Afficher les éléments cachés' }))

    const bouton = within(liste).getByText('Ancien comité, abandonné').closest('button')!
    expect(bouton).toHaveClass('italic', 'opacity-50')
  })

  it('ouvrir une réunion cachée depuis la recherche rouvre l’œil des réunions et la sélectionne', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    const recherche = screen.getByRole('searchbox', { name: 'Rechercher dans les tâches, notes, réunions' })
    await user.type(recherche, 'Ancien comité')

    const resultats = await screen.findByRole('list', { name: 'Résultats' })
    const resultat = within(resultats).getByText(/Ancien comité/).closest('button')!
    await user.dblClick(resultat)

    await attendreLApplication()
    await user.click(screen.getByRole('tab', { name: 'Réunions' }))

    expect(screen.getByRole('button', { name: 'Masquer les éléments cachés' })).toBeInTheDocument()
    const ligne = await screen.findByRole('button', { name: /Ancien comité, abandonné/ })
    expect(ligne).toHaveClass('bg-neutral-200')
  })
})

describe('Indépendance des trois yeux (§2.1, §2.6, §2.7)', () => {
  it('fermer l’œil des notes ne touche pas celui des réunions, et l’état des notes survit à un changement de projet', async () => {
    const user = userEvent.setup()
    render(<App />)
    await attendreLApplication()

    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    await user.click(screen.getByRole('tab', { name: 'Notes' }))
    const listeNotes = await screen.findByRole('list', { name: 'Notes' })
    const conteneurNotes = listeNotes.closest('.flex.h-full.flex-col') as HTMLElement
    await user.click(within(conteneurNotes).getByRole('button', { name: 'Afficher les éléments cachés' }))
    expect(within(conteneurNotes).getByRole('button', { name: 'Masquer les éléments cachés' })).toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: 'Réunions' }))
    const listeReunions = await screen.findByRole('list', { name: 'Réunions' })
    const conteneurReunions = listeReunions.closest('.flex.h-full.flex-col') as HTMLElement
    expect(within(conteneurReunions).getByRole('button', { name: 'Afficher les éléments cachés' })).toBeInTheDocument()

    // Changer de projet et revenir : l'œil des notes doit être resté ouvert.
    await user.click(screen.getByRole('button', { name: /Transverse \/ Divers/ }))
    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    await user.click(screen.getByRole('tab', { name: 'Notes' }))
    const listeNotesApres = await screen.findByRole('list', { name: 'Notes' })
    const conteneurNotesApres = listeNotesApres.closest('.flex.h-full.flex-col') as HTMLElement
    expect(
      within(conteneurNotesApres).getByRole('button', { name: 'Masquer les éléments cachés' }),
    ).toBeInTheDocument()
  })
})
