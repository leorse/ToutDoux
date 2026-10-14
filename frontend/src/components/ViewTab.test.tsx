import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ViewTab } from './ViewTab'

describe('ViewTab', () => {
  it('sans onClose, n’affiche aucune croix', () => {
    render(<ViewTab label="Projets" active onClick={() => {}} />)
    expect(screen.getByRole('tab', { name: 'Projets' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('la croix appelle onClose sans sélectionner l’onglet', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    const onClose = vi.fn()
    render(<ViewTab label="Recherche" active={false} onClick={onClick} onClose={onClose} closeLabel="Fermer la recherche" />)

    const croix = screen.getByRole('button', { name: 'Fermer la recherche' })
    expect(croix).toHaveTextContent('✕')
    await user.click(croix)

    expect(onClose).toHaveBeenCalledOnce()
    expect(onClick).not.toHaveBeenCalled()
  })

  it('l’onglet fermable garde son libellé comme nom et reste cliquable', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    render(<ViewTab label="Recherche" active={false} onClick={onClick} onClose={() => {}} />)
    await user.click(screen.getByRole('tab', { name: 'Recherche' }))
    expect(onClick).toHaveBeenCalledOnce()
  })
})
