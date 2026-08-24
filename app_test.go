package main

import (
	"errors"
	"testing"
	"time"

	"toutdoux/adapters"
	"toutdoux/adapters/sqlite"
	"toutdoux/domain"
)

// Ces tests exercent la chaîne complète — commande exposée au frontend, domaine,
// repository, SQLite réelle en mémoire. Ils vérifient le câblage : que les
// règles déjà testées unitairement soient bien invoquées, et que ce qui est
// calculé soit bien écrit.

func newTestApp(t *testing.T) *App {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("ouverture de la base : %v", err)
	}
	t.Cleanup(func() { db.Close() })
	clock := adapters.NewFixedClock(time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC))
	return newAppWithDB(db, clock)
}

// Au premier démarrage, seul le projet verrouillé existe, et il est en tête.
func TestListProjects_DiversFirst(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateProject("Alpha"); err != nil {
		t.Fatal(err)
	}
	projects, err := app.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 2 {
		t.Fatalf("%d projets, 2 attendus", len(projects))
	}
	if projects[0].ID != domain.DiversProjectID {
		t.Errorf("premier = %q, attendu le projet verrouillé", projects[0].Name)
	}
}

func TestCreateProject_RejectsDuplicateIgnoringCaseAndAccents(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateProject("Réunion Générale"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateProject("réunion générale"); !errors.Is(err, domain.ErrDuplicateName) {
		t.Errorf("erreur = %v, attendu ErrDuplicateName", err)
	}
}

func TestCreateProject_RejectsEmptyName(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateProject("   "); !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("erreur = %v, attendu ErrEmptyName", err)
	}
}

// Le projet « Transverse / Divers » ne peut être ni renommé ni supprimé (§2.1).
func TestDiversProject_IsLocked(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.RenameProject(domain.DiversProjectID, "Autre"); !errors.Is(err, domain.ErrProjectLocked) {
		t.Errorf("renommage : erreur = %v, attendu ErrProjectLocked", err)
	}
	if err := app.DeleteProject(domain.DiversProjectID); !errors.Is(err, domain.ErrProjectLocked) {
		t.Errorf("suppression : erreur = %v, attendu ErrProjectLocked", err)
	}
}

// Renommer un projet en ne changeant que la casse ne doit pas être vu comme un
// conflit avec lui-même.
func TestRenameProject_AllowsCaseOnlyChange(t *testing.T) {
	app := newTestApp(t)
	p, err := app.CreateProject("migration")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.RenameProject(p.ID, "Migration"); err != nil {
		t.Errorf("erreur = %v, le projet ne doit pas entrer en conflit avec lui-même", err)
	}
}

// order_index est peuplé dès l'insertion, sinon l'ordre des frères devient
// non déterministe (§3.2).
func TestCreateTask_AssignsIncrementalOrderIndex(t *testing.T) {
	app := newTestApp(t)
	for i, want := range []int{0, 1, 2} {
		res, err := app.CreateTask(domain.DiversProjectID, "", "tâche")
		if err != nil {
			t.Fatalf("création %d : %v", i, err)
		}
		if res.Task.OrderIndex != want {
			t.Errorf("création %d : order_index = %d, attendu %d", i, res.Task.OrderIndex, want)
		}
	}
}

// Le bout en bout de la règle du §2.2 : la cascade doit être invoquée ET écrite.
func TestCreateTask_ReactivatesAncestorsAndPersistsThem(t *testing.T) {
	app := newTestApp(t)

	parent, err := app.CreateTask(domain.DiversProjectID, "", "parent")
	if err != nil {
		t.Fatal(err)
	}
	grandParent := parent.Task.ID

	child, err := app.CreateTask(domain.DiversProjectID, grandParent, "enfant")
	if err != nil {
		t.Fatal(err)
	}

	// On termine toute la branche.
	if _, err := app.ToggleTaskCompleted(child.Task.ID); err != nil {
		t.Fatal(err)
	}
	tasks, err := app.GetTasks(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if !task.Completed {
			t.Fatalf("préalable non tenu : %q devait être terminée", task.Name)
		}
	}

	// Ajouter une sous-tâche doit réactiver toute la chaîne.
	res, err := app.CreateTask(domain.DiversProjectID, child.Task.ID, "nouvelle")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Reactived) != 2 {
		t.Errorf("%d ancêtres réactivés, 2 attendus", len(res.Reactived))
	}

	// Et surtout : la réactivation doit avoir été écrite en base, pas seulement
	// renvoyée au frontend.
	after, err := app.GetTasks(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range after {
		if !task.Active() {
			t.Errorf("%q est encore completed=%v cancelled=%v en base", task.Name, task.Completed, task.Cancelled)
		}
	}
}

func TestToggleTaskCompleted_PersistsCascade(t *testing.T) {
	app := newTestApp(t)
	parent, err := app.CreateTask(domain.DiversProjectID, "", "parent")
	if err != nil {
		t.Fatal(err)
	}
	child, err := app.CreateTask(domain.DiversProjectID, parent.Task.ID, "enfant")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := app.ToggleTaskCompleted(child.Task.ID); err != nil {
		t.Fatal(err)
	}
	tasks, err := app.GetTasks(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if !task.Completed {
			t.Errorf("%q n'est pas terminée : le parent doit suivre son unique enfant", task.Name)
		}
	}
}

// L'échéance passe par le domaine et l'horloge injectée : un vendredi doit
// donner le lundi, sans dépendre du jour où le test tourne (§2.3).
func TestSetTaskDueDateQuick_UsesInjectedClock(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	vendredi := time.Date(2026, 8, 21, 14, 0, 0, 0, time.UTC)
	app := newAppWithDB(db, adapters.NewFixedClock(vendredi))

	res, err := app.CreateTask(domain.DiversProjectID, "", "tâche")
	if err != nil {
		t.Fatal(err)
	}
	task, err := app.SetTaskDueDateQuick(res.Task.ID, "lendemain9h")
	if err != nil {
		t.Fatal(err)
	}
	if task.DueDate == nil {
		t.Fatal("échéance non posée")
	}
	if wd := task.DueDate.Weekday(); wd != time.Monday {
		t.Errorf("échéance un %v, lundi attendu depuis un vendredi", wd)
	}
	if task.DueDate.Hour() != 9 {
		t.Errorf("heure = %d, attendu 9", task.DueDate.Hour())
	}
}

func TestSetTaskDueDateQuick_UnknownKind(t *testing.T) {
	app := newTestApp(t)
	res, err := app.CreateTask(domain.DiversProjectID, "", "tâche")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.SetTaskDueDateQuick(res.Task.ID, "la-semaine-des-quatre-jeudis"); err == nil {
		t.Error("un raccourci inconnu doit être une erreur explicite")
	}
}

// Le patch distingue « non transmis » de « vidé » : sans ça, retirer une
// échéance et ne pas y toucher seraient indiscernables (§2.3).
func TestUpdateTask_ClearDueVersusUntouched(t *testing.T) {
	app := newTestApp(t)
	res, err := app.CreateTask(domain.DiversProjectID, "", "tâche")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.SetTaskDueDateQuick(res.Task.ID, "1h"); err != nil {
		t.Fatal(err)
	}

	name := "Renommée"
	task, err := app.UpdateTask(res.Task.ID, TaskPatch{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if task.DueDate == nil {
		t.Error("un patch qui ne mentionne pas l'échéance ne doit pas la retirer")
	}
	if task.Name != "Renommée" {
		t.Errorf("nom = %q", task.Name)
	}

	task, err = app.UpdateTask(res.Task.ID, TaskPatch{ClearDue: true})
	if err != nil {
		t.Fatal(err)
	}
	if task.DueDate != nil {
		t.Error("ClearDue doit retirer l'échéance")
	}
}

func TestUpdateTask_RejectsInvalidImportance(t *testing.T) {
	app := newTestApp(t)
	res, err := app.CreateTask(domain.DiversProjectID, "", "tâche")
	if err != nil {
		t.Fatal(err)
	}
	bad := "Urgentissime"
	if _, err := app.UpdateTask(res.Task.ID, TaskPatch{Importance: &bad}); !errors.Is(err, domain.ErrInvalidImportance) {
		t.Errorf("erreur = %v, attendu ErrInvalidImportance", err)
	}
}

func TestTaskDeletionSummaryAndDelete(t *testing.T) {
	app := newTestApp(t)
	parent, err := app.CreateTask(domain.DiversProjectID, "", "parent")
	if err != nil {
		t.Fatal(err)
	}
	child, err := app.CreateTask(domain.DiversProjectID, parent.Task.ID, "enfant")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateTask(domain.DiversProjectID, child.Task.ID, "petit"); err != nil {
		t.Fatal(err)
	}

	n, err := app.TaskDeletionSummary(parent.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("%d sous-tâches annoncées, 2 attendues", n)
	}

	if err := app.DeleteTask(parent.Task.ID); err != nil {
		t.Fatal(err)
	}
	tasks, err := app.GetTasks(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Errorf("%d tâches restantes, 0 attendue", len(tasks))
	}
}

func TestCreateTask_UnknownProject(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateTask("projet-inexistant", "", "tâche"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu ErrNotFound", err)
	}
}

func TestNotes_CRUD(t *testing.T) {
	app := newTestApp(t)

	note, err := app.CreateNote(domain.DiversProjectID, "Compte rendu")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := app.UpdateNote(note.ID, "Compte rendu v2", "<p>échéance repoussée</p>")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Compte rendu v2" || updated.Content != "<p>échéance repoussée</p>" {
		t.Errorf("après mise à jour : %q / %q", updated.Title, updated.Content)
	}

	notes, err := app.GetNotes(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("%d notes, 1 attendue", len(notes))
	}

	if err := app.DeleteNote(note.ID); err != nil {
		t.Fatal(err)
	}
	notes, err = app.GetNotes(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Errorf("%d notes restantes, 0 attendue", len(notes))
	}
}

// Supprimer un projet emporte ses tâches et ses notes (§2.1).
func TestDeleteProject_CascadesThroughApp(t *testing.T) {
	app := newTestApp(t)
	p, err := app.CreateProject("À supprimer")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateTask(p.ID, "", "tâche"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateNote(p.ID, "note"); err != nil {
		t.Fatal(err)
	}

	summary, err := app.ProjectDeletionSummary(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Tasks != 1 || summary.Notes != 1 {
		t.Errorf("résumé = %+v, 1 tâche et 1 note attendues", summary)
	}

	if err := app.DeleteProject(p.ID); err != nil {
		t.Fatal(err)
	}
	all, err := app.GetAllTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Errorf("%d tâches restantes, 0 attendue", len(all))
	}
}

func TestMoveTask_ReordersSiblings(t *testing.T) {
	app := newTestApp(t)
	var ids []string
	for _, nom := range []string{"a", "b", "c"} {
		res, err := app.CreateTask(domain.DiversProjectID, "", nom)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, res.Task.ID)
	}

	if _, err := app.MoveTask(ids[2], "top"); err != nil {
		t.Fatal(err)
	}
	tasks, err := app.GetTasks(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if tasks[0].Name != "c" {
		t.Errorf("premier = %q, attendu \"c\" après envoi au début", tasks[0].Name)
	}
}

func TestMoveTask_UnknownDirection(t *testing.T) {
	app := newTestApp(t)
	res, err := app.CreateTask(domain.DiversProjectID, "", "a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.MoveTask(res.Task.ID, "de-cote"); err == nil {
		t.Error("une direction inconnue doit être une erreur explicite")
	}
}

// Le drag & drop d'une tâche active sous un parent terminé réactive la chaîne,
// et l'écrit (§2.2).
func TestReparentTask_ReactivatesAndPersists(t *testing.T) {
	app := newTestApp(t)
	cible, err := app.CreateTask(domain.DiversProjectID, "", "cible")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ToggleTaskCompleted(cible.Task.ID); err != nil {
		t.Fatal(err)
	}
	libre, err := app.CreateTask(domain.DiversProjectID, "", "libre")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := app.ReparentTask(libre.Task.ID, cible.Task.ID); err != nil {
		t.Fatal(err)
	}
	tasks, err := app.GetTasks(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range tasks {
		if !x.Active() {
			t.Errorf("%q est encore completed=%v en base après la cascade", x.Name, x.Completed)
		}
		if x.Name == "libre" && (x.ParentID == nil || *x.ParentID != cible.Task.ID) {
			t.Error("la tâche déplacée n'a pas le bon parent en base")
		}
	}
}

// L'anti-cycle doit remonter jusqu'à l'appelant, pas échouer silencieusement.
func TestReparentTask_RejectsCycle(t *testing.T) {
	app := newTestApp(t)
	parent, err := app.CreateTask(domain.DiversProjectID, "", "parent")
	if err != nil {
		t.Fatal(err)
	}
	enfant, err := app.CreateTask(domain.DiversProjectID, parent.Task.ID, "enfant")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ReparentTask(parent.Task.ID, enfant.Task.ID); !errors.Is(err, domain.ErrCycle) {
		t.Errorf("erreur = %v, attendu ErrCycle", err)
	}
}

// Passer une tâche à la racine est permis et vide son parent.
func TestReparentTask_ToRoot(t *testing.T) {
	app := newTestApp(t)
	parent, err := app.CreateTask(domain.DiversProjectID, "", "parent")
	if err != nil {
		t.Fatal(err)
	}
	enfant, err := app.CreateTask(domain.DiversProjectID, parent.Task.ID, "enfant")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ReparentTask(enfant.Task.ID, ""); err != nil {
		t.Fatal(err)
	}
	tasks, err := app.GetTasks(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range tasks {
		if x.Name == "enfant" && x.ParentID != nil {
			t.Errorf("parent = %v, nil attendu à la racine", *x.ParentID)
		}
	}
}

// La vue rend visibilité, dépliement et libellés d'échéance en une seule fois,
// tous calculés par le domaine (§2.2, §2.3, §2.4).
func TestGetTaskView(t *testing.T) {
	app := newTestApp(t)
	parent, err := app.CreateTask(domain.DiversProjectID, "", "parent")
	if err != nil {
		t.Fatal(err)
	}
	enfant, err := app.CreateTask(domain.DiversProjectID, parent.Task.ID, "enfant")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.SetTaskDueDateQuick(enfant.Task.ID, "10min"); err != nil {
		t.Fatal(err)
	}

	view, err := app.GetTaskView(domain.DiversProjectID, FilterSelection{
		Status:     []string{"active"},
		Importance: []string{"Basse", "Normale", "Haute", "Critique"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Tasks) != 2 {
		t.Errorf("%d tâches, 2 attendues", len(view.Tasks))
	}
	if len(view.Visible) != 2 {
		t.Errorf("%d visibles, 2 attendues", len(view.Visible))
	}
	info := view.Due[enfant.Task.ID]
	if info == nil {
		t.Fatal("le libellé d'échéance de l'enfant est absent")
	}
	if info.Text != "dans 10 min" {
		t.Errorf("libellé = %q, attendu \"dans 10 min\"", info.Text)
	}
	if info.Urgent {
		t.Error("10 minutes n'est pas urgent : le seuil est à 5 (§2.3)")
	}
}

// Un parent terminé reste visible s'il porte un enfant actif (§2.4).
func TestGetTaskView_KeepsAncestorOfVisibleChild(t *testing.T) {
	app := newTestApp(t)
	parent, err := app.CreateTask(domain.DiversProjectID, "", "parent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateTask(domain.DiversProjectID, parent.Task.ID, "enfant"); err != nil {
		t.Fatal(err)
	}
	// On termine le parent seul, en le décorrélant de son enfant.
	if _, err := app.UpdateTask(parent.Task.ID, TaskPatch{}); err != nil {
		t.Fatal(err)
	}

	view, err := app.GetTaskView(domain.DiversProjectID, FilterSelection{
		Status: []string{"active"}, Importance: []string{"Normale"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Visible) != 2 {
		t.Errorf("%d visibles, 2 attendues", len(view.Visible))
	}
}

// Un appel sans filtre ne doit pas rendre un arbre vide, ce qui passerait pour
// une panne (§2.4).
func TestGetTaskView_EmptyFiltersFallsBackToDefaults(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateTask(domain.DiversProjectID, "", "tâche"); err != nil {
		t.Fatal(err)
	}
	view, err := app.GetTaskView(domain.DiversProjectID, FilterSelection{})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Visible) != 1 {
		t.Errorf("%d visibles, 1 attendue avec les filtres par défaut", len(view.Visible))
	}
}
