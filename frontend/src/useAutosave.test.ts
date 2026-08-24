import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAutosave } from './useAutosave'

/**
 * Sauvegarde automatique (§2.6, §2.7).
 *
 * La v1 perdait des frappes quand l'utilisateur cliquait ailleurs avant la fin
 * du délai. Ces tests fixent la correction de la v2 : le délai reste, mais tout
 * ce qui est en attente part immédiatement au moindre changement de contexte.
 */

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('useAutosave', () => {
  it('écrit après le délai d’inactivité', () => {
    const save = vi.fn()
    const { result } = renderHook(() => useAutosave(save, 2000))

    act(() => result.current.schedule('bonjour'))
    expect(save).not.toHaveBeenCalled()

    act(() => vi.advanceTimersByTime(2000))
    expect(save).toHaveBeenCalledExactlyOnceWith('bonjour')
  })

  it('repousse le délai à chaque frappe plutôt que d’écrire en rafale', () => {
    const save = vi.fn()
    const { result } = renderHook(() => useAutosave(save, 2000))

    act(() => result.current.schedule('a'))
    act(() => vi.advanceTimersByTime(1500))
    act(() => result.current.schedule('ab'))
    act(() => vi.advanceTimersByTime(1500))
    expect(save).not.toHaveBeenCalled()

    act(() => vi.advanceTimersByTime(500))
    expect(save).toHaveBeenCalledExactlyOnceWith('ab')
  })

  // Le cœur de la correction v2 : perte de focus, changement de sélection ou
  // de projet écrivent sur-le-champ, sans attendre le délai.
  it('écrit immédiatement au vidage, sans attendre le délai', () => {
    const save = vi.fn()
    const { result } = renderHook(() => useAutosave(save, 2000))

    act(() => result.current.schedule('frappe en cours'))
    act(() => result.current.flush())

    expect(save).toHaveBeenCalledExactlyOnceWith('frappe en cours')

    // Le délai armé ne doit pas produire une seconde écriture derrière.
    act(() => vi.advanceTimersByTime(2000))
    expect(save).toHaveBeenCalledOnce()
  })

  it('un vidage sans rien en attente n’écrit pas', () => {
    const save = vi.fn()
    const { result } = renderHook(() => useAutosave(save, 2000))

    act(() => result.current.flush())
    act(() => result.current.flush())

    // Sans cette garde, chaque clic dans l'application déclencherait une
    // écriture inutile en base.
    expect(save).not.toHaveBeenCalled()
  })

  it('écrit ce qui est en attente au démontage', () => {
    const save = vi.fn()
    const { result, unmount } = renderHook(() => useAutosave(save, 2000))

    act(() => result.current.schedule('non sauvegardé'))
    unmount()

    // C'est le cas que la v1 perdait : changer de note ou de projet démonte le
    // composant avant l'expiration du délai.
    expect(save).toHaveBeenCalledExactlyOnceWith('non sauvegardé')
  })

  it('utilise la dernière fonction de sauvegarde, pas celle capturée au montage', () => {
    const premier = vi.fn()
    const second = vi.fn()
    const { result, rerender } = renderHook(({ fn }) => useAutosave(fn, 2000), {
      initialProps: { fn: premier },
    })

    rerender({ fn: second })
    act(() => result.current.schedule('valeur'))
    act(() => vi.advanceTimersByTime(2000))

    expect(premier).not.toHaveBeenCalled()
    expect(second).toHaveBeenCalledExactlyOnceWith('valeur')
  })
})
