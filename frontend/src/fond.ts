import type { MouseEvent } from 'react'

/**
 * Attribut qui marque une **ligne** de liste, par opposition au fond.
 *
 * Il est posé sur les lignes réelles — un projet, une tâche, une réunion — et
 * jamais sur les messages d'état vide, qui font partie du fond : double-cliquer
 * sur « Aucune réunion, double-clic pour en créer une » doit précisément créer
 * une réunion.
 */
export const ATTRIBUT_LIGNE = 'data-ligne'

/**
 * Dit si un événement de souris a eu lieu sur le **fond** d'une liste, et non
 * sur l'une de ses lignes.
 *
 * Le double-clic sur le fond crée un élément (§2.1, §2.2, §2.7). Sans ce
 * filtre, double-cliquer sur une ligne — pour l'ouvrir, ou simplement en
 * cliquant vite deux fois — créerait un élément par mégarde, ce qui est
 * exactement le genre de geste qu'on ne pardonne pas à une application.
 *
 * Le test remonte les ancêtres plutôt que de comparer `event.target` à
 * `event.currentTarget` : une ligne contient des cases à cocher, des libellés et
 * des icônes, et le clic peut atterrir sur n'importe lequel d'entre eux.
 */
export function surLeFond(event: MouseEvent): boolean {
  const cible = event.target as HTMLElement | null
  return !cible?.closest(`[${ATTRIBUT_LIGNE}]`)
}
