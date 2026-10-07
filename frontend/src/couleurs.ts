/**
 * Palette des notes et des réunions (v1.2.0) : dix pastels, plus l'absence de
 * couleur.
 *
 * Le backend ne connaît que les clés, et refuse toute autre valeur ; la teinte
 * de chaque clé se règle ici, sans toucher aux données enregistrées. Tous les
 * tons sont assez clairs pour que le texte sombre des listes reste lisible.
 */
export const COULEURS = [
  { cle: 'rose', nom: 'Rose', hex: '#fbcfe8' },
  { cle: 'peche', nom: 'Pêche', hex: '#fed7aa' },
  { cle: 'jaune', nom: 'Jaune', hex: '#fef08a' },
  { cle: 'anis', nom: 'Anis', hex: '#d9f99d' },
  { cle: 'menthe', nom: 'Menthe', hex: '#bbf7d0' },
  { cle: 'turquoise', nom: 'Turquoise', hex: '#99f6e4' },
  { cle: 'ciel', nom: 'Ciel', hex: '#bae6fd' },
  { cle: 'lavande', nom: 'Lavande', hex: '#c7d2fe' },
  { cle: 'lilas', nom: 'Lilas', hex: '#e9d5ff' },
  { cle: 'sable', nom: 'Sable', hex: '#e8dcc8' },
] as const

/**
 * Teinte d'une clé de la palette. Rien pour l'absence de couleur, ni pour une
 * clé inconnue : l'élément s'affiche alors comme un élément sans couleur.
 */
export function teinte(cle: string | null | undefined): string | undefined {
  return COULEURS.find((c) => c.cle === cle)?.hex
}
