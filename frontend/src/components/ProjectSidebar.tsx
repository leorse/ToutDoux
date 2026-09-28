import { useEffect, useState } from 'react'
import { api } from '../api'
import { surLeFond } from '../fond'
import type { domain, stats } from '../../wailsjs/go/models'
import { ContextMenu, type MenuState } from './ContextMenu'
import { HiddenToggle } from './HiddenToggle'

type Props = {
  projects: domain.Project[]
  selectedId: string | null
  onSelect: (id: string) => void
  onChanged: () => void
  onError: (message: string) => void
  /** Incrémenté à chaque mutation ailleurs dans l'application, pour recharger
   *  compteurs et couleurs sans attendre un changement de vue (§2.1). */
  revision: number
  /** Œil de la liste des projets (§2.1) : montre ou non les projets cachés. */
  montrerCaches: boolean
  onToggleCaches: () => void
}

/**
 * Sidebar des projets (§2.1).
 *
 * L'ordre d'affichage n'est pas décidé ici : le backend rend déjà la liste
 * triée, « Transverse / Divers » en tête. Retrier côté React créerait une
 * seconde définition de la même règle.
 */
export function ProjectSidebar({
  projects,
  selectedId,
  onSelect,
  onChanged,
  onError,
  revision,
  montrerCaches,
  onToggleCaches,
}: Props) {
  const [menu, setMenu] = useState<MenuState>(null)
  const affiches = montrerCaches ? projects : projects.filter((p) => !p.hidden)

  async function tenter(action: () => Promise<unknown>) {
    try {
      await action()
      onChanged()
    } catch (err) {
      // Nom déjà pris, projet verrouillé : ces refus viennent du domaine et
      // doivent être montrés, pas avalés (§2.1).
      onError(String(err))
    }
  }

  async function creer() {
    const nom = window.prompt('Nom du projet ?')
    if (!nom?.trim()) return
    await tenter(() => api.CreateProject(nom.trim()))
  }

  async function renommer(project: domain.Project) {
    const nom = window.prompt('Nouveau nom ?', project.name)
    if (!nom?.trim() || nom === project.name) return
    await tenter(() => api.RenameProject(project.id, nom.trim()))
  }

  async function supprimer(project: domain.Project) {
    const resume = await api.ProjectDeletionSummary(project.id)
    // La suppression est en cascade et sans retour : on annonce le coût (§2.1).
    const message =
      `Supprimer « ${project.name} » ?\n\n` +
      `${resume.tasks} tâche(s), ${resume.notes} note(s) et ${resume.meetings} réunion(s) ` +
      `seront supprimées définitivement.`
    if (!window.confirm(message)) return
    await tenter(() => api.DeleteProject(project.id))
  }

  async function basculerCache(project: domain.Project) {
    await tenter(() => api.SetProjectHidden(project.id, !project.hidden))
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-end border-b border-neutral-200 bg-neutral-50 px-2 py-1">
        <HiddenToggle montrerCaches={montrerCaches} onToggle={onToggleCaches} />
      </div>
      <nav
        className="min-h-0 flex-1 overflow-auto bg-neutral-50 p-2"
        aria-label="Projets"
        onContextMenu={(e) => {
          e.preventDefault()
          setMenu({ x: e.clientX, y: e.clientY, items: [{ kind: 'action', label: '+ Nouveau projet', onSelect: creer }] })
        }}
        // Double-clic sur le fond : création directe (§2.1). Le double-clic sur
        // une ligne garde son sens, qui est de renommer — d'où le filtre.
        onDoubleClick={(e) => {
          if (surLeFond(e)) creer()
        }}
      >
        <ul className="flex flex-col gap-1">
          {affiches.map((project) => (
            <ProjectRow
              key={project.id}
              project={project}
              selected={project.id === selectedId}
              revision={revision}
              onSelect={onSelect}
              onRename={renommer}
              onMenu={(x, y) =>
                setMenu({
                  x,
                  y,
                  items: [
                    { kind: 'action', label: '+ Nouveau projet', onSelect: creer },
                    { kind: 'separator' },
                    // Le projet verrouillé ne se renomme, ne se supprime ni ne se
                    // cache : les entrées restent visibles mais grisées, plutôt
                    // qu'absentes, pour que leur indisponibilité soit lisible (§2.1).
                    { kind: 'action', label: 'Renommer', disabled: project.locked, onSelect: () => renommer(project) },
                    {
                      kind: 'action',
                      label: project.hidden ? 'Réafficher' : 'Cacher',
                      disabled: project.locked,
                      onSelect: () => basculerCache(project),
                    },
                    {
                      kind: 'action',
                      label: 'Supprimer',
                      danger: true,
                      disabled: project.locked,
                      onSelect: () => supprimer(project),
                    },
                  ],
                })
              }
            />
          ))}
        </ul>
      </nav>
      <ContextMenu state={menu} onClose={() => setMenu(null)} />
    </div>
  )
}

function ProjectRow({
  project,
  selected,
  onSelect,
  onRename,
  onMenu,
  revision,
}: {
  project: domain.Project
  selected: boolean
  onSelect: (id: string) => void
  revision: number
  onRename: (p: domain.Project) => void
  onMenu: (x: number, y: number) => void
}) {
  const [counts, setCounts] = useState<stats.SidebarStats | null>(null)

  // `revision` change à chaque mutation faite ailleurs dans l'application :
  // c'est ce qui recharge le compteur et la couleur sans attendre un
  // changement de vue. Sans cette dépendance, ajouter une tâche ou modifier une
  // criticité ne se voyait qu'après un aller-retour par l'onglet Priorités,
  // qui démontait puis remontait la sidebar.
  useEffect(() => {
    let cancelled = false

    const charger = () =>
      api
        .GetSidebarStats(project.id)
        .then((result) => {
          if (!cancelled) setCounts(result)
        })
        .catch(() => {
          // Un compteur indisponible ne doit pas faire disparaître le projet de
          // la liste : on l'affiche sans ses chiffres.
          if (!cancelled) setCounts(null)
        })

    charger()

    // L'icône d'échéance du projet dépend de l'heure : le §2.3 demande que le
    // rafraîchissement de 30 s touche l'arbre, la sidebar et les priorités.
    const timer = setInterval(charger, 30_000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [project.id, project.name, revision])

  return (
    <li data-ligne>
      <button
        type="button"
        onClick={() => onSelect(project.id)}
        onDoubleClick={() => {
          // Double-clic pour renommer (§2.1) ; sans effet sur le projet
          // verrouillé.
          if (!project.locked) onRename(project)
        }}
        onContextMenu={(e) => {
          e.preventDefault()
          e.stopPropagation()
          onSelect(project.id)
          onMenu(e.clientX, e.clientY)
        }}
        aria-current={selected ? 'true' : undefined}
        className={`w-full rounded border px-2 py-1.5 text-left text-sm ${background(project, counts)} ${
          selected ? 'border-2 border-[var(--color-selection)]' : 'border-neutral-300'
        } ${project.hidden ? 'opacity-50' : ''}`}
      >
        <div className="flex items-baseline justify-between gap-2">
          <span
            className={`truncate ${project.locked ? 'text-neutral-500' : 'text-neutral-900'} ${
              project.hidden ? 'italic' : ''
            }`}
          >
            {project.name}
          </span>
          <span className="flex shrink-0 items-center gap-1 text-xs text-neutral-700">
            {counts ? counts.activeCount : '—'}
            {counts?.dueIcon ? <span aria-hidden>{counts.dueIcon}</span> : null}
          </span>
        </div>
        <div className="text-xs text-neutral-500">{counts ? compteurs(counts) : ' '}</div>
      </button>
    </li>
  )
}

/** `N(J) notes · M(K) réunions`, où `(J)`/`(K)` n'apparaît que s'il y a des éléments cachés (§2.1). */
function compteurs(counts: stats.SidebarStats): string {
  const notes = counts.hiddenNotesCount > 0 ? `${counts.notesCount}(${counts.hiddenNotesCount})` : `${counts.notesCount}`
  const reunions =
    counts.hiddenMeetingsCount > 0
      ? `${counts.meetingsCount}(${counts.hiddenMeetingsCount})`
      : `${counts.meetingsCount}`
  return `${notes} notes · ${reunions} réunions`
}

/**
 * Couleur de fond d'un projet (§2.1).
 *
 * « Transverse / Divers » est en gris fixe, sans logique de criticité et sans
 * icône cadenas : le style grisé suffit à signaler qu'il est spécial.
 */
function background(project: domain.Project, counts: stats.SidebarStats | null): string {
  if (project.locked) return 'bg-[var(--color-annulee)]'
  if (counts?.hasCritical) return 'bg-[var(--color-critique)]'
  if (counts?.hasHigh) return 'bg-[var(--color-haute)]'
  return 'bg-white'
}
