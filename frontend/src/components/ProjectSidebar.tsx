import { useEffect, useState } from 'react'
import { api } from '../api'
import { surLeFond } from '../fond'
import type { domain, stats } from '../../wailsjs/go/models'
import { ContextMenu, type MenuState } from './ContextMenu'

type Props = {
  projects: domain.Project[]
  selectedId: string | null
  onSelect: (id: string) => void
  onChanged: () => void
  onError: (message: string) => void
  /** Incrémenté à chaque mutation ailleurs dans l'application, pour recharger
   *  compteurs et couleurs sans attendre un changement de vue (§2.1). */
  revision: number
}

/**
 * Sidebar des projets (§2.1).
 *
 * L'ordre d'affichage n'est pas décidé ici : le backend rend déjà la liste
 * triée, « Transverse / Divers » en tête. Retrier côté React créerait une
 * seconde définition de la même règle.
 */
export function ProjectSidebar({ projects, selectedId, onSelect, onChanged, onError, revision }: Props) {
  const [menu, setMenu] = useState<MenuState>(null)

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

  return (
    <>
      <nav
        className="h-full min-h-full bg-neutral-50 p-2"
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
          {projects.map((project) => (
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
                    // Le projet verrouillé ne se renomme ni ne se supprime : les
                    // entrées restent visibles mais grisées, plutôt qu'absentes,
                    // pour que leur indisponibilité soit lisible (§2.1).
                    { kind: 'action', label: 'Renommer', disabled: project.locked, onSelect: () => renommer(project) },
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
    </>
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
        }`}
      >
        <div className="flex items-baseline justify-between gap-2">
          <span className={`truncate ${project.locked ? 'text-neutral-500' : 'text-neutral-900'}`}>
            {project.name}
          </span>
          <span className="flex shrink-0 items-center gap-1 text-xs text-neutral-700">
            {counts ? counts.activeCount : '—'}
            {counts?.dueIcon ? <span aria-hidden>{counts.dueIcon}</span> : null}
          </span>
        </div>
        <div className="text-xs text-neutral-500">
          {counts ? `${counts.notesCount} notes · ${counts.meetingsCount} réunions` : ' '}
        </div>
      </button>
    </li>
  )
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
