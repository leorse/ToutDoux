package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"toutdoux/adapters"
	"toutdoux/adapters/model"
	"toutdoux/adapters/onnx"
	"toutdoux/adapters/sqlite"
	"toutdoux/domain"
	"toutdoux/domain/cascade"
	"toutdoux/domain/duedate"
	"toutdoux/domain/stats"
	"toutdoux/domain/tasktree"
	"toutdoux/ports"
)

// App est l'adaptateur pilote : les méthodes exportées ci-dessous sont celles
// que Wails expose au frontend (§3.6).
//
// Il ne contient aucune règle métier. Son rôle est de traduire un appel du
// frontend en lecture de repository, appel au domaine, puis écriture — et rien
// d'autre. Toute logique qui apparaîtrait ici appartient à domain/.
type App struct {
	ctx context.Context
	db  *sqlite.DB

	projects   ports.ProjectRepository
	tasks      ports.TaskRepository
	notes      ports.NoteRepository
	noteGroups ports.NoteGroupRepository
	meetings   ports.MeetingRepository
	index      ports.SearchIndex
	clock      ports.Clock

	// Recherche sémantique (§2.12). Optionnelle : sans modèle déposé, embedder
	// répond « indisponible » et le reste de l'application fonctionne
	// exactement comme avant.
	embeddings ports.EmbeddingRepository
	embedder   ports.EmbeddingProvider

	// modelDir est le dossier où l'utilisateur dépose le modèle (§3.3).
	modelDir string

	// enArrierePlan exécute une tâche hors du chemin de réponse.
	//
	// C'est un champ et non un `go` en dur pour une raison de test : une
	// revectorisation lancée en goroutine est impossible à observer de façon
	// déterministe, et un test qui attend « un peu » est un test qui échouera
	// un jour sur une machine chargée. Les tests remplacent ce champ par une
	// exécution synchrone.
	enArrierePlan func(func())

	// tray est nil dans les tests : la barre système est un adaptateur pilote,
	// pas une dépendance du métier.
	tray *tray
}

// NewApp construit l'application avec ses dépendances par défaut.
func NewApp() *App {
	return &App{clock: adapters.SystemClock{}, enArrierePlan: func(f func()) { go f() }}
}

// newAppWithDB câble l'application sur une base déjà ouverte et une horloge
// donnée, sans passer par startup.
//
// C'est le point d'entrée des tests : il permet de brancher une SQLite en
// mémoire et une horloge figée, sans toucher au fichier app.db de l'utilisateur
// ni dépendre de l'heure réelle.
func newAppWithDB(db *sqlite.DB, clock ports.Clock) *App {
	return &App{
		db:         db,
		projects:   sqlite.NewProjectRepository(db),
		tasks:      sqlite.NewTaskRepository(db),
		notes:      sqlite.NewNoteRepository(db),
		noteGroups: sqlite.NewNoteGroupRepository(db),
		meetings:   sqlite.NewMeetingRepository(db),
		index:      sqlite.NewSearchIndexAdapter(db),
		embeddings: sqlite.NewEmbeddingRepository(db),
		// Fournisseur factice par défaut : aucun test ne doit charger un modèle
		// de 120 Mo (§3.9). Les tests qui veulent observer les appels gardent
		// une référence sur celui qu'ils injectent.
		embedder: adapters.NewFakeEmbeddingProvider(),
		clock:    clock,
		// Exécution synchrone : voir le commentaire du champ.
		enArrierePlan: func(f func()) { f() },
	}
}

// startup ouvre la base et câble les repositories.
//
// L'échec est fatal et volontairement bruyant : sans base, aucune des méthodes
// suivantes ne peut fonctionner, et une fenêtre qui s'ouvre sur une application
// incapable de lire ou d'écrire serait plus déroutante qu'un arrêt net.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	path, err := databasePath()
	if err != nil {
		panic(fmt.Sprintf("emplacement de la base : %v", err))
	}
	db, err := sqlite.Open(path)
	if err != nil {
		panic(fmt.Sprintf("ouverture de la base %s : %v", path, err))
	}

	a.db = db
	a.projects = sqlite.NewProjectRepository(db)
	a.tasks = sqlite.NewTaskRepository(db)
	a.notes = sqlite.NewNoteRepository(db)
	a.noteGroups = sqlite.NewNoteGroupRepository(db)
	a.meetings = sqlite.NewMeetingRepository(db)
	a.index = sqlite.NewSearchIndexAdapter(db)
	a.embeddings = sqlite.NewEmbeddingRepository(db)

	// Recherche sémantique (§2.12). Le dossier du modèle est voisin de la base,
	// dans les données utilisateur : l'emplacement de l'exécutable peut ne pas
	// être inscriptible (§3.3).
	//
	// Le dossier est créé vide s'il n'existe pas. C'est le seul geste que
	// l'application fait pour le modèle : elle ne télécharge rien, jamais (§6).
	// Mais un dossier qui existe est un dossier qu'on peut ouvrir depuis
	// l'explorateur pour y déposer les fichiers, sans avoir à le créer soi-même
	// avec le bon nom.
	a.modelDir = model.Dir(path)
	if err := os.MkdirAll(a.modelDir, 0o755); err != nil {
		log.Printf("[semantique] dossier des modeles indisponible : %v", err)
	}
	// Le fournisseur ne lit rien à la construction : le modèle n'est chargé
	// qu'à la première vectorisation réelle (§3.8).
	a.embedder = onnx.NewProvider(a.modelDir)

	// L'index est reconstruit au démarrage : il peut avoir divergé si
	// l'application s'est arrêtée entre une écriture et son indexation, et il
	// est vide pour une base créée avant l'arrivée de la recherche.
	//
	// Un échec ici n'est PAS fatal, contrairement à l'ouverture de la base.
	// L'index n'est qu'un cache : sans lui la recherche rend moins de résultats,
	// mais projets, tâches et notes restent parfaitement utilisables. Faire
	// paniquer l'application pour un cache tuait le processus au démarrage, sans
	// autre symptôme visible qu'une fenêtre qui ne s'ouvre jamais.
	if err := a.reindexAll(); err != nil {
		log.Printf("[index] reconstruction impossible, la recherche sera incomplète : %v", err)
	}

	// Barre système (§2.10). Démarrée après la base : le menu lit les tâches
	// dès son premier cycle.
	a.startTray()
}

// shutdown ferme proprement la base.
func (a *App) shutdown(ctx context.Context) {
	a.stopTray()

	// Le fournisseur de vecteurs tient une session ONNX et une bibliothèque
	// native chargée. L'interface optionnelle évite de faire connaître à App
	// l'adaptateur concret : les fournisseurs factices n'ont rien à libérer.
	if c, ok := a.embedder.(interface{ Close() }); ok {
		c.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
}

// databasePath rend l'emplacement de app.db (§3.3).
//
// Les données vivent dans le répertoire de configuration de l'utilisateur et
// non à côté de l'exécutable : un binaire installé dans Program Files n'est pas
// inscriptible, et le poste peut être restreint.
func databasePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "ToutDoux")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "app.db"), nil
}

/* ---------------- Projets (§2.1) ---------------- */

// ListProjects rend les projets dans leur ordre d'affichage : « Transverse /
// Divers » en tête, les autres par nom (§2.1).
func (a *App) ListProjects() ([]domain.Project, error) {
	projects, err := a.projects.List()
	if err != nil {
		return nil, err
	}
	return stats.SortProjects(projects), nil
}

// CreateProject crée un projet après contrôle d'unicité insensible à la casse.
func (a *App) CreateProject(name string) (domain.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Project{}, domain.ErrEmptyName
	}
	exists, err := a.projects.ExistsByName(name, "")
	if err != nil {
		return domain.Project{}, err
	}
	if exists {
		return domain.Project{}, domain.ErrDuplicateName
	}

	now := a.clock.Now()
	p := domain.Project{ID: uuid.NewString(), Name: name, CreatedAt: now, UpdatedAt: now}
	if err := a.projects.Create(p); err != nil {
		return domain.Project{}, err
	}
	return p, a.indexProject(p)
}

// RenameProject renomme un projet non verrouillé.
func (a *App) RenameProject(id, newName string) (domain.Project, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return domain.Project{}, domain.ErrEmptyName
	}
	p, err := a.projects.Get(id)
	if err != nil {
		return domain.Project{}, err
	}
	if p.Locked {
		return domain.Project{}, domain.ErrProjectLocked
	}

	// L'exclusion de l'identifiant courant évite qu'un projet renommé avec une
	// simple différence de casse se déclare en conflit avec lui-même.
	exists, err := a.projects.ExistsByName(newName, id)
	if err != nil {
		return domain.Project{}, err
	}
	if exists {
		return domain.Project{}, domain.ErrDuplicateName
	}
	if err := a.projects.Rename(id, newName); err != nil {
		return domain.Project{}, err
	}
	maj, err := a.projects.Get(id)
	if err != nil {
		return domain.Project{}, err
	}
	return maj, a.indexProject(maj)
}

// SetProjectHidden masque ou réaffiche un projet non verrouillé.
//
// Purement visuel : ni la recherche, ni les Priorités, ni la barre système ne
// tiennent compte de ce marqueur (§2.1).
func (a *App) SetProjectHidden(id string, hidden bool) (domain.Project, error) {
	p, err := a.projects.Get(id)
	if err != nil {
		return domain.Project{}, err
	}
	if p.Locked {
		return domain.Project{}, domain.ErrProjectLocked
	}
	if err := a.projects.SetHidden(id, hidden); err != nil {
		return domain.Project{}, err
	}
	return a.projects.Get(id)
}

// DeleteProjectSummary annonce ce que la suppression emportera, pour la
// confirmation demandée par le §2.1.
type DeleteProjectSummary struct {
	Tasks    int `json:"tasks"`
	Notes    int `json:"notes"`
	Meetings int `json:"meetings"`
}

// ProjectDeletionSummary compte ce que la suppression d'un projet effacerait.
//
// Il faut pouvoir annoncer le coût avant d'agir : la suppression est en cascade
// et sans retour possible.
func (a *App) ProjectDeletionSummary(id string) (DeleteProjectSummary, error) {
	tasks, err := a.tasks.ListByProject(id)
	if err != nil {
		return DeleteProjectSummary{}, err
	}
	notes, err := a.notes.ListByProject(id)
	if err != nil {
		return DeleteProjectSummary{}, err
	}
	meetings, err := a.meetings.ListByProject(id)
	if err != nil {
		return DeleteProjectSummary{}, err
	}
	return DeleteProjectSummary{Tasks: len(tasks), Notes: len(notes), Meetings: len(meetings)}, nil
}

// DeleteProject supprime un projet non verrouillé et tout son contenu (§2.1).
func (a *App) DeleteProject(id string) error {
	p, err := a.projects.Get(id)
	if err != nil {
		return err
	}
	if p.Locked {
		return domain.ErrProjectLocked
	}
	if err := a.projects.Delete(id); err != nil {
		return err
	}
	return a.index.DeleteByProject(id)
}

// GetSidebarStats calcule les compteurs et le code couleur d'un projet (§2.1).
func (a *App) GetSidebarStats(projectID string) (stats.SidebarStats, error) {
	tasks, err := a.tasks.ListByProject(projectID)
	if err != nil {
		return stats.SidebarStats{}, err
	}
	notes, err := a.notes.ListByProject(projectID)
	if err != nil {
		return stats.SidebarStats{}, err
	}
	meetings, err := a.meetings.ListByProject(projectID)
	if err != nil {
		return stats.SidebarStats{}, err
	}
	return stats.Sidebar(tasks, notes, meetings, projectID, a.clock.Now()), nil
}

/* ---------------- Tâches (§2.2) ---------------- */

// GetTasks rend les tâches d'un projet, triées par order_index.
func (a *App) GetTasks(projectID string) ([]domain.Task, error) {
	return a.tasks.ListByProject(projectID)
}

// GetAllTasks rend toutes les tâches, pour les vues transverses (§2.8, §2.10).
func (a *App) GetAllTasks() ([]domain.Task, error) {
	return a.tasks.ListAll()
}

// FilterSelection est l'état des filtres transmis par le frontend (§2.4).
//
// Les listes remplacent les ensembles de Go, que JSON ne sait pas représenter.
type FilterSelection struct {
	Status     []string `json:"status"`
	Importance []string `json:"importance"`
	DueOnly    bool     `json:"dueOnly"`
}

// TaskView est tout ce qu'il faut pour dessiner l'arbre d'un projet.
//
// Les quatre informations arrivent ensemble et sont calculées par le domaine.
// C'est délibéré : la visibilité sous filtre (§2.4), le dépliement par défaut
// (§2.2) et le libellé d'échéance (§2.3) sont des règles métier. Les
// réimplémenter en TypeScript pour éviter un aller-retour créerait deux
// définitions à garder d'accord — exactement ce que l'architecture refuse.
type TaskView struct {
	Tasks           []domain.Task            `json:"tasks"`
	Visible         []string                 `json:"visible"`
	DefaultExpanded []string                 `json:"defaultExpanded"`
	Due             map[string]*duedate.Info `json:"due"`
}

// GetTaskView rend l'arbre d'un projet, filtré et annoté.
//
// Le frontend rappelle cette méthode toutes les 30 secondes pour rafraîchir les
// libellés d'échéance (§2.3) : c'est pour cela que Due est recalculé ici, à
// partir de l'horloge, plutôt que figé à la création de la tâche.
func (a *App) GetTaskView(projectID string, filters FilterSelection) (TaskView, error) {
	tasks, err := a.tasks.ListByProject(projectID)
	if err != nil {
		return TaskView{}, err
	}

	f := tasktree.Filters{
		Status:     map[domain.Status]bool{},
		Importance: map[domain.Importance]bool{},
		DueOnly:    filters.DueOnly,
	}
	for _, s := range filters.Status {
		f.Status[domain.Status(s)] = true
	}
	for _, i := range filters.Importance {
		f.Importance[domain.Importance(i)] = true
	}
	// Un appel sans filtre transmis prend l'état par défaut plutôt que de
	// masquer tout l'arbre, ce qui passerait pour une panne (§2.4).
	if len(f.Status) == 0 && len(f.Importance) == 0 {
		f = tasktree.DefaultFilters()
	}

	now := a.clock.Now()
	view := TaskView{
		Tasks:           tasks,
		Visible:         []string{},
		DefaultExpanded: []string{},
		Due:             map[string]*duedate.Info{},
	}
	for id := range tasktree.ComputeVisibility(tasks, f) {
		view.Visible = append(view.Visible, id)
	}
	for id := range tasktree.DefaultExpanded(tasks, projectID) {
		view.DefaultExpanded = append(view.DefaultExpanded, id)
	}
	for _, t := range tasks {
		// Une tâche terminée ou annulée n'affiche plus de délai : sans cela,
		// une tâche finie dans les temps passerait « en retard » avec les jours.
		if !t.Active() {
			continue
		}
		if info := duedate.Format(t.DueDate, now); info != nil {
			view.Due[t.ID] = info
		}
	}
	return view, nil
}

// CreateTaskResult porte la tâche créée et les ancêtres que sa création a
// réactivés (§2.2).
//
// Les deux voyagent ensemble parce que le frontend doit rafraîchir les deux :
// ne renvoyer que la tâche laisserait l'arbre afficher des parents encore
// barrés au-dessus d'une sous-tâche active.
type CreateTaskResult struct {
	Task      domain.Task   `json:"task"`
	Reactived []domain.Task `json:"reactivated"`
}

// CreateTask crée une tâche ou une sous-tâche (§2.2).
//
// parentID vide crée une tâche racine. La nouvelle tâche étant active par
// nature, ses ancêtres terminés ou annulés sont réactivés en cascade.
func (a *App) CreateTask(projectID string, parentID string, name string) (CreateTaskResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CreateTaskResult{}, domain.ErrEmptyName
	}
	if _, err := a.projects.Get(projectID); err != nil {
		return CreateTaskResult{}, err
	}

	var parent *string
	if parentID != "" {
		if _, err := a.tasks.Get(parentID); err != nil {
			return CreateTaskResult{}, err
		}
		parent = &parentID
	}

	all, err := a.tasks.ListAll()
	if err != nil {
		return CreateTaskResult{}, err
	}

	now := a.clock.Now()
	task := domain.Task{
		ID:         uuid.NewString(),
		ProjectID:  projectID,
		ParentID:   parent,
		Name:       name,
		Importance: domain.ImportanceNormale,
		// order_index est peuplé dès l'insertion : sans cela l'ordre des frères
		// devient non déterministe (§3.2).
		OrderIndex: tasktree.NextOrderIndex(all, projectID, parent),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := a.tasks.Create(task); err != nil {
		return CreateTaskResult{}, err
	}

	if err := a.indexTask(task); err != nil {
		return CreateTaskResult{}, err
	}

	result := CreateTaskResult{Task: task, Reactived: []domain.Task{}}
	if parent != nil {
		reactivated := cascade.ReactivateAncestors(all, *parent)
		if err := a.tasks.UpdateMany(reactivated); err != nil {
			return CreateTaskResult{}, err
		}
		result.Reactived = reactivated
	}
	return result, nil
}

// TaskPatch porte les champs modifiables du panneau de détail (§2.2).
//
// Chaque champ est un pointeur pour distinguer « non transmis » de « vidé » :
// sans cette distinction, retirer une échéance et ne pas y toucher seraient
// indiscernables (§2.3, la croix rouge).
type TaskPatch struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Importance  *string `json:"importance"`
	DueDate     *string `json:"dueDate"` // RFC3339, ou chaîne vide pour retirer
	ClearDue    bool    `json:"clearDue"`
}

// UpdateTask applique un patch au détail d'une tâche.
func (a *App) UpdateTask(taskID string, patch TaskPatch) (domain.Task, error) {
	task, err := a.tasks.Get(taskID)
	if err != nil {
		return domain.Task{}, err
	}

	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return domain.Task{}, domain.ErrEmptyName
		}
		task.Name = name
	}
	if patch.Description != nil {
		task.Description = *patch.Description
	}
	if patch.Importance != nil {
		imp := domain.Importance(*patch.Importance)
		if !imp.Valid() {
			return domain.Task{}, domain.ErrInvalidImportance
		}
		task.Importance = imp
	}
	switch {
	case patch.ClearDue:
		task.DueDate = nil
	case patch.DueDate != nil && *patch.DueDate != "":
		due, err := time.Parse(time.RFC3339, *patch.DueDate)
		if err != nil {
			return domain.Task{}, fmt.Errorf("échéance illisible : %w", err)
		}
		task.DueDate = &due
	}

	if err := a.tasks.Update(task); err != nil {
		return domain.Task{}, err
	}
	if err := a.indexTask(task); err != nil {
		return domain.Task{}, err
	}
	// Revectorisation en arrière-plan si la tâche est dans l'index sémantique
	// (§2.12). Un patch qui ne touche qu'à l'importance ou à l'échéance ne
	// déclenche aucun calcul : l'empreinte du texte est inchangée.
	a.rafraichirEnArrierePlan(taskID)
	return a.tasks.Get(taskID)
}

// SetTaskDueDateQuick applique un raccourci d'échéance (§2.3).
//
// Le calcul appartient au domaine et prend l'heure du port Clock, jamais
// time.Now() directement : c'est ce qui rend la règle « vendredi → lundi »
// testable.
func (a *App) SetTaskDueDateQuick(taskID string, kind string) (domain.Task, error) {
	task, err := a.tasks.Get(taskID)
	if err != nil {
		return domain.Task{}, err
	}
	due := duedate.Quick(duedate.Kind(kind), a.clock.Now())
	if due == nil {
		return domain.Task{}, fmt.Errorf("raccourci d'échéance inconnu : %q", kind)
	}
	task.DueDate = due
	if err := a.tasks.Update(task); err != nil {
		return domain.Task{}, err
	}
	return a.tasks.Get(taskID)
}

// ToggleTaskCompleted inverse l'état terminé d'une tâche et propage (§2.2).
//
// Rend toutes les tâches touchées — la tâche, sa descendance, ses ancêtres
// recalculés — parce que le frontend doit toutes les rafraîchir.
func (a *App) ToggleTaskCompleted(taskID string) ([]domain.Task, error) {
	task, err := a.tasks.Get(taskID)
	if err != nil {
		return nil, err
	}
	all, err := a.tasks.ListAll()
	if err != nil {
		return nil, err
	}
	changed, err := cascade.SetCompleted(all, taskID, !task.Completed)
	if err != nil {
		return nil, err
	}
	if err := a.tasks.UpdateMany(changed); err != nil {
		return nil, err
	}
	return changed, nil
}

// ToggleTaskCancelled inverse l'état annulé d'une tâche (§2.2).
func (a *App) ToggleTaskCancelled(taskID string) ([]domain.Task, error) {
	task, err := a.tasks.Get(taskID)
	if err != nil {
		return nil, err
	}
	all, err := a.tasks.ListAll()
	if err != nil {
		return nil, err
	}
	changed, err := cascade.SetCancelled(all, taskID, !task.Cancelled)
	if err != nil {
		return nil, err
	}
	if err := a.tasks.UpdateMany(changed); err != nil {
		return nil, err
	}
	return changed, nil
}

// MoveTask réordonne une tâche parmi ses frères (§2.2).
//
// direction vaut "up", "down", "top" ou "bottom". Rend les tâches dont l'ordre
// a changé, tranche vide si le mouvement était sans effet.
func (a *App) MoveTask(taskID string, direction string) ([]domain.Task, error) {
	if _, err := a.tasks.Get(taskID); err != nil {
		return nil, err
	}
	all, err := a.tasks.ListAll()
	if err != nil {
		return nil, err
	}
	changed := tasktree.Move(all, taskID, tasktree.Direction(direction))
	if changed == nil {
		return nil, fmt.Errorf("direction de déplacement inconnue : %q", direction)
	}
	if err := a.tasks.UpdateMany(changed); err != nil {
		return nil, err
	}
	return changed, nil
}

// ReparentTask déplace une tâche sous un nouveau parent (§2.2).
//
// newParentID vide place la tâche à la racine du projet. Le déplacement est
// refusé avec une erreur explicite s'il créerait un cycle, et réactive les
// ancêtres si la tâche déplacée est active.
func (a *App) ReparentTask(taskID string, newParentID string) ([]domain.Task, error) {
	all, err := a.tasks.ListAll()
	if err != nil {
		return nil, err
	}
	var parent *string
	if newParentID != "" {
		parent = &newParentID
	}
	changed, err := cascade.Reparent(all, taskID, parent)
	if err != nil {
		return nil, err
	}
	if err := a.tasks.UpdateMany(changed); err != nil {
		return nil, err
	}
	return changed, nil
}

// TaskDeletionSummary compte les sous-tâches qu'une suppression emporterait,
// pour la confirmation du §2.2.
func (a *App) TaskDeletionSummary(taskID string) (int, error) {
	all, err := a.tasks.ListAll()
	if err != nil {
		return 0, err
	}
	return len(tasktree.DescendantIDs(all, taskID)), nil
}

// DeleteTask supprime une tâche et toute sa descendance (§2.2).
func (a *App) DeleteTask(taskID string) error {
	if _, err := a.tasks.Get(taskID); err != nil {
		return err
	}
	all, err := a.tasks.ListAll()
	if err != nil {
		return err
	}
	disparues := append([]string{taskID}, keysOf(tasktree.DescendantIDs(all, taskID))...)
	if err := a.tasks.Delete(taskID); err != nil {
		return err
	}
	for _, id := range disparues {
		if err := a.index.Delete(id); err != nil {
			return err
		}
		// Un vecteur orphelin resterait dans l'index sémantique et ferait
		// remonter un résultat qu'on ne peut plus ouvrir (§2.12).
		if err := a.embeddings.Delete(id); err != nil {
			return err
		}
	}
	return nil
}

/* ---------------- Notes (§2.6) ---------------- */

// GetNotes rend les notes d'un projet, dans l'ordre choisi par l'utilisateur.
func (a *App) GetNotes(projectID string) ([]domain.Note, error) {
	return a.notes.ListByProject(projectID)
}

// CreateNote crée une note vide dans un projet, en tête de liste et hors groupe.
func (a *App) CreateNote(projectID, title string) (domain.Note, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Note{}, domain.ErrEmptyName
	}
	if _, err := a.projects.Get(projectID); err != nil {
		return domain.Note{}, err
	}
	now := a.clock.Now()
	n := domain.Note{ID: uuid.NewString(), ProjectID: projectID, Title: title, CreatedAt: now, UpdatedAt: now}
	if err := a.notes.Create(n); err != nil {
		return domain.Note{}, err
	}
	// Relue pour porter la position que le repository lui a donnée.
	n, err := a.notes.Get(n.ID)
	if err != nil {
		return domain.Note{}, err
	}
	return n, a.indexNote(n)
}

// UpdateNote écrit le titre et le contenu d'une note.
//
// C'est la cible de la sauvegarde automatique, appelée aussi bien à l'expiration
// du délai de 2 s qu'au flush immédiat déclenché par une perte de focus ou un
// changement de sélection (§2.6). L'appel est le même dans les deux cas : le
// choix du moment appartient au frontend.
func (a *App) UpdateNote(noteID, title, content string) (domain.Note, error) {
	n, err := a.notes.Get(noteID)
	if err != nil {
		return domain.Note{}, err
	}
	n.Title = title
	n.Content = content
	if err := a.notes.Update(n); err != nil {
		return domain.Note{}, err
	}
	maj, err := a.notes.Get(noteID)
	if err != nil {
		return domain.Note{}, err
	}
	// Revectorisation en arrière-plan si la note est dans l'index sémantique
	// (§2.12). Sans effet sinon : l'opt-in est strict.
	a.rafraichirEnArrierePlan(noteID)
	return maj, a.indexNote(maj)
}

// SetNoteHidden masque ou réaffiche une note (§2.6). Purement visuel : la
// recherche continue de la trouver.
func (a *App) SetNoteHidden(noteID string, hidden bool) (domain.Note, error) {
	if err := a.notes.SetHidden(noteID, hidden); err != nil {
		return domain.Note{}, err
	}
	return a.notes.Get(noteID)
}

// SetNoteColor pose ou retire (chaîne vide) la couleur d'une note. Purement
// visuel : ni le contenu, ni la place dans la liste, ni la recherche ne changent.
func (a *App) SetNoteColor(noteID string, color string) (domain.Note, error) {
	if !domain.ValidItemColor(color) {
		return domain.Note{}, domain.ErrInvalidColor
	}
	if err := a.notes.SetColor(noteID, color); err != nil {
		return domain.Note{}, err
	}
	return a.notes.Get(noteID)
}

// DeleteNote supprime une note. Son groupe disparaît si elle en était la dernière.
func (a *App) DeleteNote(noteID string) error {
	if err := a.notes.Delete(noteID); err != nil {
		return err
	}
	if err := a.embeddings.Delete(noteID); err != nil {
		return err
	}
	return a.index.Delete(noteID)
}
