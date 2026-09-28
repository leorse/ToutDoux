import type { domain } from '../wailsjs/go/models'

/**
 * Backend de démonstration, utilisé quand l'interface tourne hors de Wails :
 * `npm run dev` seul, captures d'écran, mise au point d'un composant.
 *
 * Ce n'est pas une seconde implémentation du métier — il n'y a ici aucune règle,
 * seulement un jeu de données en mémoire. Les règles (cascade, échéances,
 * filtres) vivent en Go et n'ont pas d'équivalent JavaScript : dupliquer la
 * cascade ici créerait exactement les deux définitions divergentes que
 * l'architecture cherche à éviter.
 */

const PROJET_DIVERS = 'project-divers'

function iso(offsetHeures: number): string {
  return new Date(Date.now() + offsetHeures * 3_600_000).toISOString()
}

/**
 * Jeu de données initial, reconstruit à chaque appel.
 *
 * Ce sont des fabriques et non des constantes : le backend de démonstration est
 * un module à état mutable, et deux tests qui le partageraient se
 * contamineraient — le second verrait les tâches créées par le premier.
 */
const donneesInitiales = () => ({
  projects: [
    { id: PROJET_DIVERS, name: 'Transverse / Divers', locked: true, createdAt: iso(-72), updatedAt: iso(-72) },
    { id: 'p-mig', name: 'Migration 2026', locked: false, createdAt: iso(-48), updatedAt: iso(-2) },
    { id: 'p-dr', name: 'Refonte du portail', locked: false, createdAt: iso(-24), updatedAt: iso(-1) },
    // Caché par défaut : sert à vérifier que ses tâches restent dans les
    // Priorités et que le naviguer vers lui rouvre l'œil des projets (§2.1).
    { id: 'p-old', name: 'Ancien portail', locked: false, hidden: true, createdAt: iso(-200), updatedAt: iso(-100) },
  ] as unknown as domain.Project[],

  tasks: [
    t('t1', 'p-mig', null, 'Préparer la réunion de cadrage', 'Critique', 0, { due: iso(0.5) }),
    t('t2', 'p-mig', 't1', 'Rassembler les chiffres', 'Haute', 0, { due: iso(-3) }),
    t('t3', 'p-mig', 't1', 'Relire le compte rendu', 'Normale', 1, { completed: true }),
    t('t4', 'p-mig', 't3', 'Corriger les annexes', 'Basse', 0, { completed: true }),
    t('t5', 'p-mig', null, 'Migrer le référentiel', 'Normale', 1, { due: iso(30) }),
    t('t6', 'p-mig', null, 'Ancien périmètre', 'Basse', 2, { cancelled: true }),
    t('t7', PROJET_DIVERS, null, 'Commander le matériel', 'Normale', 0, {}),
    t('t8', 'p-dr', null, 'Maquette de la page d’accueil', 'Haute', 0, { due: iso(72) }),
    // Du projet caché p-old : doit rester visible dans les Priorités (§2.1).
    t('t9', 'p-old', null, 'Tâche du projet caché', 'Critique', 0, {}),
  ] as unknown as domain.Task[],

  notes: [
    n('n1', 'p-mig', 'Compte rendu du 12/08', '<p>Échéance repoussée au 30.</p>'),
    n('n2', 'p-mig', 'Points en suspens', '<p>Valider le périmètre.</p>'),
    n('n3', PROJET_DIVERS, 'Divers', ''),
    // Cachée par défaut, pour que l'œil de démonstration ait quelque chose à révéler.
    { ...n('n4', 'p-mig', 'Ancienne version du cahier des charges', ''), hidden: true },
  ] as unknown as domain.Note[],

  meetings: [
    { id: "m1", projectId: "p-mig", title: "Comité de pilotage", createdAt: iso(-48), updatedAt: iso(-2) },
    { id: "m2", projectId: "p-mig", title: "Point technique", createdAt: iso(-24), updatedAt: iso(-1) },
    // Cachée par défaut, pour que l'œil de démonstration ait quelque chose à révéler.
    { id: "m3", projectId: "p-mig", title: "Ancien comité, abandonné", hidden: true, createdAt: iso(-96), updatedAt: iso(-96) },
  ] as unknown as domain.Meeting[],

  instances: [
    { id: "i1", meetingId: "m1", notes: "<p>Ordre du jour de la réunion du 12.</p>", timestamp: iso(-48), createdAt: iso(-48), updatedAt: iso(-48) },
    { id: "i2", meetingId: "m1", notes: "<p>Échéance repoussée, revoir le périmètre.</p>", timestamp: iso(-2), createdAt: iso(-2), updatedAt: iso(-2) },
  ] as unknown as domain.MeetingInstance[],
})

let { projects, tasks, notes, meetings, instances } = donneesInitiales()

/**
 * Index sémantique de démonstration (§2.12).
 *
 * Il conserve l'opt-in strict du vrai backend — rien n'y entre sans appel
 * explicite — parce que c'est justement la règle que l'interface doit refléter.
 * Le « modèle » est ici toujours disponible : hors de Wails il n'y a pas de
 * fichier à détecter, et refuser la fonctionnalité rendrait la vue intestable.
 */
let indexSemantique = new Set<string>()

/** Restaure le jeu de démonstration. Appelé entre deux tests. */
export function resetDemoData(): void {
  ;({ projects, tasks, notes, meetings, instances } = donneesInitiales())
  indexSemantique = new Set<string>()
}

function t(
  id: string,
  projectId: string,
  parentId: string | null,
  name: string,
  importance: string,
  orderIndex: number,
  opts: { due?: string; completed?: boolean; cancelled?: boolean },
) {
  return {
    id, projectId, parentId, name, description: '',
    importance, completed: !!opts.completed, cancelled: !!opts.cancelled,
    dueDate: opts.due ?? null, orderIndex,
    createdAt: iso(-24), updatedAt: iso(-1),
  }
}

function n(id: string, projectId: string, title: string, content: string) {
  return { id, projectId, title, content, createdAt: iso(-24), updatedAt: iso(-1) }
}

const uid = () => `demo-${Math.random().toString(36).slice(2, 10)}`

export const demoBackend = {
  ListProjects: async () => projects,

  CreateProject: async (name: string) => {
    if (projects.some((p) => p.name.toLowerCase() === name.toLowerCase())) {
      throw new Error('un projet porte déjà ce nom')
    }
    const p = { id: uid(), name, locked: false, createdAt: iso(0), updatedAt: iso(0) }
    projects = [...projects, p as unknown as domain.Project]
    return p
  },
  RenameProject: async (id: string, newName: string) => {
    projects = projects.map((p) => (p.id === id ? { ...p, name: newName } : p)) as unknown as domain.Project[]
    return projects.find((p) => p.id === id)!
  },
  ProjectDeletionSummary: async (id: string) => ({
    tasks: tasks.filter((x) => x.projectId === id).length,
    notes: notes.filter((x) => x.projectId === id).length,
    meetings: meetings.filter((m) => m.projectId === id).length,
  }),
  DeleteProject: async (id: string) => {
    projects = projects.filter((p) => p.id !== id)
    tasks = tasks.filter((x) => x.projectId !== id)
    notes = notes.filter((x) => x.projectId !== id)
    const partants = meetings.filter((m) => m.projectId === id).map((m) => m.id)
    meetings = meetings.filter((m) => m.projectId !== id)
    instances = instances.filter((i) => !partants.includes(i.meetingId))
  },

  /* -------- Masquage (§2.1, §2.6, §2.7) -------- */

  SetProjectHidden: async (id: string, hidden: boolean) => {
    const cible = projects.find((p) => p.id === id)
    if (cible?.locked) throw new Error('ce projet est verrouillé : il ne peut être ni renommé, ni supprimé, ni caché')
    projects = projects.map((p) => (p.id === id ? { ...p, hidden } : p)) as unknown as domain.Project[]
    return projects.find((p) => p.id === id)!
  },
  SetNoteHidden: async (id: string, hidden: boolean) => {
    notes = notes.map((n) => (n.id === id ? { ...n, hidden } : n)) as unknown as domain.Note[]
    return notes.find((n) => n.id === id)!
  },
  SetMeetingHidden: async (id: string, hidden: boolean) => {
    meetings = meetings.map((m) => (m.id === id ? { ...m, hidden } : m)) as unknown as domain.Meeting[]
    return meetings.find((m) => m.id === id)!
  },

  GetSidebarStats: async (projectId: string) => {
    const actives = tasks.filter((x) => x.projectId === projectId && !x.completed && !x.cancelled)
    const notesDuProjet = notes.filter((x) => x.projectId === projectId)
    const reunionsDuProjet = meetings.filter((m) => m.projectId === projectId)
    return {
      activeCount: actives.length,
      hasCritical: actives.some((x) => x.importance === 'Critique'),
      hasHigh: actives.some((x) => x.importance === 'Haute') && !actives.some((x) => x.importance === 'Critique'),
      dueIcon: actives.some((x) => x.dueDate) ? '🕐' : '',
      notesCount: notesDuProjet.filter((x) => !x.hidden).length,
      hiddenNotesCount: notesDuProjet.filter((x) => x.hidden).length,
      meetingsCount: reunionsDuProjet.filter((m) => !m.hidden).length,
      hiddenMeetingsCount: reunionsDuProjet.filter((m) => m.hidden).length,
    }
  },

  GetTasks: async (projectId: string) =>
    tasks.filter((x) => x.projectId === projectId).sort((a, b) => a.orderIndex - b.orderIndex),
  GetAllTasks: async () => tasks,

  /**
   * Vue de l'arbre en mode démonstration.
   *
   * Le filtrage reproduit ici est volontairement plat : il ne garde PAS les
   * ancêtres des tâches retenues, contrairement à la règle du §2.4 appliquée
   * par le domaine Go. Reproduire fidèlement la visibilité et le dépliement
   * demanderait de réécrire le domaine en TypeScript.
   */
  GetTaskView: async (projectId: string, filtres: { status?: string[]; importance?: string[]; dueOnly?: boolean }) => {
    const duProjet = tasks
      .filter((x) => x.projectId === projectId)
      .sort((a, b) => a.orderIndex - b.orderIndex)
    const statut = (x: (typeof duProjet)[number]) =>
      x.cancelled ? 'cancelled' : x.completed ? 'completed' : 'active'

    const retenues = duProjet.filter(
      (x) =>
        (filtres.status ?? ['active']).includes(statut(x)) &&
        (filtres.importance ?? []).includes(x.importance) &&
        (!filtres.dueOnly || !!x.dueDate),
    )
    const gardees = new Set(retenues.map((x) => x.id))
    // On remonte les parents pour que l'arbre reste navigable en démonstration.
    for (const x of retenues) {
      let p: string | null | undefined = x.parentId
      while (p) {
        gardees.add(p)
        const parent: string | null | undefined = p
        p = duProjet.find((y) => y.id === parent)?.parentId
      }
    }

    const due: Record<string, { text: string; urgent: boolean; overdue: boolean }> = {}
    for (const x of duProjet) {
      if (!x.dueDate) continue
      const delta = new Date(x.dueDate).getTime() - Date.now()
      due[x.id] = {
        text: delta < 0 ? 'En retard' : `dans ${Math.round(delta / 60000)} min`,
        urgent: delta >= 0 && delta < 5 * 60000,
        overdue: delta < 0,
      }
    }

    return {
      tasks: duProjet,
      visible: [...gardees],
      defaultExpanded: duProjet.filter((x) => !x.completed && !x.cancelled).map((x) => x.parentId ?? x.id),
      due,
    }
  },

  SetTaskDueDateQuick: async (taskId: string, kind: string) => {
    const minutes: Record<string, number> = { now: 0, '10min': 10, '1h': 60, journee: 480, lendemain9h: 1200 }
    const quand = new Date(Date.now() + (minutes[kind] ?? 0) * 60000).toISOString()
    tasks = tasks.map((x) => (x.id === taskId ? { ...x, dueDate: quand } : x)) as unknown as domain.Task[]
    return tasks.find((x) => x.id === taskId)!
  },

  MoveTask: async (taskId: string, direction: string) => {
    const cible = tasks.find((x) => x.id === taskId)!
    const freres = tasks
      .filter((x) => x.projectId === cible.projectId && (x.parentId ?? null) === (cible.parentId ?? null))
      .sort((a, b) => a.orderIndex - b.orderIndex)
    const from = freres.findIndex((x) => x.id === taskId)
    const to =
      direction === 'up' ? Math.max(0, from - 1)
      : direction === 'down' ? Math.min(freres.length - 1, from + 1)
      : direction === 'top' ? 0
      : freres.length - 1
    const [bouge] = freres.splice(from, 1)
    freres.splice(to, 0, bouge)
    const ordres = new Map(freres.map((x, i) => [x.id, i]))
    tasks = tasks.map((x) => (ordres.has(x.id) ? { ...x, orderIndex: ordres.get(x.id)! } : x)) as unknown as domain.Task[]
    return freres
  },

  ReparentTask: async (taskId: string, newParentId: string) => {
    tasks = tasks.map((x) =>
      x.id === taskId ? { ...x, parentId: newParentId || null } : x,
    ) as unknown as domain.Task[]
    return [tasks.find((x) => x.id === taskId)!]
  },

  CreateTask: async (projectId: string, parentId: string, name: string) => {
    const siblings = tasks.filter((x) => x.projectId === projectId && (x.parentId ?? null) === (parentId || null))
    const task = t(uid(), projectId, parentId || null, name, 'Normale', siblings.length, {})
    tasks = [...tasks, task as unknown as domain.Task]
    return { task, reactivated: [] }
  },

  UpdateTask: async (taskId: string, patch: Record<string, unknown>) => {
    tasks = tasks.map((x) =>
      x.id === taskId
        ? {
            ...x,
            name: (patch.name as string) ?? x.name,
            description: (patch.description as string) ?? x.description,
            importance: (patch.importance as string) ?? x.importance,
            dueDate: patch.clearDue ? null : ((patch.dueDate as string) ?? x.dueDate),
          }
        : x,
    ) as unknown as domain.Task[]
    return tasks.find((x) => x.id === taskId)!
  },

  DeleteTask: async (taskId: string) => {
    const doomed = new Set([taskId])
    let grew = true
    while (grew) {
      grew = false
      for (const x of tasks) {
        if (x.parentId && doomed.has(x.parentId) && !doomed.has(x.id)) {
          doomed.add(x.id)
          grew = true
        }
      }
    }
    tasks = tasks.filter((x) => !doomed.has(x.id))
  },

  TaskDeletionSummary: async (taskId: string) =>
    tasks.filter((x) => x.parentId === taskId).length,

  // Attention : ces deux bascules ne reproduisent PAS la cascade du §2.2, qui
  // n'existe qu'en Go. En mode démonstration, cocher un enfant ne remonte pas
  // sur ses parents. C'est assumé — ce mode sert à travailler la mise en page,
  // pas à valider le métier, qui est couvert par les tests Go.
  ToggleTaskCompleted: async (taskId: string) => {
    tasks = tasks.map((x) =>
      x.id === taskId ? { ...x, completed: !x.completed, cancelled: false } : x,
    ) as unknown as domain.Task[]
    return [tasks.find((x) => x.id === taskId)!]
  },
  ToggleTaskCancelled: async (taskId: string) => {
    tasks = tasks.map((x) =>
      x.id === taskId ? { ...x, cancelled: !x.cancelled, completed: false } : x,
    ) as unknown as domain.Task[]
    return [tasks.find((x) => x.id === taskId)!]
  },

  GetPriorityTasks: async () => {
    const actives = tasks.filter((x) => !x.completed && !x.cancelled)
    const parUrgence = (a: (typeof actives)[number], b: (typeof actives)[number]) => {
      if (a.dueDate && b.dueDate) return +new Date(a.dueDate) - +new Date(b.dueDate)
      if (a.dueDate) return -1
      if (b.dueDate) return 1
      return 0
    }
    const critiques = actives
      .filter((x) => x.importance === 'Critique' || x.importance === 'Haute')
      .sort(parUrgence)
    const datees = actives.filter((x) => x.dueDate).sort(parUrgence)
    return {
      criticalOrHigh: critiques,
      dueSoon: datees,
      // Dédoublonné : une tâche critique **et** datée figure dans les deux
      // listes mais ne compte que pour une (§2.8).
      total: new Set([...critiques, ...datees].map((x) => x.id)).size,
    }
  },

  /**
   * Recherche de démonstration : simple inclusion de sous-chaîne, là où la
   * version réelle interroge FTS5. Suffisant pour la mise en page, sans
   * prétendre reproduire la tokenisation du moteur.
   */
  SearchGlobal: async (query: string) => {
    const q = query.trim().toLowerCase()
    if (q.length < 3) return []
    const nomProjet = (id: string) => projects.find((p) => p.id === id)?.name ?? ''
    const extrait = (texte: string) => {
      const brut = texte.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
      const i = brut.toLowerCase().indexOf(q)
      if (i === -1) return { leadingEllipsis: false, parts: [{ text: brut, match: false }], trailingEllipsis: false }
      return {
        leadingEllipsis: i > 0,
        parts: [
          { text: brut.slice(Math.max(0, i - 40), i), match: false },
          { text: brut.slice(i, i + q.length), match: true },
          { text: brut.slice(i + q.length, i + q.length + 40), match: false },
        ],
        trailingEllipsis: i + q.length + 40 < brut.length,
      }
    }

    const out: unknown[] = []
    // Les projets sont cherchables comme le reste (§2.9).
    for (const p of projects) {
      if (p.name.toLowerCase().includes(q)) {
        out.push({
          type: 'project', id: p.id, title: p.name, projectId: p.id,
          projectName: p.name, date: p.updatedAt, project: p,
          snippet: extrait(p.name),
        })
      }
    }
    for (const t of tasks) {
      if (`${t.name} ${t.description}`.toLowerCase().includes(q)) {
        out.push({
          type: 'task', id: t.id, title: t.name, projectId: t.projectId,
          projectName: nomProjet(t.projectId), date: t.dueDate, task: t,
          snippet: extrait(t.description || t.name),
        })
      }
    }
    for (const n of notes) {
      if (`${n.title} ${n.content}`.toLowerCase().includes(q)) {
        out.push({
          type: 'note', id: n.id, title: n.title || 'Sans titre', projectId: n.projectId,
          projectName: nomProjet(n.projectId), date: n.updatedAt, note: n,
          snippet: extrait(n.content || n.title),
        })
      }
    }
    for (const m of meetings) {
      if (m.title.toLowerCase().includes(q)) {
        out.push({
          type: 'meeting', id: m.id, title: m.title, projectId: m.projectId,
          projectName: nomProjet(m.projectId), date: m.updatedAt, meeting: m,
          snippet: extrait(m.title),
        })
      }
    }
    for (const i of instances) {
      if ((i.notes ?? '').toLowerCase().includes(q)) {
        const m = meetings.find((x) => x.id === i.meetingId)
        if (!m) continue
        out.push({
          type: 'meeting', id: i.id, title: m.title, projectId: m.projectId,
          projectName: nomProjet(m.projectId), date: i.timestamp, meeting: m, instance: i,
          snippet: extrait(i.notes ?? ''),
        })
      }
    }
    return out
  },

  GetMeetings: async (projectId: string) => meetings.filter((m) => m.projectId === projectId),
  CreateMeeting: async (projectId: string, title: string) => {
    const m = { id: uid(), projectId, title, createdAt: iso(0), updatedAt: iso(0) }
    meetings = [...meetings, m as unknown as domain.Meeting]
    return m
  },
  RenameMeeting: async (meetingId: string, newTitle: string) => {
    meetings = meetings.map((m) => (m.id === meetingId ? { ...m, title: newTitle } : m)) as unknown as domain.Meeting[]
    return meetings.find((m) => m.id === meetingId)!
  },
  DeleteMeeting: async (meetingId: string) => {
    meetings = meetings.filter((m) => m.id !== meetingId)
    instances = instances.filter((i) => i.meetingId !== meetingId)
  },
  GetInstances: async (meetingId: string) =>
    instances
      .filter((i) => i.meetingId === meetingId)
      .sort((a, b) => +new Date(b.timestamp) - +new Date(a.timestamp)),
  AddMeetingInstance: async (meetingId: string) => {
    const i = { id: uid(), meetingId, notes: '', timestamp: iso(0), createdAt: iso(0), updatedAt: iso(0) }
    instances = [...instances, i as unknown as domain.MeetingInstance]
    return i
  },
  UpdateInstanceNotes: async (instanceId: string, notesHtml: string) => {
    instances = instances.map((i) =>
      i.id === instanceId ? { ...i, notes: notesHtml, updatedAt: iso(0) } : i,
    ) as unknown as domain.MeetingInstance[]
    return instances.find((i) => i.id === instanceId)!
  },
  DeleteInstance: async (instanceId: string) => {
    instances = instances.filter((i) => i.id !== instanceId)
  },

  GetNotes: async (projectId: string) => notes.filter((x) => x.projectId === projectId),
  CreateNote: async (projectId: string, title: string) => {
    const note = n(uid(), projectId, title, '')
    notes = [...notes, note as unknown as domain.Note]
    return note
  },
  UpdateNote: async (noteId: string, title: string, content: string) => {
    notes = notes.map((x) => (x.id === noteId ? { ...x, title, content, updatedAt: new Date().toISOString() } : x)) as unknown as domain.Note[]
    return notes.find((x) => x.id === noteId)!
  },
  DeleteNote: async (noteId: string) => {
    notes = notes.filter((x) => x.id !== noteId)
  },

  /* -------- À propos -------- */

  GetAppInfo: async () => ({
    version: '1.1.0',
    releases: [
      {
        version: '1.1.0',
        date: '2026-09-28',
        important: 'Démonstration : ce texte est un exemple d’**information importante**.',
        changes: ['Le **numéro de version** s’affiche dans la barre de titre', 'Nouvel onglet *À propos*'],
      },
      { version: '1.0.0', date: '2026-08-27', changes: ['Première version'] },
    ],
  }),

  /* -------- Recherche sémantique (§2.12) et modèle (§2.13) -------- */

  GetModelStatus: async () => ({
    available: true,
    directory: 'C:\Users\Demo\AppData\Roaming\ToutDoux\models',
    expectedFiles: ['model.onnx', 'tokenizer.json', 'sentencepiece.bpe.model'],
    missingFiles: [],
    downloadUrl: 'https://huggingface.co/Xenova/multilingual-e5-small/tree/main',
    files: [
      { name: 'model.onnx', source: 'onnx/model_int8.onnx', present: true, size: 118_000_000 },
      { name: 'tokenizer.json', source: 'tokenizer.json', present: true, size: 17_000_000 },
      { name: 'sentencepiece.bpe.model', source: 'sentencepiece.bpe.model', present: true, size: 5_000_000 },
    ],
    runtime: { name: 'onnxruntime.dll', source: 'onnxruntime-win-x64-1.29.0.zip, dans lib/', present: true, size: 16_000_000 },
  }),

  GetSemanticStatus: async () => ({
    modelAvailable: true,
    filesPresent: true,
    indexedCount: indexSemantique.size,
  }),

  AddToSemanticIndex: async (_entityType: string, entityId: string) => {
    indexSemantique.add(entityId)
  },
  RemoveFromSemanticIndex: async (entityId: string) => {
    indexSemantique.delete(entityId)
  },
  IsInSemanticIndex: async (entityId: string) => indexSemantique.has(entityId),
  RefreshEmbedding: async (_entityId: string) => {},

  /**
   * Recherche sémantique de démonstration.
   *
   * Elle ne simule évidemment pas un modèle : elle compte les mots partagés
   * entre la requête et le texte. Ce qu'elle reproduit fidèlement, et c'est
   * tout ce que l'interface a besoin de montrer : seules les entités ajoutées
   * remontent, le tri est par score décroissant, et l'extrait ne porte aucune
   * surbrillance.
   */
  SearchSemantic: async (query: string) => {
    const mots = query.trim().toLowerCase().split(/\W+/).filter(Boolean)
    if (query.trim().length < 3) return []

    const nomProjet = (id: string) => projects.find((p) => p.id === id)?.name ?? ''
    const brut = (texte: string) => texte.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
    const score = (texte: string) => {
      const cible = brut(texte).toLowerCase()
      const touches = mots.filter((m) => cible.includes(m)).length
      return mots.length === 0 ? 0 : touches / mots.length
    }
    const extrait = (texte: string) => ({
      leadingEllipsis: false,
      parts: [{ text: brut(texte).slice(0, 180), match: false }],
      trailingEllipsis: brut(texte).length > 180,
    })

    const out: { id: string; score: number }[] = []
    const lignes = new Map<string, unknown>()

    for (const t of tasks) {
      if (!indexSemantique.has(t.id)) continue
      const s = score(`${t.name} ${t.description}`)
      if (s <= 0) continue
      out.push({ id: t.id, score: s })
      lignes.set(t.id, {
        type: 'task', id: t.id, title: t.name, projectId: t.projectId,
        projectName: nomProjet(t.projectId), date: t.dueDate, task: t,
        snippet: extrait(t.description || t.name), score: s,
      })
    }
    for (const n of notes) {
      if (!indexSemantique.has(n.id)) continue
      const s = score(`${n.title} ${n.content}`)
      if (s <= 0) continue
      out.push({ id: n.id, score: s })
      lignes.set(n.id, {
        type: 'note', id: n.id, title: n.title || 'Sans titre', projectId: n.projectId,
        projectName: nomProjet(n.projectId), date: n.updatedAt, note: n,
        snippet: extrait(n.content || n.title), score: s,
      })
    }
    for (const i of instances) {
      if (!indexSemantique.has(i.id)) continue
      const reunion = meetings.find((m) => m.id === i.meetingId)
      if (!reunion) continue
      const s = score(`${reunion.title} ${i.notes}`)
      if (s <= 0) continue
      out.push({ id: i.id, score: s })
      lignes.set(i.id, {
        type: 'meeting', id: i.id, title: reunion.title, projectId: reunion.projectId,
        projectName: nomProjet(reunion.projectId), date: i.timestamp,
        meeting: reunion, instance: i, snippet: extrait(i.notes), score: s,
      })
    }

    return out.sort((a, b) => b.score - a.score).map((x) => lignes.get(x.id)) as never
  },
}
