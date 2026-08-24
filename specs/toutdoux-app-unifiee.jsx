import { useState, useEffect, useRef, useCallback, useMemo } from "react";

/* ============================================================
   TOUT DOUX — Prototype unifié
   Fusionne les 3 prototypes précédents (Tâches / Notes / Réunions)
   en une seule app : projets + navigation par onglets partagés,
   plutôt que 3 sandbox isolées avec sidebar dupliquée.
   ============================================================ */

/* ---------------- Constantes & projets ---------------- */

const IMPORTANCES = ["Basse", "Normale", "Haute", "Critique"];

const IMPORTANCE_STYLE = {
  Critique: { bg: "#ff4d4d", text: "#ffffff", border: "#c22e2e" },
  Haute: { bg: "#ffb84d", text: "#3b2200", border: "#cc8a1f" },
  Normale: { bg: "#ffffff", text: "#1c2321", border: "#dce3df" },
  Basse: { bg: "#d9d9d9", text: "#333333", border: "#bbbbbb" },
};
const STATUS_STYLE = {
  completed: { bg: "#cdeed8", text: "#1c5230", border: "#8fcfa6" },
  cancelled: { bg: "#d9d9d9", text: "#6b6b6b", border: "#bbbbbb" },
};

const DIVERS_PROJECT_ID = "project-divers";
const DEFAULT_PROJECT_IDS = {
  mig: "project-mig24",
  vitrine: "project-vitrine",
  divers: DIVERS_PROJECT_ID,
};

function seedProjects() {
  return [
    { id: DEFAULT_PROJECT_IDS.mig, name: "Mig24", locked: false },
    { id: DEFAULT_PROJECT_IDS.vitrine, name: "Site vitrine", locked: false },
    { id: DIVERS_PROJECT_ID, name: "Transverse / Divers", locked: true },
  ];
}

function ensureDivers(projects) {
  if (projects.some((p) => p.id === DIVERS_PROJECT_ID)) return projects;
  return [...projects, { id: DIVERS_PROJECT_ID, name: "Transverse / Divers", locked: true }];
}

function uid() {
  return Math.random().toString(36).slice(2, 10);
}

/* ---------------- Helpers dates (tâches + échéances) ---------------- */

function addMinutes(date, m) {
  return new Date(date.getTime() + m * 60000);
}
function addDays(date, d) {
  const n = new Date(date);
  n.setDate(n.getDate() + d);
  return n;
}
function atTime(date, h, m) {
  const n = new Date(date);
  n.setHours(h, m, 0, 0);
  return n;
}
function isSameDay(a, b) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}
function startOfDay(d) {
  const n = new Date(d);
  n.setHours(0, 0, 0, 0);
  return n;
}
function nextWeekday9h(from) {
  let d = addDays(from, 1);
  while (d.getDay() === 0 || d.getDay() === 6) d = addDays(d, 1);
  return atTime(d, 9, 0);
}
function toDatetimeLocalValue(date) {
  const pad = (n) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function quickDueDate(kind, now) {
  switch (kind) {
    case "now":
      return new Date(now);
    case "10min":
      return addMinutes(now, 10);
    case "1h":
      return addMinutes(now, 60);
    case "journee":
      return atTime(now, 18, 0);
    case "lendemain9h":
      return nextWeekday9h(now);
    default:
      return null;
  }
}
function formatDueDate(due, now) {
  if (!due) return null;
  const diffMs = due.getTime() - now.getTime();

  if (diffMs < 0) {
    const overdueMs = -diffMs;
    const hours = Math.ceil(overdueMs / 3600000);
    if (hours < 24) return { text: `En retard de ${hours}h`, overdue: true };
    const days = Math.ceil(overdueMs / 86400000);
    return { text: `En retard de ${days}j`, overdue: true };
  }

  if (isSameDay(due, now)) {
    if (diffMs < 3600000) {
      const min = Math.max(1, Math.round(diffMs / 60000));
      return { text: `dans ${min} min`, urgent: diffMs < 5 * 60000 };
    }
    if (diffMs < 2 * 3600000) {
      const halfHours = Math.round(diffMs / 1800000);
      const h = Math.floor(halfHours / 2);
      const hasHalf = halfHours % 2 === 1;
      return { text: `~${h}h${hasHalf ? "30" : ""}` };
    }
    const h = Math.round(diffMs / 3600000);
    return { text: `~${h}h` };
  }

  const tomorrow = startOfDay(addDays(now, 1));
  if (isSameDay(due, tomorrow)) return { text: "Demain" };

  const diffDays = Math.ceil((startOfDay(due) - startOfDay(now)) / 86400000);
  if (diffDays <= 6) return { text: `dans ${diffDays} jours` };
  if (diffDays <= 13) return { text: "Semaine prochaine" };

  const weeks = Math.floor(diffDays / 7);
  const days = diffDays % 7;
  let text = `${weeks} semaine${weeks > 1 ? "s" : ""}`;
  if (days > 0) text += ` et ${days} jour${days > 1 ? "s" : ""}`;
  return { text };
}

function timeAgo(date, now) {
  if (!date) return "";
  const diffMs = now.getTime() - date.getTime();
  const min = Math.round(diffMs / 60000);
  if (min < 1) return "à l'instant";
  if (min < 60) return `il y a ${min} min`;
  const h = Math.round(min / 60);
  if (h < 24) return `il y a ${h}h`;
  const d = Math.round(h / 24);
  return `il y a ${d}j`;
}
function previewText(content, max = 60) {
  const flat = (content || "").replace(/\s+/g, " ").trim();
  return flat.length > max ? flat.slice(0, max) + "…" : flat;
}
function formatTimestamp(date) {
  return date.toLocaleString("fr-FR", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" });
}

// Recherche transverse tâches/notes/réunions — fonction pure, testable sans React ni backend
function computeSearchResults(query, scope, selectedProjectId, tasks, notes, meetings, instances) {
  const q = query.trim().toLowerCase();
  if (q.length <= 2) return [];
  const inScope = (projectId) => scope === "global" || projectId === selectedProjectId;
  const results = [];

  for (const t of tasks) {
    if (!inScope(t.projectId)) continue;
    const hay = `${t.name} ${t.description || ""}`.toLowerCase();
    if (hay.includes(q)) {
      results.push({ type: "task", id: t.id, title: t.name, projectId: t.projectId, date: t.dueDate || null, task: t });
    }
  }
  for (const n of notes) {
    if (!inScope(n.projectId)) continue;
    const hay = `${n.title} ${n.content || ""}`.toLowerCase();
    if (hay.includes(q)) {
      results.push({ type: "note", id: n.id, title: n.title || "Sans titre", projectId: n.projectId, date: n.updatedAt, note: n });
    }
  }
  for (const m of meetings) {
    if (!inScope(m.projectId)) continue;
    if (m.title.toLowerCase().includes(q)) {
      results.push({ type: "meeting", id: m.id, title: m.title, projectId: m.projectId, date: m.updatedAt, meeting: m, instance: null });
    }
    for (const inst of instances.filter((i) => i.meetingId === m.id)) {
      if ((inst.notes || "").toLowerCase().includes(q)) {
        results.push({ type: "meeting", id: inst.id, title: m.title, projectId: m.projectId, date: inst.timestamp, meeting: m, instance: inst });
      }
    }
  }

  // Plus récent d'abord ; les résultats sans date (tâche sans échéance) passent en dernier
  results.sort((a, b) => {
    if (a.date && b.date) return b.date.getTime() - a.date.getTime();
    if (a.date) return -1;
    if (b.date) return 1;
    return 0;
  });
  return results;
}

// Découpe un texte autour de la première occurrence et surligne le terme cherché
function highlightSnippet(text, query, contextChars = 90) {
  if (!text) return null;
  const lower = text.toLowerCase();
  const q = query.toLowerCase();
  const idx = lower.indexOf(q);
  if (idx === -1) return text.length > 160 ? text.slice(0, 160) + "…" : text;

  const start = Math.max(0, idx - contextChars);
  const end = Math.min(text.length, idx + q.length + contextChars);
  return (
    <>
      {start > 0 ? "…" : ""}
      {text.slice(start, idx)}
      <mark style={styles.searchHighlight}>{text.slice(idx, idx + q.length)}</mark>
      {text.slice(idx + q.length, end)}
      {end < text.length ? "…" : ""}
    </>
  );
}

/* ---------------- Logique arbre tâches ---------------- */

function buildChildrenMap(tasks) {
  const map = new Map();
  for (const t of tasks) {
    const key = t.parentId || "__root__";
    if (!map.has(key)) map.set(key, []);
    map.get(key).push(t);
  }
  for (const arr of map.values()) arr.sort((a, b) => a.orderIndex - b.orderIndex);
  return map;
}
function getDescendantIds(tasks, rootId) {
  const childrenMap = buildChildrenMap(tasks);
  const ids = new Set();
  const stack = [rootId];
  while (stack.length) {
    const id = stack.pop();
    const kids = childrenMap.get(id) || [];
    for (const k of kids) {
      ids.add(k.id);
      stack.push(k.id);
    }
  }
  return ids;
}
function matchesFilter(task, filters) {
  const status = task.cancelled ? "cancelled" : task.completed ? "completed" : "active";
  if (!filters.status.has(status)) return false;
  if (!filters.importance.has(task.importance)) return false;
  if (filters.dueOnly && !task.dueDate) return false;
  return true;
}
function computeVisibility(tasks, filters) {
  const childrenMap = buildChildrenMap(tasks);
  const visible = new Set();
  function visit(task) {
    const kids = childrenMap.get(task.id) || [];
    let anyChildVisible = false;
    for (const k of kids) if (visit(k)) anyChildVisible = true;
    const isVisible = matchesFilter(task, filters) || anyChildVisible;
    if (isVisible) visible.add(task.id);
    return isVisible;
  }
  for (const t of tasks) if (!t.parentId) visit(t);
  return visible;
}

function defaultExpandedForProject(tasks, projectId) {
  const projectTasks = tasks.filter((t) => t.projectId === projectId);
  const childrenMap = buildChildrenMap(projectTasks);
  const ids = new Set();

  function subtreeHasActive(task) {
    const kids = childrenMap.get(task.id) || [];
    let childActive = false;
    for (const k of kids) {
      if (subtreeHasActive(k)) childActive = true;
    }
    const selfActive = !task.completed && !task.cancelled;
    const result = selfActive || childActive;
    if (result) ids.add(task.id);
    return result;
  }

  for (const t of projectTasks) if (!t.parentId) subtreeHasActive(t);
  return ids;
}

// Stats sidebar : compteur actif + code couleur (criticité tâches) + icône échéance + volumes notes/réunions
function getSidebarStats(tasks, notes, meetings, projectId, now) {
  const activeTasks = tasks.filter((t) => t.projectId === projectId && !t.completed && !t.cancelled);
  const hasCritical = activeTasks.some((t) => t.importance === "Critique");
  const hasHigh = !hasCritical && activeTasks.some((t) => t.importance === "Haute");

  let dueIcon = null;
  for (const t of activeTasks) {
    const info = formatDueDate(t.dueDate, now);
    if (!info) continue;
    if (info.urgent || info.overdue) {
      dueIcon = "⏰";
      break;
    }
    if (!dueIcon) dueIcon = "🕐";
  }

  return {
    activeCount: activeTasks.length,
    hasCritical,
    hasHigh,
    dueIcon,
    notesCount: notes.filter((n) => n.projectId === projectId).length,
    meetingsCount: meetings.filter((m) => m.projectId === projectId).length,
  };
}

/* ---------------- Données de démo ---------------- */

function seedTasks(now) {
  const t = (over) => new Date(now.getTime() + over);
  const { mig, vitrine, divers } = DEFAULT_PROJECT_IDS;
  const root = (name, importance, projectId, extra = {}) => ({
    id: uid(),
    parentId: null,
    projectId,
    name,
    description: "",
    importance,
    completed: false,
    cancelled: false,
    dueDate: null,
    orderIndex: 0,
    ...extra,
  });

  const task1 = root("Préparer la démo client", "Haute", mig, { dueDate: t(-2 * 3600000), orderIndex: 0 });
  const task1_1 = root("Finaliser les slides", "Normale", mig, { parentId: task1.id, completed: true, orderIndex: 0 });
  const task1_2 = root("Relire le script", "Critique", mig, { parentId: task1.id, orderIndex: 1 });
  const task1_2_1 = root("Corriger la partie technique", "Haute", mig, { parentId: task1_2.id, orderIndex: 0 });
  const task2 = root("Répondre aux emails", "Normale", divers, { orderIndex: 0 });
  const task2_1 = root("Email fournisseur", "Basse", divers, { parentId: task2.id, cancelled: true, orderIndex: 0 });
  const task3 = root("Réviser le budget Q3", "Basse", vitrine, { dueDate: t(3 * 60000), orderIndex: 0 });
  const task4 = root("Appeler le prestataire", "Haute", mig, { dueDate: t(90 * 60000), orderIndex: 1 });

  return [task1, task1_1, task1_2, task1_2_1, task2, task2_1, task3, task4];
}

function seedNotes(now) {
  const { mig, vitrine, divers } = DEFAULT_PROJECT_IDS;
  const ago = (min) => new Date(now.getTime() - min * 60000);
  const mk = (projectId, title, content, minutesAgo) => ({
    id: uid(),
    projectId,
    title,
    content,
    updatedAt: ago(minutesAgo),
  });
  return [
    mk(mig, "Kickoff migration", "Notes de lancement :\n- Périmètre validé\n- Risques identifiés sur la base legacy", 18),
    mk(mig, "Architecture cible", "Découpage en services autour de la facturation, file d'attente pour les jobs async.", 240),
    mk(vitrine, "Brief client", "Site vitrine sobre, ton institutionnel. 5 pages prévues.", 60),
    mk(divers, "Pense-bête", "Renouveler le certificat SSL avant fin de mois.", 5),
  ];
}

function seedMeetingsAndInstances(now) {
  const { mig, vitrine } = DEFAULT_PROJECT_IDS;
  const ago = (min) => new Date(now.getTime() - min * 60000);

  const meetings = [
    { id: uid(), projectId: mig, title: "Point hebdo", createdAt: ago(60 * 24 * 30), updatedAt: ago(60 * 24) },
    { id: uid(), projectId: mig, title: "Revue technique", createdAt: ago(60 * 24 * 10), updatedAt: ago(60 * 3) },
    { id: uid(), projectId: vitrine, title: "Revue design", createdAt: ago(60 * 24 * 5), updatedAt: ago(60 * 20) },
  ];
  const [pointHebdo, revueTech, revueDesign] = meetings;

  const instances = [
    { id: uid(), meetingId: pointHebdo.id, notes: "Audit legacy terminé.\nBloquant : accès prod pas encore obtenu.", timestamp: ago(60 * 24), createdAt: ago(60 * 24), updatedAt: ago(60 * 24) },
    { id: uid(), meetingId: pointHebdo.id, notes: "Accès obtenu, migration du schéma en cours.", timestamp: ago(60 * 24 * 7), createdAt: ago(60 * 24 * 7), updatedAt: ago(60 * 24 * 7) },
    { id: uid(), meetingId: pointHebdo.id, notes: "Kickoff : périmètre validé avec le client.", timestamp: ago(60 * 24 * 14), createdAt: ago(60 * 24 * 14), updatedAt: ago(60 * 24 * 14) },
    { id: uid(), meetingId: revueTech.id, notes: "Architecture : microservices facturation. Décision finale la semaine prochaine.", timestamp: ago(60 * 3), createdAt: ago(60 * 3), updatedAt: ago(60 * 3) },
    { id: uid(), meetingId: revueDesign.id, notes: "Maquettes validées pour l'accueil. Reste : page contact.", timestamp: ago(60 * 20), createdAt: ago(60 * 20), updatedAt: ago(60 * 20) },
  ];

  return { meetings, instances };
}

/* ---------------- Panneaux redimensionnables ---------------- */

function useResizablePanel(initialWidth, min, max, invert = false) {
  const [width, setWidth] = useState(initialWidth);
  const dragState = useRef(null);

  useEffect(() => {
    const onMove = (e) => {
      if (!dragState.current) return;
      const delta = e.clientX - dragState.current.startX;
      const raw = dragState.current.startWidth + (invert ? -delta : delta);
      setWidth(Math.max(min, Math.min(max, raw)));
    };
    const onUp = () => {
      if (!dragState.current) return;
      dragState.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
    return () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
  }, [min, max, invert]);

  const onMouseDown = (e) => {
    e.preventDefault();
    dragState.current = { startX: e.clientX, startWidth: width };
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
  };

  return [width, onMouseDown];
}

function ResizeHandle({ onMouseDown }) {
  const [active, setActive] = useState(false);
  return (
    <div
      style={styles.resizeHandle}
      onMouseDown={(e) => {
        setActive(true);
        onMouseDown(e);
      }}
      onMouseEnter={() => setActive(true)}
      onMouseLeave={() => setActive(false)}
      onMouseUp={() => setActive(false)}
    >
      <div style={{ ...styles.resizeHandleBar, ...(active ? styles.resizeHandleBarActive : {}) }} />
    </div>
  );
}

/* ============================================================
   Composant principal
   ============================================================ */

export default function ToutDouxApp() {
  const [now, setNow] = useState(() => new Date());
  const [activeTab, setActiveTab] = useState("tasks"); // tasks | notes | meetings (sous-onglets de la vue Projets)
  const [mainView, setMainView] = useState("projects"); // projects | priorities

  // Largeurs des panneaux (une par séparation, réutilisées selon l'onglet actif)
  const [sidebarWidth, onSidebarResizeStart] = useResizablePanel(190, 150, 420);
  const [taskDetailWidth, onTaskDetailResizeStart] = useResizablePanel(320, 240, 560, true);
  const [noteListWidth, onNoteListResizeStart] = useResizablePanel(230, 180, 420);
  const [meetingsColWidth, onMeetingsResizeStart] = useResizablePanel(200, 160, 400);
  const [instancesColWidth, onInstancesResizeStart] = useResizablePanel(200, 160, 400);
  const [priorityDetailWidth, onPriorityDetailResizeStart] = useResizablePanel(320, 240, 560, true);

  // Recherche
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState("");
  const [selectedSearchResultKey, setSelectedSearchResultKey] = useState(null);
  const [searchResultsWidth, onSearchResultsResizeStart] = useResizablePanel(240, 180, 420);

  // Projets (partagés par tous les onglets)
  const [projects, setProjects] = useState(() => seedProjects());
  const [selectedProjectId, setSelectedProjectId] = useState(DEFAULT_PROJECT_IDS.mig);
  const [confirmDeleteProjectId, setConfirmDeleteProjectId] = useState(null);

  // Tâches
  const taskSeedRef = useRef(null);
  if (!taskSeedRef.current) taskSeedRef.current = seedTasks(new Date());
  const [tasks, setTasks] = useState(() => taskSeedRef.current);
  const [selectedTaskId, setSelectedTaskId] = useState(null);
  const [expandedTaskIds, setExpandedTaskIds] = useState(() => defaultExpandedForProject(taskSeedRef.current, DEFAULT_PROJECT_IDS.mig));
  const [taskFilters, setTaskFilters] = useState({ status: new Set(["active"]), importance: new Set(IMPORTANCES), dueOnly: false });
  const [dragTaskId, setDragTaskId] = useState(null);
  const [confirmDeleteTaskId, setConfirmDeleteTaskId] = useState(null);
  const [creatingTask, setCreatingTask] = useState(null); // { parentId, projectId } | null
  const [taskContextMenu, setTaskContextMenu] = useState(null);

  // Notes
  const [notes, setNotes] = useState(() => seedNotes(new Date()));
  const [selectedNoteId, setSelectedNoteId] = useState(null);
  const [noteContextMenu, setNoteContextMenu] = useState(null);
  const [confirmDeleteNoteId, setConfirmDeleteNoteId] = useState(null);
  const [renamingNoteId, setRenamingNoteId] = useState(null);
  const [noteRenameDraft, setNoteRenameDraft] = useState("");

  // Réunions
  const meetingSeedRef = useRef(null);
  if (!meetingSeedRef.current) meetingSeedRef.current = seedMeetingsAndInstances(new Date());
  const [meetings, setMeetings] = useState(() => meetingSeedRef.current.meetings);
  const [instances, setInstances] = useState(() => meetingSeedRef.current.instances);
  const [selectedMeetingId, setSelectedMeetingId] = useState(null);
  const [selectedInstanceId, setSelectedInstanceId] = useState(null);
  const [meetingContextMenu, setMeetingContextMenu] = useState(null);
  const [creatingMeeting, setCreatingMeeting] = useState(false);
  const [renamingMeetingId, setRenamingMeetingId] = useState(null);
  const [meetingRenameDraft, setMeetingRenameDraft] = useState("");
  const [confirmDeleteMeeting, setConfirmDeleteMeeting] = useState(null);
  const [confirmDeleteInstance, setConfirmDeleteInstance] = useState(null);

  // Priorités
  const [selectedPriorityTaskId, setSelectedPriorityTaskId] = useState(null);

  const [loaded, setLoaded] = useState(false);
  const saveTimer = useRef(null);

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 30000);
    return () => clearInterval(id);
  }, []);

  // Persistance unifiée (une seule clé pour tout l'état métier)
  useEffect(() => {
    (async () => {
      try {
        const res = await window.storage.get("toutdoux-unified-demo-v1");
        if (res && res.value) {
          const parsed = JSON.parse(res.value);
          if (parsed.projects) setProjects(ensureDivers(parsed.projects));
          if (parsed.tasks) {
            const loadedTasks = parsed.tasks.map((t) => ({ ...t, dueDate: t.dueDate ? new Date(t.dueDate) : null }));
            setTasks(loadedTasks);
            setExpandedTaskIds(defaultExpandedForProject(loadedTasks, DEFAULT_PROJECT_IDS.mig));
          }
          if (parsed.notes) setNotes(parsed.notes.map((n) => ({ ...n, updatedAt: new Date(n.updatedAt) })));
          if (parsed.meetings) {
            setMeetings(parsed.meetings.map((m) => ({ ...m, createdAt: new Date(m.createdAt), updatedAt: new Date(m.updatedAt) })));
          }
          if (parsed.instances) {
            setInstances(
              parsed.instances.map((i) => ({ ...i, timestamp: new Date(i.timestamp), createdAt: new Date(i.createdAt), updatedAt: new Date(i.updatedAt) }))
            );
          }
        }
      } catch (e) {
        // pas de données sauvegardées, on garde le seed
      } finally {
        setLoaded(true);
      }
    })();
  }, []);

  useEffect(() => {
    if (!loaded) return;
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(async () => {
      try {
        await window.storage.set("toutdoux-unified-demo-v1", JSON.stringify({ projects, tasks, notes, meetings, instances }));
      } catch (e) {
        // stockage indisponible, on continue en mémoire seulement
      }
    }, 400);
  }, [projects, tasks, notes, meetings, instances, loaded]);

  // Ferme le menu contextuel actif (un seul actif à la fois, quel que soit l'onglet) au clic ailleurs / Échap
  useEffect(() => {
    if (!taskContextMenu && !noteContextMenu && !meetingContextMenu) return;
    const close = () => {
      setTaskContextMenu(null);
      setNoteContextMenu(null);
      setMeetingContextMenu(null);
    };
    const onKey = (e) => {
      if (e.key === "Escape") close();
    };
    window.addEventListener("click", close);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("click", close);
      window.removeEventListener("keydown", onKey);
    };
  }, [taskContextMenu, noteContextMenu, meetingContextMenu]);

  // Debounce 300ms (spec 2.9)
  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearchQuery(searchQuery), 300);
    return () => clearTimeout(t);
  }, [searchQuery]);

  const searchActive = debouncedSearchQuery.trim().length > 2; // "Typing > 2 caractères"

  // Échap ferme la recherche et revient à l'onglet précédent (spec 2.9)
  useEffect(() => {
    if (!searchActive) return;
    const onKey = (e) => {
      if (e.key === "Escape") setSearchQuery("");
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [searchActive]);

  const selectedProject = projects.find((p) => p.id === selectedProjectId) || null;
  const sortedProjects = useMemo(
    () => [...projects].sort((a, b) => (a.locked === b.locked ? 0 : a.locked ? -1 : 1)),
    [projects]
  );
  const projectsById = useMemo(() => new Map(projects.map((p) => [p.id, p.name])), [projects]);

  /* ----- Dérivés Priorités (transverse, tous projets) ----- */
  const urgencySort = (a, b) => {
    if (a.dueDate && b.dueDate) return a.dueDate.getTime() - b.dueDate.getTime();
    if (a.dueDate) return -1;
    if (b.dueDate) return 1;
    return 0;
  };
  const criticalOrHighTasks = useMemo(
    () => tasks.filter((t) => !t.completed && !t.cancelled && (t.importance === "Critique" || t.importance === "Haute")).sort(urgencySort),
    [tasks]
  );
  const dueSoonTasks = useMemo(
    () => tasks.filter((t) => !t.completed && !t.cancelled && t.dueDate).sort(urgencySort),
    [tasks]
  );

  /* ----- Dérivés Recherche ----- */
  const searchResults = useMemo(
    () => (searchActive ? computeSearchResults(debouncedSearchQuery, "global", selectedProjectId, tasks, notes, meetings, instances) : []),
    [searchActive, debouncedSearchQuery, selectedProjectId, tasks, notes, meetings, instances]
  );
  useEffect(() => {
    setSelectedSearchResultKey(searchResults.length ? `${searchResults[0].type}:${searchResults[0].id}` : null);
  }, [searchResults]);
  const selectedSearchResult = searchResults.find((r) => `${r.type}:${r.id}` === selectedSearchResultKey) || null;

  /* ----- Dérivés Tâches ----- */
  const projectTasks = useMemo(() => tasks.filter((t) => t.projectId === selectedProjectId), [tasks, selectedProjectId]);
  const taskChildrenMap = useMemo(() => buildChildrenMap(projectTasks), [projectTasks]);
  const tasksById = useMemo(() => new Map(tasks.map((t) => [t.id, t])), [tasks]);
  const visibleTaskIds = useMemo(() => computeVisibility(projectTasks, taskFilters), [projectTasks, taskFilters]);
  const selectedTask = selectedTaskId ? tasksById.get(selectedTaskId) : null;
  const rootTasks = taskChildrenMap.get("__root__") || [];

  /* ----- Dérivés Notes ----- */
  const projectNotes = useMemo(
    () => notes.filter((n) => n.projectId === selectedProjectId).sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime()),
    [notes, selectedProjectId]
  );
  const notesById = useMemo(() => new Map(notes.map((n) => [n.id, n])), [notes]);
  const selectedNote = selectedNoteId ? notesById.get(selectedNoteId) : null;

  /* ----- Dérivés Réunions ----- */
  const projectMeetings = useMemo(
    () => meetings.filter((m) => m.projectId === selectedProjectId).sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime()),
    [meetings, selectedProjectId]
  );
  const meetingsById = useMemo(() => new Map(meetings.map((m) => [m.id, m])), [meetings]);
  const instancesById = useMemo(() => new Map(instances.map((i) => [i.id, i])), [instances]);
  const meetingInstances = useMemo(
    () => instances.filter((i) => i.meetingId === selectedMeetingId).sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime()),
    [instances, selectedMeetingId]
  );
  const selectedMeeting = selectedMeetingId ? meetingsById.get(selectedMeetingId) : null;
  const selectedInstance = selectedInstanceId ? instancesById.get(selectedInstanceId) : null;

  /* ----- Mutations Projets ----- */

  const addProject = useCallback((name) => {
    const trimmed = name.trim();
    if (!trimmed) return;
    setProjects((prev) => {
      const exists = prev.some((p) => p.name.toLowerCase() === trimmed.toLowerCase());
      if (exists) {
        window.alert(`Un projet nommé « ${trimmed} » existe déjà.`);
        return prev;
      }
      return [...prev, { id: uid(), name: trimmed, locked: false }];
    });
  }, []);

  const renameProject = useCallback((id, name) => {
    const trimmed = name.trim();
    if (!trimmed) return;
    setProjects((prev) => {
      const target = prev.find((p) => p.id === id);
      if (!target || target.locked) return prev;
      const exists = prev.some((p) => p.id !== id && p.name.toLowerCase() === trimmed.toLowerCase());
      if (exists) {
        window.alert(`Un projet nommé « ${trimmed} » existe déjà.`);
        return prev;
      }
      return prev.map((p) => (p.id === id ? { ...p, name: trimmed } : p));
    });
  }, []);

  // Cascade complète : tâches, notes, réunions + leurs instances
  const deleteProjectCascade = useCallback(
    (id) => {
      setProjects((prev) => {
        const target = prev.find((p) => p.id === id);
        if (!target || target.locked) return prev;
        return prev.filter((p) => p.id !== id);
      });
      setTasks((prev) => prev.filter((t) => t.projectId !== id));
      setNotes((prev) => prev.filter((n) => n.projectId !== id));
      const meetingIdsToRemove = new Set(meetings.filter((m) => m.projectId === id).map((m) => m.id));
      setMeetings((prev) => prev.filter((m) => m.projectId !== id));
      setInstances((prev) => prev.filter((i) => !meetingIdsToRemove.has(i.meetingId)));
      setSelectedProjectId((cur) => (cur === id ? DIVERS_PROJECT_ID : cur));
      setConfirmDeleteProjectId(null);
    },
    [meetings]
  );

  // Changer de projet réinitialise toutes les sélections/filtres par onglet (spec 2.1)
  const selectProject = useCallback(
    (id) => {
      setSelectedProjectId(id);
      setSelectedTaskId(null);
      setExpandedTaskIds(defaultExpandedForProject(tasks, id));
      // Les filtres de tâches restent tels quels d'un projet à l'autre (préférence utilisateur)
      setSelectedNoteId(null);
      setSelectedMeetingId(null);
      setSelectedInstanceId(null);
      setTaskContextMenu(null);
      setNoteContextMenu(null);
      setMeetingContextMenu(null);
    },
    [tasks]
  );

  /* ----- Mutations Tâches ----- */

  const updateTask = useCallback((id, patch) => {
    setTasks((prev) => prev.map((t) => (t.id === id ? { ...t, ...patch } : t)));
  }, []);

  const addTask = useCallback((parentId, name, projectId) => {
    setTasks((prev) => {
      const siblings = prev.filter((t) => (t.parentId || null) === (parentId || null) && t.projectId === projectId);
      const maxOrder = siblings.length ? Math.max(...siblings.map((s) => s.orderIndex)) : -1;
      const nt = {
        id: uid(),
        parentId: parentId || null,
        projectId,
        name: name || "Nouvelle tâche",
        description: "",
        importance: "Normale",
        completed: false,
        cancelled: false,
        dueDate: null,
        orderIndex: maxOrder + 1,
      };
      let next = [...prev, nt];

      if (parentId) {
        // Une sous-tâche fraîchement ajoutée est active par nature : ses ancêtres
        // ne peuvent donc plus rester marqués terminés/annulés. On remonte la
        // chaîne jusqu'au premier ancêtre déjà actif (pas besoin d'aller plus haut).
        const byId = new Map(next.map((t) => [t.id, t]));
        let cur = byId.get(parentId);
        while (cur && (cur.completed || cur.cancelled)) {
          const updated = { ...cur, completed: false, cancelled: false };
          byId.set(cur.id, updated);
          cur = cur.parentId ? byId.get(cur.parentId) : null;
        }
        next = next.map((t) => byId.get(t.id) || t);
      }

      setSelectedTaskId(nt.id);
      if (parentId) setExpandedTaskIds((ex) => new Set(ex).add(parentId));
      return next;
    });
  }, []);

  const deleteTaskCascade = useCallback((id) => {
    setTasks((prev) => {
      const toRemove = new Set([id, ...getDescendantIds(prev, id)]);
      return prev.filter((t) => !toRemove.has(t.id));
    });
    setSelectedTaskId((s) => (s === id ? null : s));
    setConfirmDeleteTaskId(null);
  }, []);

  const toggleTaskCompleted = useCallback((id) => {
    setTasks((prev) => {
      const map = new Map(prev.map((t) => [t.id, { ...t }]));
      const target = map.get(id);
      const newVal = !target.completed;
      const descendants = getDescendantIds(prev, id);
      for (const t of map.values()) {
        if (t.id === id || descendants.has(t.id)) {
          t.completed = newVal;
          if (newVal) t.cancelled = false;
        }
      }
      let cur = target.parentId;
      while (cur) {
        const parent = map.get(cur);
        const kids = prev.filter((t) => t.parentId === cur).map((t) => map.get(t.id));
        const allDone = kids.length > 0 && kids.every((k) => k.completed);
        parent.completed = allDone;
        if (!allDone) parent.cancelled = false;
        cur = parent.parentId;
      }
      return Array.from(map.values());
    });
  }, []);

  const toggleTaskCancelled = useCallback(
    (id) => {
      updateTask(id, { cancelled: !tasksById.get(id).cancelled, completed: false });
    },
    [tasksById, updateTask]
  );

  const moveTask = useCallback((id, direction) => {
    setTasks((prev) => {
      const task = prev.find((t) => t.id === id);
      const siblings = prev
        .filter((t) => (t.parentId || null) === (task.parentId || null) && t.projectId === task.projectId)
        .sort((a, b) => a.orderIndex - b.orderIndex);
      const idx = siblings.findIndex((t) => t.id === id);
      let newIdx = idx;
      if (direction === "up") newIdx = Math.max(0, idx - 1);
      if (direction === "down") newIdx = Math.min(siblings.length - 1, idx + 1);
      if (direction === "top") newIdx = 0;
      if (direction === "bottom") newIdx = siblings.length - 1;
      if (newIdx === idx) return prev;
      const reordered = [...siblings];
      const [moved] = reordered.splice(idx, 1);
      reordered.splice(newIdx, 0, moved);
      reordered.forEach((t, i) => (t.orderIndex = i));
      const byId = new Map(reordered.map((t) => [t.id, t]));
      return prev.map((t) => byId.get(t.id) || t);
    });
  }, []);

  const reparentTask = useCallback(
    (draggedId, targetId) => {
      if (draggedId === targetId) return;
      const descendants = getDescendantIds(tasks, draggedId);
      if (descendants.has(targetId)) {
        window.alert("Déplacement impossible : une tâche ne peut pas devenir sous-tâche d'un de ses propres descendants.");
        return;
      }
      setTasks((prev) => {
        const target = prev.find((t) => t.id === targetId);
        const dragged = prev.find((t) => t.id === draggedId);
        const siblings = prev.filter((t) => t.parentId === targetId);
        const maxOrder = siblings.length ? Math.max(...siblings.map((s) => s.orderIndex)) : -1;
        let next = prev.map((t) =>
          t.id === draggedId ? { ...t, parentId: targetId, projectId: target.projectId, orderIndex: maxOrder + 1 } : t
        );

        if (dragged && !dragged.completed && !dragged.cancelled) {
          // Même logique que pour l'ajout : une tâche active déplacée sous un parent
          // terminé/annulé le réactive, en cascade jusqu'au premier ancêtre déjà actif.
          const byId = new Map(next.map((t) => [t.id, t]));
          let cur = byId.get(targetId);
          while (cur && (cur.completed || cur.cancelled)) {
            const updated = { ...cur, completed: false, cancelled: false };
            byId.set(cur.id, updated);
            cur = cur.parentId ? byId.get(cur.parentId) : null;
          }
          next = next.map((t) => byId.get(t.id) || t);
        }

        return next;
      });
      setExpandedTaskIds((ex) => new Set(ex).add(targetId));
    },
    [tasks]
  );

  // Depuis Priorités : ouvre la tâche dans l'onglet Tâches, quel que soit son projet
  const navigateToTask = useCallback(
    (task) => {
      setSelectedProjectId(task.projectId);
      const byId = new Map(tasks.map((t) => [t.id, t]));
      const ancestors = [];
      let cur = task.parentId;
      while (cur) {
        ancestors.push(cur);
        cur = byId.get(cur)?.parentId;
      }
      setExpandedTaskIds((ex) => {
        const next = new Set(ex);
        ancestors.forEach((id) => next.add(id));
        return next;
      });
      // On complète les filtres existants (sans les écraser) pour garantir que la tâche
      // ciblée reste visible, tout en respectant les préférences de filtre de l'utilisateur.
      setTaskFilters((f) => ({
        ...f,
        status: new Set(f.status).add("active"),
        importance: new Set(f.importance).add(task.importance),
      }));
      setSelectedTaskId(task.id);
      setActiveTab("tasks");
      setMainView("projects");
    },
    [tasks]
  );

  const expandAllTasks = useCallback(() => {
    setExpandedTaskIds((ex) => {
      const next = new Set(ex);
      projectTasks.forEach((t) => next.add(t.id));
      return next;
    });
  }, [projectTasks]);

  const collapseAllTasks = useCallback(() => {
    setExpandedTaskIds((ex) => {
      const next = new Set(ex);
      projectTasks.forEach((t) => next.delete(t.id));
      return next;
    });
  }, [projectTasks]);

  const openTaskRootMenu = (e) => {
    e.preventDefault();
    setTaskContextMenu({ x: e.clientX, y: e.clientY, type: "root" });
  };
  const openTaskMenu = (e, taskId) => {
    e.preventDefault();
    e.stopPropagation();
    setSelectedTaskId(taskId);
    setTaskContextMenu({ x: e.clientX, y: e.clientY, type: "task", taskId });
  };

  const toggleTaskStatusFilter = (status) =>
    setTaskFilters((f) => {
      const next = new Set(f.status);
      if (next.has(status)) next.delete(status);
      else next.add(status);
      return { ...f, status: next.size ? next : f.status };
    });
  const toggleTaskImportanceFilter = (imp) =>
    setTaskFilters((f) => {
      const next = new Set(f.importance);
      if (next.has(imp)) next.delete(imp);
      else next.add(imp);
      return { ...f, importance: next };
    });
  const toggleTaskDueOnly = () => setTaskFilters((f) => ({ ...f, dueOnly: !f.dueOnly }));

  /* ----- Mutations Notes ----- */

  const updateNote = useCallback((id, patch) => {
    setNotes((prev) => prev.map((n) => (n.id === id ? { ...n, ...patch } : n)));
  }, []);
  const addNote = useCallback((projectId, title) => {
    setNotes((prev) => {
      const nn = { id: uid(), projectId, title: title || "Nouvelle note", content: "", updatedAt: new Date() };
      setSelectedNoteId(nn.id);
      return [...prev, nn];
    });
  }, []);
  const deleteNote = useCallback((id) => {
    setNotes((prev) => prev.filter((n) => n.id !== id));
    setSelectedNoteId((s) => (s === id ? null : s));
    setConfirmDeleteNoteId(null);
  }, []);

  const openNoteRootMenu = (e) => {
    e.preventDefault();
    setNoteContextMenu({ x: e.clientX, y: e.clientY, type: "root" });
  };
  const openNoteMenu = (e, noteId) => {
    e.preventDefault();
    e.stopPropagation();
    setSelectedNoteId(noteId);
    setNoteContextMenu({ x: e.clientX, y: e.clientY, type: "note", noteId });
  };
  const startRenameNote = (id) => {
    const n = notesById.get(id);
    setRenamingNoteId(id);
    setNoteRenameDraft(n?.title || "");
  };
  const commitRenameNote = () => {
    if (renamingNoteId) updateNote(renamingNoteId, { title: noteRenameDraft.trim() || "Sans titre" });
    setRenamingNoteId(null);
  };

  /* ----- Mutations Réunions ----- */

  const addMeeting = useCallback(
    (title) => {
      const trimmed = title.trim();
      if (!trimmed) return;
      const nm = { id: uid(), projectId: selectedProjectId, title: trimmed, createdAt: new Date(), updatedAt: new Date() };
      setMeetings((prev) => [...prev, nm]);
      setSelectedMeetingId(nm.id);
      setSelectedInstanceId(null);
    },
    [selectedProjectId]
  );

  const renameMeeting = useCallback((id, title) => {
    const trimmed = title.trim();
    if (!trimmed) return;
    setMeetings((prev) => prev.map((m) => (m.id === id ? { ...m, title: trimmed, updatedAt: new Date() } : m)));
  }, []);

  const deleteMeetingCascade = useCallback(
    (id) => {
      setMeetings((prev) => prev.filter((m) => m.id !== id));
      setInstances((prev) => prev.filter((i) => i.meetingId !== id));
      setSelectedMeetingId((cur) => (cur === id ? null : cur));
      setSelectedInstanceId((cur) => {
        const stillValid = instances.find((i) => i.id === cur && i.meetingId !== id);
        return stillValid ? cur : null;
      });
      setConfirmDeleteMeeting(null);
    },
    [instances]
  );

  const selectMeeting = useCallback(
    (id) => {
      setSelectedMeetingId(id);
      const latest = instances.filter((i) => i.meetingId === id).sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())[0];
      setSelectedInstanceId(latest ? latest.id : null);
    },
    [instances]
  );

  /* ----- Recherche ----- */

  const openSearchResult = useCallback(
    (r) => {
      setSearchQuery(""); // ferme la recherche, revient à l'onglet précédent
      if (r.type === "task") {
        navigateToTask(r.task);
      } else if (r.type === "note") {
        setSelectedProjectId(r.projectId);
        setSelectedNoteId(r.id);
        setActiveTab("notes");
        setMainView("projects");
      } else if (r.type === "meeting") {
        setSelectedProjectId(r.projectId);
        selectMeeting(r.meeting.id);
        if (r.instance) setSelectedInstanceId(r.instance.id);
        setActiveTab("meetings");
        setMainView("projects");
      }
    },
    [navigateToTask, selectMeeting]
  );

  const addInstance = useCallback(() => {
    if (!selectedMeetingId) return;
    const now2 = new Date();
    const ni = { id: uid(), meetingId: selectedMeetingId, notes: "", timestamp: now2, createdAt: now2, updatedAt: now2 };
    setInstances((prev) => [...prev, ni]);
    setMeetings((prev) => prev.map((m) => (m.id === selectedMeetingId ? { ...m, updatedAt: now2 } : m)));
    setSelectedInstanceId(ni.id);
  }, [selectedMeetingId]);

  const updateInstance = useCallback((id, patch) => {
    setInstances((prev) => prev.map((i) => (i.id === id ? { ...i, ...patch } : i)));
  }, []);

  const deleteInstance = useCallback(
    (id) => {
      setInstances((prev) => prev.filter((i) => i.id !== id));
      setSelectedInstanceId((cur) => {
        if (cur !== id) return cur;
        const remaining = instances.filter((i) => i.meetingId === selectedMeetingId && i.id !== id).sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime());
        return remaining[0]?.id || null;
      });
      setConfirmDeleteInstance(null);
    },
    [instances, selectedMeetingId]
  );

  const openMeetingRootMenu = (e) => {
    e.preventDefault();
    setMeetingContextMenu({ x: e.clientX, y: e.clientY, type: "meetingRoot" });
  };
  const openMeetingMenu = (e, id) => {
    e.preventDefault();
    e.stopPropagation();
    selectMeeting(id);
    setMeetingContextMenu({ x: e.clientX, y: e.clientY, type: "meeting", id });
  };
  const openInstanceRootMenu = (e) => {
    e.preventDefault();
    if (!selectedMeetingId) return; // aucune réunion sélectionnée : le clic droit ne fait rien
    setMeetingContextMenu({ x: e.clientX, y: e.clientY, type: "instanceRoot" });
  };
  const openInstanceMenu = (e, id) => {
    e.preventDefault();
    e.stopPropagation();
    setSelectedInstanceId(id);
    setMeetingContextMenu({ x: e.clientX, y: e.clientY, type: "instance", id });
  };
  const startRenameMeeting = (id) => {
    const m = meetingsById.get(id);
    setRenamingMeetingId(id);
    setMeetingRenameDraft(m?.title || "");
  };
  const commitRenameMeeting = () => {
    if (renamingMeetingId) renameMeeting(renamingMeetingId, meetingRenameDraft);
    setRenamingMeetingId(null);
  };

  /* ----- Rendu ----- */

  return (
    <div style={styles.app}>
      <style>{fontImport}</style>
      <header style={styles.header}>
        <div style={styles.headerLeft}>
          <span style={styles.logoMark} aria-hidden="true">⌘</span>
          <div>
            <div style={styles.title}>Tout Doux</div>
            <div style={styles.subtitle}>Prototype unifié</div>
          </div>
        </div>
        <div style={styles.headerRight}>
          <span style={styles.projectPill}>{selectedProject?.name}</span>
        </div>
      </header>

      <div style={styles.topBarRow}>
        <div style={styles.mainTabsGroup}>
          <button style={mainView === "projects" ? styles.tabActive : styles.tabInactive} onClick={() => setMainView("projects")}>
            Projets
          </button>
          <button style={mainView === "priorities" ? styles.tabActive : styles.tabInactive} onClick={() => setMainView("priorities")}>
            Priorités
          </button>
        </div>

        <div style={styles.searchBarInline}>
          <span style={styles.searchIcon} aria-hidden="true">🔍</span>
          <input
            style={styles.searchInput}
            placeholder="Rechercher dans les tâches, notes, réunions…"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          {searchQuery && (
            <button style={styles.searchClearBtn} title="Effacer" onClick={() => setSearchQuery("")}>
              ✕
            </button>
          )}
        </div>
      </div>

      <div style={styles.body}>
        {searchActive ? (
          <SearchResultsView
            results={searchResults}
            query={debouncedSearchQuery}
            selectedKey={selectedSearchResultKey}
            onSelect={setSelectedSearchResultKey}
            onOpen={openSearchResult}
            onClose={() => setSearchQuery("")}
            projectsById={projectsById}
            resultsWidth={searchResultsWidth}
            onResultsResizeStart={onSearchResultsResizeStart}
          />
        ) : mainView === "priorities" ? (
          <PrioritiesTab
            criticalOrHighTasks={criticalOrHighTasks}
            dueSoonTasks={dueSoonTasks}
            projectsById={projectsById}
            now={now}
            onNavigate={navigateToTask}
            selectedPriorityTaskId={selectedPriorityTaskId}
            setSelectedPriorityTaskId={setSelectedPriorityTaskId}
            tasksById={tasksById}
            detailWidth={priorityDetailWidth}
            onDetailResizeStart={onPriorityDetailResizeStart}
          />
        ) : (
          <>
            <ProjectSidebar
              projects={sortedProjects}
              tasks={tasks}
              notes={notes}
              meetings={meetings}
              now={now}
              selectedProjectId={selectedProjectId}
              onSelect={selectProject}
              onAdd={addProject}
              onRename={renameProject}
              onRequestDelete={(id) => setConfirmDeleteProjectId(id)}
              width={sidebarWidth}
            />

            <ResizeHandle onMouseDown={onSidebarResizeStart} />

            <div style={styles.contentColumn}>
              <div style={styles.tabsRow}>
                <button style={activeTab === "tasks" ? styles.tabActive : styles.tabInactive} onClick={() => setActiveTab("tasks")}>
                  Tâches
                </button>
                <button style={activeTab === "notes" ? styles.tabActive : styles.tabInactive} onClick={() => setActiveTab("notes")}>
                  Notes
                </button>
                <button style={activeTab === "meetings" ? styles.tabActive : styles.tabInactive} onClick={() => setActiveTab("meetings")}>
                  Réunions
                </button>
              </div>

              <div style={styles.tabContent}>
                {activeTab === "tasks" && (
                  <TasksTab
                    now={now}
                    filters={taskFilters}
                    onToggleStatus={toggleTaskStatusFilter}
                    onToggleImportance={toggleTaskImportanceFilter}
                    onToggleDueOnly={toggleTaskDueOnly}
                    rootTasks={rootTasks}
                    childrenMap={taskChildrenMap}
                    visibleIds={visibleTaskIds}
                    expanded={expandedTaskIds}
                    setExpanded={setExpandedTaskIds}
                    selectedId={selectedTaskId}
                    setSelectedId={setSelectedTaskId}
                    toggleCompleted={toggleTaskCompleted}
                    dragId={dragTaskId}
                    setDragId={setDragTaskId}
                    onDrop={reparentTask}
                    onContextMenu={openTaskMenu}
                    onRootContextMenu={openTaskRootMenu}
                    selected={selectedTask}
                    onChangeTask={(patch) => selectedTask && updateTask(selectedTask.id, patch)}
                    onToggleCancel={() => selectedTask && toggleTaskCancelled(selectedTask.id)}
                    detailWidth={taskDetailWidth}
                    onDetailResizeStart={onTaskDetailResizeStart}
                  />
                )}

                {activeTab === "notes" && (
                  <NotesTab
                    now={now}
                    projectNotes={projectNotes}
                    selectedNoteId={selectedNoteId}
                    setSelectedNoteId={setSelectedNoteId}
                    onRootContextMenu={openNoteRootMenu}
                    onContextMenu={openNoteMenu}
                    renamingNoteId={renamingNoteId}
                    renameDraft={noteRenameDraft}
                    setRenameDraft={setNoteRenameDraft}
                    commitRename={commitRenameNote}
                    selectedNote={selectedNote}
                    onChangeNote={(patch) => selectedNote && updateNote(selectedNote.id, patch)}
                    listWidth={noteListWidth}
                    onListResizeStart={onNoteListResizeStart}
                  />
                )}

                {activeTab === "meetings" && (
                  <MeetingsTab
                    projectMeetings={projectMeetings}
                    instances={instances}
                    meetingInstances={meetingInstances}
                    selectedMeetingId={selectedMeetingId}
                    selectedInstanceId={selectedInstanceId}
                    selectedMeeting={selectedMeeting}
                    selectedInstance={selectedInstance}
                    selectMeeting={selectMeeting}
                    setSelectedInstanceId={setSelectedInstanceId}
                    onMeetingRootMenu={openMeetingRootMenu}
                    onMeetingMenu={openMeetingMenu}
                    onInstanceRootMenu={openInstanceRootMenu}
                    onInstanceMenu={openInstanceMenu}
                    renamingMeetingId={renamingMeetingId}
                    renameDraft={meetingRenameDraft}
                    setRenameDraft={setMeetingRenameDraft}
                    commitRename={commitRenameMeeting}
                    onChangeInstance={(patch) => selectedInstance && updateInstance(selectedInstance.id, patch)}
                    meetingsColWidth={meetingsColWidth}
                    onMeetingsResizeStart={onMeetingsResizeStart}
                    instancesColWidth={instancesColWidth}
                    onInstancesResizeStart={onInstancesResizeStart}
                  />
                )}
              </div>
            </div>
          </>
        )}
      </div>

      {/* ----- Modales & menus contextuels : Tâches ----- */}
      {creatingTask && (
        <CreateModal
          title={creatingTask.parentId ? `Nouvelle sous-tâche de « ${tasksById.get(creatingTask.parentId)?.name} »` : "Nouvelle tâche"}
          placeholder="Nom de la tâche"
          onCancel={() => setCreatingTask(null)}
          onConfirm={(name) => {
            addTask(creatingTask.parentId, name, creatingTask.projectId);
            setCreatingTask(null);
          }}
        />
      )}
      {confirmDeleteTaskId && (
        <ConfirmModal
          title={`Supprimer « ${tasksById.get(confirmDeleteTaskId)?.name} » ?`}
          text={(() => {
            const count = getDescendantIds(tasks, confirmDeleteTaskId).size;
            return count > 0 ? `Cette tâche a ${count} sous-tâche${count > 1 ? "s" : ""}. Tout sera supprimé.` : "Cette action est irréversible.";
          })()}
          onCancel={() => setConfirmDeleteTaskId(null)}
          onConfirm={() => deleteTaskCascade(confirmDeleteTaskId)}
        />
      )}
      {taskContextMenu && (
        <TaskContextMenu
          menu={taskContextMenu}
          task={taskContextMenu.taskId ? tasksById.get(taskContextMenu.taskId) : null}
          onClose={() => setTaskContextMenu(null)}
          onAddRoot={() => setCreatingTask({ parentId: null, projectId: selectedProjectId })}
          onAddSubtask={(id) => {
            setExpandedTaskIds((ex) => new Set(ex).add(id));
            setCreatingTask({ parentId: id, projectId: tasksById.get(id).projectId });
          }}
          onDelete={(id) => setConfirmDeleteTaskId(id)}
          onToggleCancel={(id) => toggleTaskCancelled(id)}
          onMove={(id, dir) => moveTask(id, dir)}
          onExpandAll={expandAllTasks}
          onCollapseAll={collapseAllTasks}
        />
      )}

      {/* ----- Modales & menus contextuels : Notes ----- */}
      {noteContextMenu && (
        <NoteContextMenu
          menu={noteContextMenu}
          onClose={() => setNoteContextMenu(null)}
          onAddRoot={() => addNote(selectedProjectId, "Nouvelle note")}
          onRename={(id) => startRenameNote(id)}
          onDelete={(id) => setConfirmDeleteNoteId(id)}
        />
      )}
      {confirmDeleteNoteId && (
        <ConfirmModal
          title={`Supprimer « ${notesById.get(confirmDeleteNoteId)?.title} » ?`}
          text="Cette action est irréversible."
          onCancel={() => setConfirmDeleteNoteId(null)}
          onConfirm={() => deleteNote(confirmDeleteNoteId)}
        />
      )}

      {/* ----- Modales & menus contextuels : Réunions ----- */}
      {creatingMeeting && (
        <CreateModal
          title="Nouvelle réunion"
          placeholder="Titre de la réunion"
          onCancel={() => setCreatingMeeting(false)}
          onConfirm={(title) => {
            addMeeting(title);
            setCreatingMeeting(false);
          }}
        />
      )}
      {meetingContextMenu && (
        <MeetingContextMenu
          menu={meetingContextMenu}
          onClose={() => setMeetingContextMenu(null)}
          onAddMeeting={() => setCreatingMeeting(true)}
          onRenameMeeting={(id) => startRenameMeeting(id)}
          onDeleteMeeting={(id) => setConfirmDeleteMeeting(id)}
          onAddInstance={() => addInstance()}
          onDeleteInstance={(id) => setConfirmDeleteInstance(id)}
        />
      )}
      {confirmDeleteMeeting && (
        <ConfirmModal
          title={`Supprimer « ${meetingsById.get(confirmDeleteMeeting)?.title} » ?`}
          text={(() => {
            const count = instances.filter((i) => i.meetingId === confirmDeleteMeeting).length;
            return count > 0 ? `Cette réunion a ${count} instance${count > 1 ? "s" : ""}. Tout sera supprimé.` : "Cette action est irréversible.";
          })()}
          onCancel={() => setConfirmDeleteMeeting(null)}
          onConfirm={() => deleteMeetingCascade(confirmDeleteMeeting)}
        />
      )}
      {confirmDeleteInstance && (
        <ConfirmModal
          title={`Supprimer l'instance du ${formatTimestamp(instancesById.get(confirmDeleteInstance)?.timestamp || new Date())} ?`}
          text="Cette action est irréversible."
          onCancel={() => setConfirmDeleteInstance(null)}
          onConfirm={() => deleteInstance(confirmDeleteInstance)}
        />
      )}

      {/* ----- Modale : Projet (partagée) ----- */}
      {confirmDeleteProjectId && (
        <ConfirmModal
          title={`Supprimer le projet « ${projects.find((p) => p.id === confirmDeleteProjectId)?.name} » ?`}
          text={(() => {
            const t = tasks.filter((x) => x.projectId === confirmDeleteProjectId).length;
            const n = notes.filter((x) => x.projectId === confirmDeleteProjectId).length;
            const m = meetings.filter((x) => x.projectId === confirmDeleteProjectId).length;
            const total = t + n + m;
            return total > 0
              ? `Ce projet contient ${t} tâche(s), ${n} note(s) et ${m} réunion(s). Tout sera supprimé.`
              : "Cette action est irréversible.";
          })()}
          onCancel={() => setConfirmDeleteProjectId(null)}
          onConfirm={() => deleteProjectCascade(confirmDeleteProjectId)}
        />
      )}
    </div>
  );
}

/* ============================================================
   Sidebar projets (partagée entre onglets)
   ============================================================ */

function ProjectSidebar({ projects, tasks, notes, meetings, now, selectedProjectId, onSelect, onAdd, onRename, onRequestDelete, width }) {
  const [renamingId, setRenamingId] = useState(null);
  const [renameDraft, setRenameDraft] = useState("");
  const [adding, setAdding] = useState(false);
  const [addDraft, setAddDraft] = useState("");

  const startRename = (p) => {
    setRenamingId(p.id);
    setRenameDraft(p.name);
  };
  const commitRename = () => {
    if (renamingId) onRename(renamingId, renameDraft);
    setRenamingId(null);
  };
  const commitAdd = () => {
    if (addDraft.trim()) onAdd(addDraft);
    setAddDraft("");
    setAdding(false);
  };

  return (
    <div style={{ ...styles.projSidebar, flex: `0 0 ${width}px`, width }}>
      <div style={styles.projSidebarLabel}>Projets</div>
      <div style={styles.projList}>
        {projects.map((p) => {
          const stats = getSidebarStats(tasks, notes, meetings, p.id, now);
          const isSelected = p.id === selectedProjectId;
          const rowStyle = p.locked
            ? styles.projRowLocked
            : stats.hasCritical
            ? { background: IMPORTANCE_STYLE.Critique.bg, color: IMPORTANCE_STYLE.Critique.text, border: `1px solid ${IMPORTANCE_STYLE.Critique.border}` }
            : stats.hasHigh
            ? { background: IMPORTANCE_STYLE.Haute.bg, color: IMPORTANCE_STYLE.Haute.text, border: `1px solid ${IMPORTANCE_STYLE.Haute.border}` }
            : styles.projRowNormal;

          return (
            <div key={p.id} style={{ ...styles.projRow, ...rowStyle, ...(isSelected ? styles.projRowSelected : {}) }} onClick={() => onSelect(p.id)}>
              {renamingId === p.id ? (
                <input
                  autoFocus
                  style={styles.renameInput}
                  value={renameDraft}
                  onClick={(e) => e.stopPropagation()}
                  onChange={(e) => setRenameDraft(e.target.value)}
                  onBlur={commitRename}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") commitRename();
                    if (e.key === "Escape") setRenamingId(null);
                  }}
                />
              ) : (
                <div
                  style={styles.projName}
                  onDoubleClick={(e) => {
                    e.stopPropagation();
                    if (!p.locked) startRename(p);
                  }}
                  title={p.locked ? "Projet verrouillé" : "Double-clic pour renommer"}
                >
                  {p.name}
                </div>
              )}

              <div style={styles.projMeta}>
                <span style={styles.projCounts}>
                  {stats.activeCount} active{stats.activeCount > 1 ? "s" : ""}
                </span>
                {stats.dueIcon && <span aria-hidden="true">{stats.dueIcon}</span>}
                {!p.locked && (
                  <button style={styles.projDeleteBtn} title="Supprimer le projet" onClick={(e) => { e.stopPropagation(); onRequestDelete(p.id); }}>
                    ✕
                  </button>
                )}
              </div>
              <div style={styles.projSubMeta}>
                {stats.notesCount} note{stats.notesCount > 1 ? "s" : ""} · {stats.meetingsCount} réunion{stats.meetingsCount > 1 ? "s" : ""}
              </div>
            </div>
          );
        })}
      </div>

      {adding ? (
        <input
          autoFocus
          style={styles.renameInput}
          placeholder="Nom du projet…"
          value={addDraft}
          onChange={(e) => setAddDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") commitAdd();
            if (e.key === "Escape") {
              setAdding(false);
              setAddDraft("");
            }
          }}
          onBlur={commitAdd}
        />
      ) : (
        <button style={styles.projAddBtn} onClick={() => setAdding(true)}>
          + Projet
        </button>
      )}
    </div>
  );
}

/* ============================================================
   Onglet Tâches
   ============================================================ */

function TasksTab({
  now,
  filters,
  onToggleStatus,
  onToggleImportance,
  onToggleDueOnly,
  rootTasks,
  childrenMap,
  visibleIds,
  expanded,
  setExpanded,
  selectedId,
  setSelectedId,
  toggleCompleted,
  dragId,
  setDragId,
  onDrop,
  onContextMenu,
  onRootContextMenu,
  selected,
  onChangeTask,
  onToggleCancel,
  detailWidth,
  onDetailResizeStart,
}) {
  return (
    <>
      <div style={styles.leftCol}>
        <FilterBar filters={filters} onStatus={onToggleStatus} onImportance={onToggleImportance} onDueOnly={onToggleDueOnly} />
        <div style={styles.treeWrap} onContextMenu={onRootContextMenu}>
          {rootTasks.length === 0 && <div style={styles.emptyState}>Aucune tâche. Clic droit ici pour en créer une.</div>}
          {rootTasks.map((t) => (
            <TaskNode
              key={t.id}
              task={t}
              depth={0}
              childrenMap={childrenMap}
              visibleIds={visibleIds}
              expanded={expanded}
              setExpanded={setExpanded}
              selectedId={selectedId}
              setSelectedId={setSelectedId}
              now={now}
              toggleCompleted={toggleCompleted}
              dragId={dragId}
              setDragId={setDragId}
              onDrop={onDrop}
              onContextMenu={onContextMenu}
            />
          ))}
        </div>
      </div>

      <ResizeHandle onMouseDown={onDetailResizeStart} />

      <div style={{ ...styles.rightCol, flex: `0 0 ${detailWidth}px`, width: detailWidth }}>
        {selected ? (
          <DetailPanel key={selected.id} task={selected} now={now} onChange={onChangeTask} onToggleCancel={onToggleCancel} />
        ) : (
          <div style={styles.detailEmpty}>Sélectionne une tâche pour voir son détail, ou crée-en une nouvelle.</div>
        )}
      </div>
    </>
  );
}

const STATUS_ORDER = ["active", "completed", "cancelled"];
const STATUS_LABELS = { active: "Actives", completed: "Complétées", cancelled: "Annulées" };

function ImportanceDot({ color, active, size = 16 }) {
  return (
    <span
      style={{
        display: "inline-block",
        width: size,
        height: size,
        borderRadius: "50%",
        background: color,
        border: `${active ? 3 : 1.25}px solid #1c2321`,
        boxSizing: "border-box",
      }}
    />
  );
}

function StatusIcon({ status, active }) {
  const bw = active ? 3 : 1.25;
  if (status === "active") {
    return <span style={{ ...styles.statusIconBox, borderWidth: bw }} />;
  }
  if (status === "completed") {
    return (
      <span
        style={{
          ...styles.statusIconBox,
          borderWidth: bw,
          background: STATUS_STYLE.completed.bg,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <span style={{ fontSize: 9, color: STATUS_STYLE.completed.text, lineHeight: 1 }}>✓</span>
      </span>
    );
  }
  // cancelled : mini représentation "case + texte barré" comme une tâche annulée
  return (
    <span style={{ display: "flex", alignItems: "center", gap: 4 }}>
      <span style={{ ...styles.statusIconBox, borderWidth: bw }} />
      <span style={{ fontSize: 11, textDecoration: "line-through", color: "#6b6b6b" }}>abc</span>
    </span>
  );
}

function FilterBar({ filters, onStatus, onImportance, onDueOnly }) {
  return (
    <div style={styles.filterBarRow}>
      <div style={styles.filterInlineGroup} title="Statut">
        {STATUS_ORDER.map((s) => (
          <button key={s} onClick={() => onStatus(s)} style={styles.iconChipBtn} title={STATUS_LABELS[s]}>
            <StatusIcon status={s} active={filters.status.has(s)} />
          </button>
        ))}
      </div>
      <span style={styles.filterDivider} />
      <div style={styles.filterInlineGroup} title="Importance">
        {IMPORTANCES.map((imp) => (
          <button key={imp} onClick={() => onImportance(imp)} style={styles.iconChipBtn} title={imp}>
            <ImportanceDot color={IMPORTANCE_STYLE[imp].bg} active={filters.importance.has(imp)} />
          </button>
        ))}
      </div>
      <span style={styles.filterDivider} />
      <button onClick={onDueOnly} style={styles.iconChipBtn} title="Avec échéance uniquement">
        <span style={{ ...styles.dueIconInner, borderWidth: filters.dueOnly ? 3 : 1.25 }}>🕐</span>
      </button>
    </div>
  );
}

function TaskNode({ task, depth, childrenMap, visibleIds, expanded, setExpanded, selectedId, setSelectedId, now, toggleCompleted, dragId, setDragId, onDrop, onContextMenu }) {
  if (!visibleIds.has(task.id)) return null;

  const kids = childrenMap.get(task.id) || [];
  const hasKids = kids.length > 0;
  const isExpanded = expanded.has(task.id);
  const isSelected = selectedId === task.id;

  const status = task.cancelled ? "cancelled" : task.completed ? "completed" : null;
  const style = status ? STATUS_STYLE[status] : IMPORTANCE_STYLE[task.importance];

  const dueInfo = !task.completed && !task.cancelled ? formatDueDate(task.dueDate, now) : null;
  const icon = dueInfo?.urgent || dueInfo?.overdue ? "⏰" : dueInfo ? "🕐" : null;

  return (
    <div>
      <div
        draggable
        onDragStart={(e) => {
          e.stopPropagation();
          setDragId(task.id);
        }}
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault();
          e.stopPropagation();
          if (dragId) onDrop(dragId, task.id);
          setDragId(null);
        }}
        onClick={() => setSelectedId(task.id)}
        onContextMenu={(e) => onContextMenu(e, task.id)}
        style={{
          ...styles.taskRow,
          marginLeft: depth * 20,
          background: style.bg,
          color: style.text,
          border: `1px solid ${isSelected ? "#3f6656" : style.border}`,
          outline: isSelected ? "2px solid #3f6656" : "none",
          outlineOffset: -1,
          textDecoration: task.cancelled ? "line-through" : "none",
        }}
      >
        <button
          aria-label={isExpanded ? "Réduire" : "Déplier"}
          onClick={(e) => {
            e.stopPropagation();
            setExpanded((ex) => {
              const next = new Set(ex);
              if (next.has(task.id)) next.delete(task.id);
              else next.add(task.id);
              return next;
            });
          }}
          style={{ ...styles.expandBtn, visibility: hasKids ? "visible" : "hidden", color: style.text }}
        >
          {isExpanded ? "▾" : "▸"}
        </button>

        <input type="checkbox" checked={task.completed} onClick={(e) => e.stopPropagation()} onChange={() => toggleCompleted(task.id)} style={{ marginRight: 4 }} />

        <span style={styles.taskName}>
          {task.name}
          {dueInfo && <span style={styles.dueText}> ({dueInfo.text})</span>}
        </span>

        {icon && <span aria-hidden="true">{icon}</span>}
      </div>

      {hasKids && isExpanded && (
        <div>
          {kids.map((k) => (
            <TaskNode
              key={k.id}
              task={k}
              depth={depth + 1}
              childrenMap={childrenMap}
              visibleIds={visibleIds}
              expanded={expanded}
              setExpanded={setExpanded}
              selectedId={selectedId}
              setSelectedId={setSelectedId}
              now={now}
              toggleCompleted={toggleCompleted}
              dragId={dragId}
              setDragId={setDragId}
              onDrop={onDrop}
              onContextMenu={onContextMenu}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function DetailPanel({ task, now, onChange, onToggleCancel }) {
  const [localName, setLocalName] = useState(task.name);
  const [localDesc, setLocalDesc] = useState(task.description || "");
  const debTimer = useRef(null);
  const dirtyRef = useRef(false);
  const latestRef = useRef({ name: localName, desc: localDesc });
  const onChangeRef = useRef(onChange);

  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);
  useEffect(() => {
    latestRef.current = { name: localName, desc: localDesc };
  }, [localName, localDesc]);

  const commit = useCallback(() => {
    if (debTimer.current) {
      clearTimeout(debTimer.current);
      debTimer.current = null;
    }
    if (!dirtyRef.current) return;
    onChangeRef.current({ name: latestRef.current.name, description: latestRef.current.desc });
    dirtyRef.current = false;
  }, []);

  useEffect(() => {
    if (localName === task.name && localDesc === (task.description || "")) return;
    dirtyRef.current = true;
    if (debTimer.current) clearTimeout(debTimer.current);
    debTimer.current = setTimeout(commit, 500);
    return () => clearTimeout(debTimer.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [localName, localDesc]);

  useEffect(() => {
    return () => commit();
  }, [commit]);

  const dueInfo = formatDueDate(task.dueDate, now);

  return (
    <div style={styles.detailPanel}>
      <label style={styles.fieldLabel}>Nom</label>
      <input style={styles.input} value={localName} onChange={(e) => setLocalName(e.target.value)} onBlur={commit} />

      <label style={styles.fieldLabel}>Description</label>
      <textarea
        style={styles.textarea}
        rows={4}
        placeholder="Description (éditeur riche simplifié pour ce prototype)"
        value={localDesc}
        onChange={(e) => setLocalDesc(e.target.value)}
        onBlur={commit}
      />

      <label style={styles.fieldLabel}>Importance</label>
      <div style={styles.filterInlineGroup}>
        {IMPORTANCES.map((imp) => {
          const st = IMPORTANCE_STYLE[imp];
          const active = task.importance === imp;
          return (
            <button
              key={imp}
              onClick={() => onChange({ importance: imp })}
              style={{
                fontFamily: "inherit",
                fontSize: 12,
                padding: "5px 10px",
                borderRadius: 20,
                background: st.bg,
                color: st.text,
                border: `${active ? 3 : 1}px solid #1c2321`,
                cursor: "pointer",
              }}
            >
              {imp}
            </button>
          );
        })}
      </div>

      <label style={styles.fieldLabel}>Échéance</label>
      {task.dueDate ? (
        <div style={styles.dueSummary}>
          <span>{task.dueDate.toLocaleString("fr-FR")}</span>
          {dueInfo && <span style={{ color: dueInfo.overdue ? "#a32d2d" : "#3f6656" }}> — {dueInfo.text}</span>}
          <button style={styles.removeDueBtn} title="Retirer l'échéance" onClick={() => onChange({ dueDate: null })}>
            ✕
          </button>
        </div>
      ) : (
        <div style={styles.dueSummary}>Aucune échéance</div>
      )}

      <div style={styles.quickDueRow}>
        {[
          ["now", "Tout de suite"],
          ["10min", "Dans 10 min"],
          ["1h", "Dans 1h"],
          ["journee", "Journée"],
          ["lendemain9h", "Lendemain 9h"],
        ].map(([key, label]) => {
          const disabled = key === "journee" && now.getHours() >= 17;
          return (
            <button
              key={key}
              disabled={disabled}
              title={disabled ? "Après 17h, utilise plutôt Lendemain 9h" : undefined}
              style={{ ...styles.chip, opacity: disabled ? 0.4 : 1 }}
              onClick={() => onChange({ dueDate: quickDueDate(key, now) })}
            >
              {label}
            </button>
          );
        })}
      </div>

      <div style={styles.customDueRow}>
        <span aria-hidden="true">📅</span>
        <input
          type="datetime-local"
          style={styles.dateTimeInput}
          value={task.dueDate ? toDatetimeLocalValue(task.dueDate) : ""}
          onChange={(e) => {
            if (!e.target.value) return;
            onChange({ dueDate: new Date(e.target.value) });
          }}
        />
      </div>

      <div style={styles.moveRow}>
        <button style={{ ...styles.btnSecondary, width: "100%" }} onClick={onToggleCancel}>
          {task.cancelled ? "Réactiver la tâche" : "Annuler la tâche"}
        </button>
      </div>
    </div>
  );
}

function TaskContextMenu({ menu, task, onAddRoot, onAddSubtask, onDelete, onToggleCancel, onMove, onExpandAll, onCollapseAll, onClose }) {
  const items =
    menu.type === "root"
      ? [
          { label: "+ Nouvelle tâche", onClick: onAddRoot },
          { divider: true },
          { label: "Tout étendre", onClick: onExpandAll },
          { label: "Tout réduire", onClick: onCollapseAll },
        ]
      : [
          { label: "+ Sous-tâche", onClick: () => onAddSubtask(menu.taskId) },
          { divider: true },
          { label: "⇑ Envoyer au début", onClick: () => onMove(menu.taskId, "top") },
          { label: "↑ Monter", onClick: () => onMove(menu.taskId, "up") },
          { label: "↓ Descendre", onClick: () => onMove(menu.taskId, "down") },
          { label: "⇓ Envoyer à la fin", onClick: () => onMove(menu.taskId, "bottom") },
          { divider: true },
          { label: "Tout étendre", onClick: onExpandAll },
          { label: "Tout réduire", onClick: onCollapseAll },
          { divider: true },
          { label: task?.cancelled ? "Réactiver" : "Annuler la tâche", onClick: () => onToggleCancel(menu.taskId) },
          { label: "Supprimer", danger: true, onClick: () => onDelete(menu.taskId) },
        ];
  return <GenericContextMenu items={items} x={menu.x} y={menu.y} onClose={onClose} />;
}

/* ============================================================
   Onglet Notes
   ============================================================ */

function NotesTab({
  now,
  projectNotes,
  selectedNoteId,
  setSelectedNoteId,
  onRootContextMenu,
  onContextMenu,
  renamingNoteId,
  renameDraft,
  setRenameDraft,
  commitRename,
  selectedNote,
  onChangeNote,
  listWidth,
  onListResizeStart,
}) {
  return (
    <>
      <div style={{ ...styles.noteListCol, flex: `0 0 ${listWidth}px`, width: listWidth }}>
        <div style={styles.listWrap} onContextMenu={onRootContextMenu}>
          {projectNotes.length === 0 && <div style={styles.emptyState}>Aucune note. Clic droit ici pour en créer une.</div>}
          {projectNotes.map((n) => (
            <div
              key={n.id}
              style={{ ...styles.rowItem, ...(selectedNoteId === n.id ? styles.rowItemSelected : {}) }}
              onClick={() => setSelectedNoteId(n.id)}
              onContextMenu={(e) => onContextMenu(e, n.id)}
            >
              {renamingNoteId === n.id ? (
                <input
                  autoFocus
                  style={styles.renameInput}
                  value={renameDraft}
                  onClick={(e) => e.stopPropagation()}
                  onChange={(e) => setRenameDraft(e.target.value)}
                  onBlur={commitRename}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") commitRename();
                    if (e.key === "Escape") commitRename();
                  }}
                />
              ) : (
                <>
                  <div style={styles.rowTitle}>{n.title || "Sans titre"}</div>
                  <div style={styles.rowMeta}>{previewText(n.content) || "Note vide"}</div>
                  <div style={styles.rowSubMeta}>{timeAgo(n.updatedAt, now)}</div>
                </>
              )}
            </div>
          ))}
        </div>
      </div>

      <ResizeHandle onMouseDown={onListResizeStart} />

      <div style={styles.editorCol}>
        {selectedNote ? (
          <NoteEditor key={selectedNote.id} note={selectedNote} onChange={onChangeNote} />
        ) : (
          <div style={styles.detailEmpty}>Sélectionne une note pour l'éditer, ou clic droit dans la liste pour en créer une.</div>
        )}
      </div>
    </>
  );
}

function NoteEditor({ note, onChange }) {
  const [localTitle, setLocalTitle] = useState(note.title);
  const [localContent, setLocalContent] = useState(note.content);
  const [status, setStatus] = useState("idle");
  const [savedAt, setSavedAt] = useState(note.updatedAt);
  const timer = useRef(null);
  const dirtyRef = useRef(false);
  const latestRef = useRef({ title: localTitle, content: localContent });
  const onChangeRef = useRef(onChange);

  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);
  useEffect(() => {
    latestRef.current = { title: localTitle, content: localContent };
  }, [localTitle, localContent]);

  const commit = useCallback(() => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = null;
    }
    if (!dirtyRef.current) return null;
    const savedNow = new Date();
    onChangeRef.current({ title: latestRef.current.title.trim() || "Sans titre", content: latestRef.current.content, updatedAt: savedNow });
    dirtyRef.current = false;
    return savedNow;
  }, []);

  const flush = useCallback(() => {
    const savedNow = commit();
    if (savedNow) {
      setStatus("saved");
      setSavedAt(savedNow);
    }
  }, [commit]);

  useEffect(() => {
    if (localTitle === note.title && localContent === note.content) return;
    dirtyRef.current = true;
    setStatus("dirty");
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(flush, 2000);
    return () => clearTimeout(timer.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [localTitle, localContent]);

  useEffect(() => {
    return () => commit();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [commit]);

  return (
    <div style={styles.editorPane}>
      <input style={styles.noteTitleInput} value={localTitle} onChange={(e) => setLocalTitle(e.target.value)} onBlur={flush} placeholder="Titre de la note" />
      <textarea
        style={styles.noteContentArea}
        value={localContent}
        onChange={(e) => setLocalContent(e.target.value)}
        onBlur={flush}
        placeholder="Contenu de la note (éditeur riche simplifié pour ce prototype)…"
      />
      <div style={styles.saveStatus}>
        {status === "dirty" && "Modifications en cours…"}
        {status === "saved" && `Note sauvegardée à ${savedAt.toLocaleTimeString("fr-FR")}`}
        {status === "idle" && note.updatedAt && `Dernière sauvegarde à ${note.updatedAt.toLocaleTimeString("fr-FR")}`}
      </div>
    </div>
  );
}

function NoteContextMenu({ menu, onAddRoot, onRename, onDelete, onClose }) {
  const items =
    menu.type === "root"
      ? [{ label: "+ Nouvelle note", onClick: onAddRoot }]
      : [
          { label: "Renommer", onClick: () => onRename(menu.noteId) },
          { label: "Supprimer", danger: true, onClick: () => onDelete(menu.noteId) },
        ];
  return <GenericContextMenu items={items} x={menu.x} y={menu.y} onClose={onClose} />;
}

/* ============================================================
   Onglet Réunions
   ============================================================ */

function MeetingsTab({
  projectMeetings,
  instances,
  meetingInstances,
  selectedMeetingId,
  selectedInstanceId,
  selectedMeeting,
  selectedInstance,
  selectMeeting,
  setSelectedInstanceId,
  onMeetingRootMenu,
  onMeetingMenu,
  onInstanceRootMenu,
  onInstanceMenu,
  renamingMeetingId,
  renameDraft,
  setRenameDraft,
  commitRename,
  onChangeInstance,
  meetingsColWidth,
  onMeetingsResizeStart,
  instancesColWidth,
  onInstancesResizeStart,
}) {
  return (
    <>
      <div style={{ ...styles.listCol, flex: `0 0 ${meetingsColWidth}px`, width: meetingsColWidth }}>
        <div style={styles.listColLabel}>Réunions</div>
        <div style={styles.listWrap} onContextMenu={onMeetingRootMenu}>
          {projectMeetings.length === 0 && <div style={styles.emptyState}>Aucune réunion. Clic droit ici pour en créer une.</div>}
          {projectMeetings.map((m) => {
            const instCount = instances.filter((i) => i.meetingId === m.id).length;
            return (
              <div
                key={m.id}
                style={{ ...styles.rowItem, ...(selectedMeetingId === m.id ? styles.rowItemSelected : {}) }}
                onClick={() => selectMeeting(m.id)}
                onContextMenu={(e) => onMeetingMenu(e, m.id)}
              >
                {renamingMeetingId === m.id ? (
                  <input
                    autoFocus
                    style={styles.renameInput}
                    value={renameDraft}
                    onClick={(e) => e.stopPropagation()}
                    onChange={(e) => setRenameDraft(e.target.value)}
                    onBlur={commitRename}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") commitRename();
                      if (e.key === "Escape") commitRename();
                    }}
                  />
                ) : (
                  <>
                    <div style={styles.rowTitle}>{m.title}</div>
                    <div style={styles.rowMeta}>
                      {instCount} instance{instCount > 1 ? "s" : ""}
                    </div>
                  </>
                )}
              </div>
            );
          })}
        </div>
      </div>

      <ResizeHandle onMouseDown={onMeetingsResizeStart} />

      <div style={{ ...styles.listCol, flex: `0 0 ${instancesColWidth}px`, width: instancesColWidth }}>
        <div style={styles.listColLabel}>Instances</div>
        <div style={styles.listWrap} onContextMenu={onInstanceRootMenu}>
          {!selectedMeeting && <div style={styles.emptyState}>Sélectionne une réunion.</div>}
          {selectedMeeting && meetingInstances.length === 0 && <div style={styles.emptyState}>Aucune instance. Clic droit ici pour en créer une.</div>}
          {selectedMeeting &&
            meetingInstances.map((inst) => (
              <div
                key={inst.id}
                style={{ ...styles.rowItem, ...(selectedInstanceId === inst.id ? styles.rowItemSelected : {}) }}
                onClick={() => setSelectedInstanceId(inst.id)}
                onContextMenu={(e) => onInstanceMenu(e, inst.id)}
              >
                <div style={styles.rowTitle}>{formatTimestamp(inst.timestamp)}</div>
                <div style={styles.rowMeta}>{previewText(inst.notes) || "Notes vides"}</div>
              </div>
            ))}
        </div>
      </div>

      <ResizeHandle onMouseDown={onInstancesResizeStart} />

      <div style={styles.editorCol}>
        {selectedInstance ? (
          <InstanceEditor key={selectedInstance.id} instance={selectedInstance} onChange={onChangeInstance} />
        ) : (
          <div style={styles.detailEmpty}>
            {selectedMeeting ? "Sélectionne une instance, ou clic droit dans la colonne Instances pour en créer une." : "Sélectionne une réunion pour voir ses instances."}
          </div>
        )}
      </div>
    </>
  );
}

function InstanceEditor({ instance, onChange }) {
  const [localNotes, setLocalNotes] = useState(instance.notes);
  const [status, setStatus] = useState("idle");
  const [savedAt, setSavedAt] = useState(instance.updatedAt);
  const timer = useRef(null);
  const dirtyRef = useRef(false);
  const latestRef = useRef(localNotes);
  const onChangeRef = useRef(onChange);

  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);
  useEffect(() => {
    latestRef.current = localNotes;
  }, [localNotes]);

  const commit = useCallback(() => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = null;
    }
    if (!dirtyRef.current) return null;
    const savedNow = new Date();
    onChangeRef.current({ notes: latestRef.current, updatedAt: savedNow });
    dirtyRef.current = false;
    return savedNow;
  }, []);

  const flush = useCallback(() => {
    const savedNow = commit();
    if (savedNow) {
      setStatus("saved");
      setSavedAt(savedNow);
    }
  }, [commit]);

  useEffect(() => {
    if (localNotes === instance.notes) return;
    dirtyRef.current = true;
    setStatus("dirty");
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(flush, 2000);
    return () => clearTimeout(timer.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [localNotes]);

  useEffect(() => {
    return () => commit();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [commit]);

  return (
    <div style={styles.editorPane}>
      <div style={styles.instanceHeader}>Instance du {formatTimestamp(instance.timestamp)}</div>
      <textarea
        style={styles.noteContentArea}
        value={localNotes}
        onChange={(e) => setLocalNotes(e.target.value)}
        onBlur={flush}
        placeholder="Notes de la réunion (éditeur riche simplifié pour ce prototype)…"
      />
      <div style={styles.saveStatus}>
        {status === "dirty" && "Modifications en cours…"}
        {status === "saved" && `Notes sauvegardées à ${savedAt.toLocaleTimeString("fr-FR")}`}
        {status === "idle" && instance.updatedAt && `Dernière sauvegarde à ${instance.updatedAt.toLocaleTimeString("fr-FR")}`}
      </div>
    </div>
  );
}

function MeetingContextMenu({ menu, onAddMeeting, onRenameMeeting, onDeleteMeeting, onAddInstance, onDeleteInstance, onClose }) {
  let items = [];
  if (menu.type === "meetingRoot") items = [{ label: "+ Nouvelle réunion", onClick: onAddMeeting }];
  if (menu.type === "meeting")
    items = [
      { label: "Renommer", onClick: () => onRenameMeeting(menu.id) },
      { label: "Supprimer", danger: true, onClick: () => onDeleteMeeting(menu.id) },
    ];
  if (menu.type === "instanceRoot") items = [{ label: "+ Nouvelle instance", onClick: onAddInstance }];
  if (menu.type === "instance") items = [{ label: "Supprimer", danger: true, onClick: () => onDeleteInstance(menu.id) }];
  return <GenericContextMenu items={items} x={menu.x} y={menu.y} onClose={onClose} />;
}

/* ============================================================
   Onglet Priorités (transverse, tous projets)
   ============================================================ */

function PrioritiesTab({
  criticalOrHighTasks,
  dueSoonTasks,
  projectsById,
  now,
  onNavigate,
  selectedPriorityTaskId,
  setSelectedPriorityTaskId,
  tasksById,
  detailWidth,
  onDetailResizeStart,
}) {
  const selectedTask = selectedPriorityTaskId ? tasksById.get(selectedPriorityTaskId) : null;

  return (
    <>
      <div style={styles.prioritiesListCol}>
        <PrioritySection
          title="Tâches Critiques & Hautes"
          tasks={criticalOrHighTasks}
          projectsById={projectsById}
          now={now}
          onNavigate={onNavigate}
          selectedId={selectedPriorityTaskId}
          onSelect={setSelectedPriorityTaskId}
          emptyLabel="Aucune tâche critique ou haute active. 🎉"
        />
        <PrioritySection
          title="Échéances à venir"
          tasks={dueSoonTasks}
          projectsById={projectsById}
          now={now}
          onNavigate={onNavigate}
          selectedId={selectedPriorityTaskId}
          onSelect={setSelectedPriorityTaskId}
          emptyLabel="Aucune tâche active avec échéance."
        />
      </div>

      <ResizeHandle onMouseDown={onDetailResizeStart} />

      <div style={{ ...styles.rightCol, flex: `0 0 ${detailWidth}px`, width: detailWidth }}>
        {selectedTask ? (
          <TaskReadOnlyPanel task={selectedTask} now={now} projectName={projectsById.get(selectedTask.projectId)} onOpen={() => onNavigate(selectedTask)} />
        ) : (
          <div style={styles.detailEmpty}>Clic sur une tâche pour voir son détail, double-clic pour l'ouvrir dans l'onglet Tâches.</div>
        )}
      </div>
    </>
  );
}

function PrioritySection({ title, tasks, projectsById, now, onNavigate, selectedId, onSelect, emptyLabel }) {
  return (
    <div style={styles.prioritySection}>
      <div style={styles.prioritySectionTitle}>
        {title} ({tasks.length})
      </div>
      {tasks.length === 0 ? (
        <div style={styles.emptyState}>{emptyLabel}</div>
      ) : (
        tasks.map((t) => (
          <PriorityRow
            key={t.id}
            task={t}
            projectsById={projectsById}
            now={now}
            selected={selectedId === t.id}
            onClick={() => onSelect(t.id)}
            onDoubleClick={() => onNavigate(t)}
          />
        ))
      )}
    </div>
  );
}

function PriorityRow({ task, projectsById, now, selected, onClick, onDoubleClick }) {
  const style = IMPORTANCE_STYLE[task.importance];
  const dueInfo = formatDueDate(task.dueDate, now);
  return (
    <div
      style={{ ...styles.priorityRow, ...(selected ? styles.priorityRowSelected : {}) }}
      onClick={onClick}
      onDoubleClick={onDoubleClick}
      title="Clic : aperçu — Double-clic : ouvrir dans Tâches"
    >
      <ImportanceDot color={style.bg} active={false} size={13} />
      <span style={styles.priorityName}>{task.name}</span>
      <span style={styles.priorityProject}>{projectsById.get(task.projectId) || "?"}</span>
      <span style={{ ...styles.priorityDue, color: dueInfo?.overdue ? "#a32d2d" : "#5b6b65" }}>{task.dueDate ? dueInfo.text : "Sans échéance"}</span>
    </div>
  );
}

// Lecture seule : même structure visuelle que DetailPanel, sans aucune interaction d'édition
function TaskReadOnlyPanel({ task, now, projectName, onOpen }) {
  const dueInfo = formatDueDate(task.dueDate, now);
  const impStyle = IMPORTANCE_STYLE[task.importance];
  return (
    <div style={styles.detailPanel}>
      <div style={styles.readOnlyHeaderRow}>
        <span style={styles.fieldLabel}>{projectName}</span>
        <button style={styles.btnSecondary} onClick={onOpen}>
          Ouvrir dans Tâches
        </button>
      </div>

      <h3 style={styles.readOnlyName}>{task.name}</h3>

      <label style={styles.fieldLabel}>Importance</label>
      <span
        style={{
          display: "inline-flex",
          alignItems: "center",
          gap: 6,
          fontSize: 12,
          padding: "5px 10px",
          borderRadius: 20,
          background: impStyle.bg,
          color: impStyle.text,
          border: "1px solid #1c2321",
          width: "fit-content",
        }}
      >
        {task.importance}
      </span>

      <label style={styles.fieldLabel}>Échéance</label>
      <div style={styles.dueSummary}>
        {task.dueDate ? (
          <>
            <span>{task.dueDate.toLocaleString("fr-FR")}</span>
            {dueInfo && <span style={{ color: dueInfo.overdue ? "#a32d2d" : "#3f6656" }}> — {dueInfo.text}</span>}
          </>
        ) : (
          "Aucune échéance"
        )}
      </div>

      <label style={styles.fieldLabel}>Description</label>
      <div style={styles.readOnlyDesc}>{task.description ? task.description : <span style={{ color: "#9aa8a2" }}>Aucune description</span>}</div>
    </div>
  );
}

/* ============================================================
   Recherche globale
   ============================================================ */

function SearchResultsView({ results, query, selectedKey, onSelect, onOpen, onClose, projectsById, resultsWidth, onResultsResizeStart }) {
  const selected = results.find((r) => `${r.type}:${r.id}` === selectedKey) || null;

  return (
    <div style={styles.searchViewWrap}>
      <div style={styles.searchResultsHeader}>
        <span style={styles.searchResultsCount}>
          {results.length} résultat{results.length !== 1 ? "s" : ""}
        </span>
        <button style={styles.btnSecondary} onClick={onClose}>
          Fermer ⊗
        </button>
      </div>

      <div style={styles.tabContent}>
        <div style={{ ...styles.listCol, flex: `0 0 ${resultsWidth}px`, width: resultsWidth }}>
          <div style={styles.listWrap}>
            {results.length === 0 && <div style={styles.emptyState}>Aucun résultat pour « {query} ».</div>}
            {results.map((r) => {
              const key = `${r.type}:${r.id}`;
              return (
                <div
                  key={key}
                  style={{ ...styles.rowItem, ...(selectedKey === key ? styles.rowItemSelected : {}) }}
                  onClick={() => onSelect(key)}
                  onDoubleClick={() => onOpen(r)}
                >
                  <div style={styles.searchRowTop}>
                    {r.type === "task" ? (
                      <>
                        <input type="checkbox" checked={r.task.completed} disabled style={{ margin: 0 }} />
                        <ImportanceDot color={IMPORTANCE_STYLE[r.task.importance].bg} active={false} size={12} />
                      </>
                    ) : (
                      <span aria-hidden="true">{r.type === "note" ? "📝" : "📞"}</span>
                    )}
                    <span style={styles.rowTitle}>{r.title}</span>
                  </div>
                  <div style={styles.rowMeta}>{projectsById.get(r.projectId) || "?"}</div>
                </div>
              );
            })}
          </div>
        </div>

        <ResizeHandle onMouseDown={onResultsResizeStart} />

        <div style={styles.editorCol}>
          {selected ? (
            <SearchDetailPanel result={selected} query={query} projectsById={projectsById} onOpen={() => onOpen(selected)} />
          ) : (
            <div style={styles.detailEmpty}>Sélectionne un résultat pour voir le détail.</div>
          )}
        </div>
      </div>
    </div>
  );
}

function SearchDetailPanel({ result, query, projectsById, onOpen }) {
  const content =
    result.type === "note" ? result.note.content : result.type === "task" ? result.task.description || "" : result.instance ? result.instance.notes : "";
  const dateLabel = result.date ? result.date.toLocaleString("fr-FR") : null;

  return (
    <div style={styles.detailPanel}>
      <div style={styles.readOnlyHeaderRow}>
        <span style={styles.fieldLabel}>{projectsById.get(result.projectId)}</span>
        <button style={styles.btnSecondary} onClick={onOpen}>
          Ouvrir
        </button>
      </div>

      <h3 style={styles.readOnlyName}>{result.title}</h3>

      <label style={styles.fieldLabel}>Extrait</label>
      <div style={styles.readOnlyDesc}>
        {highlightSnippet(content, query) || <span style={{ color: "#9aa8a2" }}>Aucun contenu</span>}
      </div>

      {dateLabel && (
        <>
          <label style={styles.fieldLabel}>Date</label>
          <div style={styles.dueSummary}>{dateLabel}</div>
        </>
      )}
    </div>
  );
}

/* ============================================================
   Composants génériques (menu contextuel, modales)
   ============================================================ */

function GenericContextMenu({ items, x, y, onClose }) {
  return (
    <div style={{ ...styles.ctxMenu, top: y, left: x }} onClick={(e) => e.stopPropagation()} onContextMenu={(e) => e.preventDefault()}>
      {items.map((item, i) =>
        item.divider ? (
          <div key={i} style={styles.ctxDivider} />
        ) : (
          <button
            key={i}
            style={{ ...styles.ctxItem, ...(item.danger ? styles.ctxItemDanger : {}) }}
            onClick={() => {
              item.onClick();
              onClose();
            }}
          >
            {item.label}
          </button>
        )
      )}
    </div>
  );
}

function CreateModal({ title, placeholder, onCancel, onConfirm }) {
  const [value, setValue] = useState("");
  return (
    <div style={styles.overlay} onClick={onCancel}>
      <div style={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div style={styles.modalTitle}>{title}</div>
        <input
          autoFocus
          style={styles.input}
          placeholder={placeholder}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && value.trim()) onConfirm(value.trim());
            if (e.key === "Escape") onCancel();
          }}
        />
        <div style={styles.modalActions}>
          <button style={styles.btnSecondary} onClick={onCancel}>
            Annuler
          </button>
          <button style={styles.btnPrimary} disabled={!value.trim()} onClick={() => value.trim() && onConfirm(value.trim())}>
            Créer
          </button>
        </div>
      </div>
    </div>
  );
}

function ConfirmModal({ title, text, onCancel, onConfirm }) {
  return (
    <div style={styles.overlay} onClick={onCancel}>
      <div style={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div style={styles.modalTitle}>{title}</div>
        <p style={styles.modalText}>{text}</p>
        <div style={styles.modalActions}>
          <button style={styles.btnSecondary} onClick={onCancel}>
            Annuler
          </button>
          <button style={styles.btnDanger} onClick={onConfirm}>
            Supprimer
          </button>
        </div>
      </div>
    </div>
  );
}

/* ---------------- Styles ---------------- */

const fontImport = `@import url('https://fonts.googleapis.com/css2?family=Manrope:wght@500;700&family=Inter:wght@400;500&family=IBM+Plex+Mono:wght@400;500&display=swap');`;

const styles = {
  app: {
    fontFamily: "'Inter', system-ui, sans-serif",
    background: "#eff3f0",
    color: "#1c2321",
    minHeight: "640px",
    borderRadius: 12,
    overflow: "hidden",
    border: "1px solid #dce3df",
    position: "relative",
  },
  header: { display: "flex", alignItems: "center", justifyContent: "space-between", padding: "14px 20px", background: "#ffffff", borderBottom: "1px solid #dce3df" },
  headerLeft: { display: "flex", alignItems: "center", gap: 10 },
  logoMark: {
    width: 32,
    height: 32,
    borderRadius: 8,
    background: "#3f6656",
    color: "#ffffff",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontFamily: "'Manrope', sans-serif",
    fontWeight: 700,
  },
  title: { fontFamily: "'Manrope', sans-serif", fontWeight: 700, fontSize: 16 },
  subtitle: { fontSize: 12, color: "#5b6b65" },
  headerRight: {},
  projectPill: { fontFamily: "'IBM Plex Mono', monospace", fontSize: 12, background: "#eff3f0", border: "1px solid #dce3df", borderRadius: 20, padding: "4px 12px" },

  tabsRow: { display: "flex", gap: 4, padding: "6px 10px 0", background: "#ffffff", border: "1px solid #dce3df", borderRadius: 10, marginBottom: 10 },
  tabActive: {
    fontFamily: "inherit",
    fontSize: 13,
    fontWeight: 600,
    padding: "8px 14px",
    color: "#1c2321",
    background: "none",
    border: "none",
    borderBottom: "2px solid #3f6656",
    cursor: "pointer",
  },
  tabInactive: {
    fontFamily: "inherit",
    fontSize: 13,
    padding: "8px 14px",
    color: "#5b6b65",
    background: "none",
    border: "none",
    borderBottom: "2px solid transparent",
    cursor: "pointer",
  },
  tabDisabled: { fontSize: 13, padding: "8px 14px", color: "#c2cac6", cursor: "not-allowed" },

  body: { display: "flex", gap: 0, padding: 16, alignItems: "stretch" },
  contentColumn: { flex: "1 1 auto", display: "flex", flexDirection: "column", gap: 0, minWidth: 0 },

  topBarRow: {
    display: "flex",
    alignItems: "center",
    gap: 16,
    padding: "8px 12px",
    background: "#ffffff",
    borderBottom: "1px solid #dce3df",
  },
  mainTabsGroup: { display: "flex", gap: 4, flexShrink: 0 },
  searchBarInline: { display: "flex", alignItems: "center", gap: 8, flex: "1 1 auto", minWidth: 0 },
  searchIcon: { fontSize: 13, opacity: 0.6 },
  searchInput: { flex: 1, fontFamily: "inherit", fontSize: 13, border: "none", outline: "none", background: "transparent", color: "#1c2321" },
  searchClearBtn: { background: "none", border: "none", color: "#9aa8a2", cursor: "pointer", fontSize: 12, padding: "2px 4px" },
  searchHighlight: { background: "#fff176", padding: "0 2px", borderRadius: 2 },
  searchViewWrap: { display: "flex", flexDirection: "column", flex: "1 1 auto", gap: 0, minWidth: 0 },
  searchResultsHeader: { display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 10 },
  searchResultsCount: { fontFamily: "'Manrope', sans-serif", fontWeight: 700, fontSize: 14 },
  searchRowTop: { display: "flex", alignItems: "center", gap: 6 },

  projSidebar: { display: "flex", flexDirection: "column", gap: 10, background: "#ffffff", border: "1px solid #dce3df", borderRadius: 10, padding: 12, alignSelf: "flex-start", minWidth: 0 },
  projSidebarLabel: { fontSize: 11, textTransform: "uppercase", letterSpacing: 0.4, color: "#5b6b65" },
  projList: { display: "flex", flexDirection: "column", gap: 6 },
  projRow: { borderRadius: 8, border: "1px solid #dce3df", padding: "8px 9px", cursor: "pointer", display: "flex", flexDirection: "column", gap: 3 },
  projRowNormal: { background: "#ffffff" },
  projRowLocked: { background: STATUS_STYLE.cancelled.bg, color: STATUS_STYLE.cancelled.text, border: `1px solid ${STATUS_STYLE.cancelled.border}` },
  projRowSelected: { outline: "2px solid #3f6656", outlineOffset: -1 },
  projName: { fontSize: 13, fontWeight: 500, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" },
  projMeta: { display: "flex", alignItems: "center", gap: 6, fontSize: 11, color: "#5b6b65" },
  projSubMeta: { fontSize: 10, color: "#9aa8a2" },
  projCounts: { fontFamily: "'IBM Plex Mono', monospace" },
  projDeleteBtn: { marginLeft: "auto", border: "none", background: "none", color: "#a32d2d", cursor: "pointer", fontSize: 11, padding: "0 2px" },
  projAddBtn: { fontFamily: "inherit", fontSize: 12.5, background: "none", border: "1px dashed #dce3df", borderRadius: 8, padding: "7px 9px", color: "#5b6b65", cursor: "pointer" },

  resizeHandle: { flex: "0 0 14px", cursor: "col-resize", position: "relative" },
  resizeHandleBar: { position: "absolute", top: 0, bottom: 0, left: "50%", width: 2, background: "#dce3df", transform: "translateX(-50%)", borderRadius: 2 },
  resizeHandleBarActive: { background: "#3f6656", width: 3 },

  tabContent: { display: "flex", flex: "1 1 auto", gap: 0, minWidth: 0 },

  leftCol: { flex: "1 1 auto", display: "flex", flexDirection: "column", gap: 12, minWidth: 260 },
  rightCol: { minWidth: 0 },

  filterBarRow: { display: "flex", alignItems: "center", gap: 10, background: "#ffffff", border: "1px solid #dce3df", borderRadius: 10, padding: "8px 12px" },
  filterInlineGroup: { display: "flex", gap: 6, alignItems: "center" },
  filterDivider: { width: 1, height: 18, background: "#dce3df", flexShrink: 0 },
  iconChipBtn: { background: "none", border: "none", padding: 2, cursor: "pointer", display: "flex", alignItems: "center" },
  statusIconBox: { display: "inline-block", width: 14, height: 14, borderRadius: 3, borderStyle: "solid", borderColor: "#1c2321", boxSizing: "border-box", background: "#ffffff" },
  dueIconInner: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    width: 22,
    height: 22,
    borderRadius: "50%",
    fontSize: 11,
    borderStyle: "solid",
    borderColor: "#1c2321",
    boxSizing: "border-box",
  },
  chip: { fontFamily: "inherit", fontSize: 12, padding: "5px 10px", borderRadius: 20, border: "1px solid #dce3df", background: "#ffffff", cursor: "pointer", color: "#1c2321" },
  chipActive: { background: "#3f6656", color: "#ffffff", borderColor: "#3f6656" },

  treeWrap: { background: "#ffffff", border: "1px solid #dce3df", borderRadius: 10, padding: 10, minHeight: 320, flex: 1, overflow: "auto" },
  emptyState: { color: "#5b6b65", fontSize: 12.5, padding: 16, textAlign: "center" },

  taskRow: { display: "flex", alignItems: "center", gap: 6, padding: "7px 10px", borderRadius: 6, marginBottom: 4, cursor: "pointer", fontSize: 13 },
  expandBtn: { background: "none", border: "none", cursor: "pointer", fontSize: 11, width: 14, padding: 0 },
  taskName: { flex: 1 },
  dueText: { fontFamily: "'IBM Plex Mono', monospace", fontSize: 11, opacity: 0.85 },

  detailPanel: { background: "#ffffff", border: "1px solid #dce3df", borderRadius: 10, padding: 16, display: "flex", flexDirection: "column", gap: 4 },
  detailEmpty: {
    background: "#ffffff",
    border: "1px dashed #dce3df",
    borderRadius: 10,
    padding: 24,
    fontSize: 13,
    color: "#5b6b65",
    textAlign: "center",
    height: "100%",
    boxSizing: "border-box",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  },
  fieldLabel: { fontSize: 11, textTransform: "uppercase", letterSpacing: 0.4, color: "#5b6b65", marginTop: 10, marginBottom: 4 },
  input: { fontFamily: "inherit", fontSize: 13, padding: "8px 10px", border: "1px solid #dce3df", borderRadius: 6, width: "100%", boxSizing: "border-box" },
  textarea: { fontFamily: "inherit", fontSize: 13, padding: "8px 10px", border: "1px solid #dce3df", borderRadius: 6, width: "100%", boxSizing: "border-box", resize: "vertical" },
  dueSummary: { fontSize: 12, color: "#5b6b65", display: "flex", alignItems: "center", gap: 4, flexWrap: "wrap" },
  removeDueBtn: {
    background: "#fceaea",
    color: "#a32d2d",
    border: "1px solid #f0b8b8",
    borderRadius: "50%",
    width: 20,
    height: 20,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: 11,
    cursor: "pointer",
    padding: 0,
    marginLeft: 6,
  },
  customDueRow: { display: "flex", alignItems: "center", gap: 8, marginTop: 8 },
  dateTimeInput: { fontFamily: "inherit", fontSize: 12.5, padding: "6px 8px", border: "1px solid #dce3df", borderRadius: 6 },
  quickDueRow: { display: "flex", gap: 6, flexWrap: "wrap", marginTop: 6 },
  moveRow: { marginTop: 10 },

  noteListCol: { display: "flex", flexDirection: "column", minWidth: 0 },
  editorCol: { flex: "1 1 auto", minWidth: 280 },
  editorPane: { background: "#ffffff", border: "1px solid #dce3df", borderRadius: 10, padding: 16, display: "flex", flexDirection: "column", gap: 10, height: "100%", boxSizing: "border-box" },
  noteTitleInput: { fontFamily: "'Manrope', sans-serif", fontWeight: 700, fontSize: 16, padding: "8px 10px", border: "1px solid #dce3df", borderRadius: 8, boxSizing: "border-box" },
  noteContentArea: {
    fontFamily: "inherit",
    fontSize: 13,
    padding: "10px 12px",
    border: "1px solid #dce3df",
    borderRadius: 8,
    boxSizing: "border-box",
    flex: 1,
    minHeight: 260,
    resize: "vertical",
    lineHeight: 1.5,
  },
  saveStatus: { fontSize: 11.5, color: "#5b6b65", fontStyle: "italic" },
  instanceHeader: { fontFamily: "'Manrope', sans-serif", fontWeight: 700, fontSize: 14, color: "#1c2321" },

  listCol: { display: "flex", flexDirection: "column", gap: 6, minWidth: 0 },
  listColLabel: { fontSize: 11, textTransform: "uppercase", letterSpacing: 0.4, color: "#5b6b65" },
  listWrap: { background: "#ffffff", border: "1px solid #dce3df", borderRadius: 10, padding: 8, flex: 1, overflow: "auto", minHeight: 320 },
  rowItem: { padding: "9px 10px", borderRadius: 8, cursor: "pointer", marginBottom: 3, border: "1px solid transparent" },
  rowItemSelected: { background: "#eff3f0", border: "1px solid #3f6656" },
  rowTitle: { fontSize: 13, fontWeight: 600 },
  rowMeta: { fontSize: 11.5, color: "#5b6b65", marginTop: 2, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" },
  rowSubMeta: { fontSize: 10.5, color: "#9aa8a2", marginTop: 3, fontFamily: "'IBM Plex Mono', monospace" },
  renameInput: { fontFamily: "inherit", fontSize: 13, padding: "6px 8px", border: "1px solid #3f6656", borderRadius: 6, width: "100%", boxSizing: "border-box" },

  btnPrimary: { fontFamily: "inherit", background: "#3f6656", color: "#fff", border: "none", borderRadius: 8, padding: "8px 14px", fontSize: 13, fontWeight: 500, cursor: "pointer" },
  btnSecondary: { fontFamily: "inherit", background: "#ffffff", color: "#1c2321", border: "1px solid #dce3df", borderRadius: 8, padding: "8px 14px", fontSize: 13, cursor: "pointer" },
  btnDanger: { fontFamily: "inherit", background: "#fceaea", color: "#a32d2d", border: "1px solid #f0b8b8", borderRadius: 8, padding: "8px 14px", fontSize: 13, cursor: "pointer" },

  overlay: { position: "absolute", inset: 0, background: "rgba(28,35,33,0.45)", display: "flex", alignItems: "center", justifyContent: "center", borderRadius: 12 },
  modal: { background: "#ffffff", borderRadius: 10, padding: 20, width: 320, boxShadow: "0 8px 24px rgba(0,0,0,0.15)" },
  modalTitle: { fontFamily: "'Manrope', sans-serif", fontWeight: 700, fontSize: 14, marginBottom: 12 },
  modalText: { fontSize: 13, color: "#5b6b65", marginBottom: 12 },
  modalActions: { display: "flex", justifyContent: "flex-end", gap: 8, marginTop: 14 },

  ctxMenu: {
    position: "fixed",
    zIndex: 1000,
    minWidth: 180,
    background: "#ffffff",
    border: "1px solid #dce3df",
    borderRadius: 10,
    boxShadow: "0 8px 24px rgba(0,0,0,0.15)",
    padding: 6,
    display: "flex",
    flexDirection: "column",
    gap: 1,
  },
  ctxItem: { fontFamily: "inherit", fontSize: 12.5, textAlign: "left", background: "none", border: "none", borderRadius: 7, padding: "8px 10px", cursor: "pointer", color: "#1c2321" },
  ctxItemDanger: { color: "#a32d2d" },
  ctxDivider: { height: 1, background: "#dce3df", margin: "4px 2px" },

  prioritiesListCol: {
    flex: "1 1 auto",
    minWidth: 300,
    background: "#ffffff",
    border: "1px solid #dce3df",
    borderRadius: 10,
    padding: 16,
    overflow: "auto",
  },
  prioritySection: { marginBottom: 22 },
  prioritySectionTitle: { fontFamily: "'Manrope', sans-serif", fontWeight: 700, fontSize: 14, marginBottom: 10, color: "#1c2321" },
  priorityRow: {
    display: "flex",
    alignItems: "center",
    gap: 10,
    padding: "9px 12px",
    borderRadius: 8,
    cursor: "pointer",
    border: "1px solid #dce3df",
    marginBottom: 6,
    background: "#ffffff",
  },
  priorityRowSelected: { background: "#eff3f0", border: "1px solid #3f6656" },
  priorityName: { flex: 1, fontSize: 13, fontWeight: 500, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" },
  priorityProject: { fontSize: 11.5, color: "#5b6b65", fontFamily: "'IBM Plex Mono', monospace", flexShrink: 0 },
  priorityDue: { fontSize: 11.5, minWidth: 100, textAlign: "right", flexShrink: 0 },

  readOnlyHeaderRow: { display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 6 },
  readOnlyName: { fontFamily: "'Manrope', sans-serif", fontWeight: 700, fontSize: 16, margin: "4px 0 12px" },
  readOnlyDesc: { fontSize: 13, whiteSpace: "pre-wrap", lineHeight: 1.5, background: "#eff3f0", borderRadius: 8, padding: 10 },
};
