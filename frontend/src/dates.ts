/**
 * Mois en toutes lettres, avec une majuscule.
 *
 * La typographie française met normalement les mois en minuscule. La majuscule
 * est ici un **choix d'affichage assumé** : ces dates servent d'étiquettes dans
 * une liste étroite, où la capitale aide à repérer le mois d'un coup d'œil.
 * Ne pas « corriger » sans demander.
 */
const MOIS = [
  'Janvier',
  'Février',
  'Mars',
  'Avril',
  'Mai',
  'Juin',
  'Juillet',
  'Août',
  'Septembre',
  'Octobre',
  'Novembre',
  'Décembre',
]

/**
 * Formate la date d'une instance de réunion : « 25 Août 2026 17:43 » (§2.7).
 *
 * Écrit à la main plutôt qu'avec `toLocaleString('fr-FR')` : le format long de
 * l'API intercale un « à » entre la date et l'heure — « 25 août 2026 à 17:43 » —
 * qui n'est pas voulu ici, et met le mois en minuscule.
 *
 * Rend une chaîne vide sur une date illisible, plutôt que « Invalid Date » :
 * une étiquette vide se remarque moins qu'un message d'erreur anglais au milieu
 * d'une liste.
 */
export function dateInstance(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''

  const deuxChiffres = (n: number) => String(n).padStart(2, '0')
  return (
    `${d.getDate()} ${MOIS[d.getMonth()]} ${d.getFullYear()} ` +
    `${deuxChiffres(d.getHours())}:${deuxChiffres(d.getMinutes())}`
  )
}
