import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import App from './App'
import { setBackend } from './api'

/**
 * Recherche sémantique et Préférences (§2.12, §2.13).
 *
 * Ce que ces tests verrouillent, ce sont les règles d'interface : l'opt-in
 * visible, la séparation stricte des deux modes de recherche, la distinction
 * entre index vide et absence de résultat, et le fait qu'aucun écran ne tente
 * quoi que ce soit sur le réseau.
 *
 * Ce qu'ils ne testent pas, et ne peuvent pas tester : la qualité sémantique du
 * modèle. Elle relève du modèle lui-même et se vérifie à la main (§3.10).
 */

/** Ouvre l'onglet Notes du projet sélectionné. */
async function allerDansNotes(user: ReturnType<typeof userEvent.setup>) {
  await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())
  await user.click(screen.getByRole('tab', { name: 'Notes' }))
  return screen.findByRole('list', { name: 'Notes' })
}

describe('Opt-in sémantique (§2.12)', () => {
  it('le bouton est présent sur une note et bascule à la demande', async () => {
    const user = userEvent.setup()
    render(<App />)
    await allerDansNotes(user)

    const bouton = await screen.findByRole('button', { name: /Ajouter à la recherche sémantique/ })
    expect(bouton).toHaveAttribute('aria-pressed', 'false')

    await user.click(bouton)
    const actif = await screen.findByRole('button', { name: /Dans la recherche sémantique/ })
    expect(actif).toHaveAttribute('aria-pressed', 'true')

    // Le retrait est possible : l'opt-in se reprend.
    await user.click(actif)
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Ajouter à la recherche sémantique/ })).toBeInTheDocument(),
    )
  })

  it('le bouton est présent sur une tâche', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())

    // Le projet verrouillé n'a pas de tâche visible sous les filtres par
    // défaut du jeu de démonstration : on passe sur un projet qui en a.
    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    const arbre = await screen.findByRole('tree', { name: 'Tâches' })
    await user.click(await within(arbre).findByText('Préparer la réunion de cadrage'))

    expect(
      await screen.findByRole('button', { name: /Ajouter à la recherche sémantique/ }),
    ).toBeInTheDocument()
  })
})

describe('Bascule et résultats sémantiques (§2.12)', () => {
  it('seules les entités ajoutées remontent, et le mode est signalé', async () => {
    const user = userEvent.setup()
    render(<App />)
    await allerDansNotes(user)

    // La première note du projet Divers est « Divers » ; on prend celle du
    // projet Migration, qui porte du texte.
    await user.click(await screen.findByRole('button', { name: /Migration 2026/ }))
    const liste = await screen.findByRole('list', { name: 'Notes' })
    await user.click(within(liste).getByRole('button', { name: 'Compte rendu du 12/08' }))
    await user.click(await screen.findByRole('button', { name: /Ajouter à la recherche sémantique/ }))
    await screen.findByRole('button', { name: /Dans la recherche sémantique/ })

    await user.click(screen.getByRole('button', { name: 'Recherche sémantique' }))
    await user.type(screen.getByRole('searchbox', { name: /Rechercher/ }), 'échéance repoussée')

    // Le mode est visible : sans cela, un utilisateur ne saurait pas pourquoi
    // ses résultats habituels ont disparu.
    expect(await screen.findByText(/sémantique/)).toBeInTheDocument()

    const resultats = await screen.findByRole('list', { name: 'Résultats' })
    const lignes = within(resultats).getAllByRole('listitem')
    // La note « Points en suspens » et l'instance de réunion contiennent aussi
    // ces mots, mais n'ont pas été ajoutées à l'index.
    expect(lignes).toHaveLength(1)
    expect(within(resultats).getByText('Compte rendu du 12/08')).toBeInTheDocument()

    // Le score est affiché brut, à trois décimales : c'est ce qui explique
    // l'ordre de la liste, qui n'est plus chronologique (§2.12).
    expect(within(resultats).getByText(/^\d\.\d{3}$/)).toBeInTheDocument()
  })

  it('le score n’apparaît pas en recherche mot-clé', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())

    await user.type(screen.getByRole('searchbox', { name: /Rechercher/ }), 'échéance')

    // La pertinence FTS5 n'est pas exposée : afficher un 0.000 partout ferait
    // croire à une mesure alors qu'il n'y en a pas.
    const resultats = await screen.findByRole('list', { name: 'Résultats' })
    expect(within(resultats).queryByText(/^\d\.\d{3}$/)).not.toBeInTheDocument()
  })

  it('un index vide se dit, il ne s’affiche pas comme « aucun résultat »', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())

    await user.click(screen.getByRole('button', { name: 'Recherche sémantique' }))
    await user.type(screen.getByRole('searchbox', { name: /Rechercher/ }), 'échéance')

    // La nuance n'est pas cosmétique : elle sépare « rien ne correspond » de
    // « rien n'a été indexé ».
    expect(await screen.findByText(/index sémantique est vide/i)).toBeInTheDocument()
  })

  it('les deux modes ne sont jamais fusionnés', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())

    // Index sémantique vide, mais la recherche mot-clé trouve : la bascule doit
    // donc changer la liste du tout au tout.
    await user.type(screen.getByRole('searchbox', { name: /Rechercher/ }), 'échéance')
    expect(await screen.findByRole('list', { name: 'Résultats' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Recherche sémantique' }))
    await waitFor(() =>
      expect(screen.queryByRole('list', { name: 'Résultats' })).not.toBeInTheDocument(),
    )
  })
})

describe('Modèle absent (§2.12)', () => {
  it('la bascule ouvre une fenêtre donnant chemin, fichiers et lien', async () => {
    const user = userEvent.setup()
    setBackend({
      GetSemanticStatus: async () => ({ modelAvailable: false, filesPresent: false, indexedCount: 0 }),
      GetModelStatus: async () => ({
        available: false,
        directory: 'C:\\Donnees\\ToutDoux\\models',
        expectedFiles: ['model.onnx', 'tokenizer.json', 'sentencepiece.bpe.model'],
        missingFiles: ['model.onnx', 'tokenizer.json', 'sentencepiece.bpe.model'],
        downloadUrl: 'https://exemple.invalid/modele',
        files: [
          { name: 'model.onnx', source: 'onnx/model_int8.onnx', present: false, size: 0 },
          { name: 'tokenizer.json', source: 'tokenizer.json', present: false, size: 0 },
          { name: 'sentencepiece.bpe.model', source: 'sentencepiece.bpe.model', present: false, size: 0 },
        ],
        runtime: { name: 'onnxruntime.dll', source: 'onnxruntime-win-x64.zip', present: true, size: 16_000_000 },
      }),
    } as never)

    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: 'Recherche sémantique' }))

    const fenetre = await screen.findByRole('dialog')
    expect(within(fenetre).getByText('C:\\Donnees\\ToutDoux\\models')).toBeInTheDocument()
    expect(within(fenetre).getByText('https://exemple.invalid/modele')).toBeInTheDocument()
    const fichiers = within(fenetre).getByRole('list', { name: 'Fichiers du modèle' })
    expect(within(fichiers).getAllByRole('listitem')).toHaveLength(3)
    expect(within(fenetre).getByRole('button', { name: 'Réessayer' })).toBeInTheDocument()

    // Zéro réseau (§6) : le lien est du texte copiable, jamais une ancre
    // cliquable qui déclencherait une navigation sortante.
    expect(within(fenetre).queryByRole('link')).not.toBeInTheDocument()

    // Le moteur d'inférence figure au même endroit que le modèle : voir une
    // seule moitié du problème conduirait à retélécharger 120 Mo pour rien.
    expect(within(fenetre).getByText(/Moteur d’inférence/)).toBeInTheDocument()
    expect(within(fenetre).getByText('onnxruntime.dll')).toBeInTheDocument()
  })

  it('distingue « modèle manquant » de « moteur manquant »', async () => {
    const user = userEvent.setup()
    setBackend({
      GetSemanticStatus: async () => ({ modelAvailable: false, filesPresent: true, indexedCount: 0 }),
      GetModelStatus: async () => ({
        available: true,
        directory: 'C:\\Donnees\\ToutDoux\\models',
        expectedFiles: [],
        missingFiles: [],
        downloadUrl: 'https://exemple.invalid/modele',
        files: [],
        runtime: { name: 'onnxruntime.dll', source: 'onnxruntime-win-x64.zip', present: false, size: 0 },
      }),
    } as never)

    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: 'Recherche sémantique' }))

    const fenetre = await screen.findByRole('dialog')
    expect(within(fenetre).getByText(/fichiers du modèle sont bien déposés/i)).toBeInTheDocument()
  })

  it('la bascule reste inactive tant que le modèle manque', async () => {
    const user = userEvent.setup()
    setBackend({
      GetSemanticStatus: async () => ({ modelAvailable: false, filesPresent: false, indexedCount: 0 }),
    } as never)

    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())

    const bascule = screen.getByRole('button', { name: 'Recherche sémantique' })
    await user.click(bascule)
    await screen.findByRole('dialog')

    // Basculer d'abord pour n'afficher qu'une erreur ensuite laisserait
    // l'interface dans un mode incapable de rien rendre.
    expect(bascule).toHaveAttribute('aria-pressed', 'false')
  })
})

describe('Vue Préférences (§2.13)', () => {
  it('affiche en permanence le chemin, les fichiers et le lien', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => expect(screen.getByRole('navigation', { name: 'Projets' })).toBeInTheDocument())

    await user.click(screen.getByRole('button', { name: 'Préférences' }))

    // Aucune condition d'erreur à reproduire : l'écran est une référence.
    expect(await screen.findByText('Modèle sémantique')).toBeInTheDocument()
    expect(screen.getByRole('list', { name: 'Fichiers du modèle' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Vérifier' })).toBeInTheDocument()
    expect(screen.getByText(/aucune requête réseau/i)).toBeInTheDocument()
  })
})
