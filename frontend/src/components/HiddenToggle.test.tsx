import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { HiddenToggle } from './HiddenToggle'

describe('HiddenToggle', () => {
  it('affiche « Afficher les éléments cachés » quand les cachés sont masqués', () => {
    render(<HiddenToggle montrerCaches={false} onToggle={() => {}} />)
    const bouton = screen.getByRole('button', { name: 'Afficher les éléments cachés' })
    expect(bouton).toHaveAttribute('aria-pressed', 'false')
  })

  it('affiche « Masquer les éléments cachés » quand les cachés sont montrés', () => {
    render(<HiddenToggle montrerCaches={true} onToggle={() => {}} />)
    const bouton = screen.getByRole('button', { name: 'Masquer les éléments cachés' })
    expect(bouton).toHaveAttribute('aria-pressed', 'true')
  })

  it('appelle onToggle au clic', async () => {
    const user = userEvent.setup()
    const onToggle = vi.fn()
    render(<HiddenToggle montrerCaches={false} onToggle={onToggle} />)
    await user.click(screen.getByRole('button'))
    expect(onToggle).toHaveBeenCalledOnce()
  })
})
