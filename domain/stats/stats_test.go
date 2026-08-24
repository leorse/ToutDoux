package stats

import (
	"testing"
	"time"

	"toutdoux/domain"
)

var now = time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)

func at(d time.Duration) *time.Time {
	t := now.Add(d)
	return &t
}

func task(id string, imp domain.Importance, completed, cancelled bool, due *time.Time) domain.Task {
	return domain.Task{
		ID: id, ProjectID: "p1", Name: id, Importance: imp,
		Completed: completed, Cancelled: cancelled, DueDate: due,
	}
}

func TestSidebar_CountsOnlyActiveTasks(t *testing.T) {
	tasks := []domain.Task{
		task("a", domain.ImportanceNormale, false, false, nil),
		task("b", domain.ImportanceNormale, true, false, nil),
		task("c", domain.ImportanceNormale, false, true, nil),
	}
	s := Sidebar(tasks, nil, nil, "p1", now)
	if s.ActiveCount != 1 {
		t.Errorf("ActiveCount = %d, attendu 1 (les terminées et annulées ne comptent pas)", s.ActiveCount)
	}
}

// Le rouge prime sur l'orange : un projet ne porte qu'une couleur (§2.1).
func TestSidebar_CriticalWinsOverHigh(t *testing.T) {
	tasks := []domain.Task{
		task("haute", domain.ImportanceHaute, false, false, nil),
		task("critique", domain.ImportanceCritique, false, false, nil),
	}
	s := Sidebar(tasks, nil, nil, "p1", now)
	if !s.HasCritical {
		t.Error("HasCritical attendu")
	}
	if s.HasHigh {
		t.Error("HasHigh doit être faux quand une critique existe")
	}
}

// Une tâche critique mais terminée ne colore plus le projet.
func TestSidebar_CompletedCriticalDoesNotColour(t *testing.T) {
	tasks := []domain.Task{task("critique", domain.ImportanceCritique, true, false, nil)}
	if Sidebar(tasks, nil, nil, "p1", now).HasCritical {
		t.Error("une critique terminée ne doit plus colorer le projet")
	}
}

func TestSidebar_DueIconPriority(t *testing.T) {
	cases := []struct {
		name  string
		tasks []domain.Task
		want  DueIcon
	}{
		{"aucune échéance", []domain.Task{task("a", domain.ImportanceNormale, false, false, nil)}, DueIconNone},
		{"échéance lointaine", []domain.Task{task("a", domain.ImportanceNormale, false, false, at(72*time.Hour))}, DueIconUpcoming},
		{"échéance urgente", []domain.Task{task("a", domain.ImportanceNormale, false, false, at(2*time.Minute))}, DueIconUrgent},
		{"échéance dépassée", []domain.Task{task("a", domain.ImportanceNormale, false, false, at(-2*time.Hour))}, DueIconUrgent},
		{
			"l'urgente l'emporte sur la lointaine, quel que soit l'ordre",
			[]domain.Task{
				task("lointaine", domain.ImportanceNormale, false, false, at(72*time.Hour)),
				task("urgente", domain.ImportanceNormale, false, false, at(2*time.Minute)),
			},
			DueIconUrgent,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Sidebar(c.tasks, nil, nil, "p1", now).DueIcon; got != c.want {
				t.Errorf("DueIcon = %q, attendu %q", got, c.want)
			}
		})
	}
}

func TestSidebar_ScopedToProject(t *testing.T) {
	tasks := []domain.Task{
		task("ici", domain.ImportanceNormale, false, false, nil),
		{ID: "ailleurs", ProjectID: "p2", Importance: domain.ImportanceCritique},
	}
	notes := []domain.Note{{ID: "n1", ProjectID: "p1"}, {ID: "n2", ProjectID: "p2"}}
	meetings := []domain.Meeting{{ID: "m1", ProjectID: "p2"}}

	s := Sidebar(tasks, notes, meetings, "p1", now)
	if s.ActiveCount != 1 {
		t.Errorf("ActiveCount = %d, attendu 1", s.ActiveCount)
	}
	if s.HasCritical {
		t.Error("la critique appartient à un autre projet")
	}
	if s.NotesCount != 1 {
		t.Errorf("NotesCount = %d, attendu 1", s.NotesCount)
	}
	if s.MeetingsCount != 0 {
		t.Errorf("MeetingsCount = %d, attendu 0", s.MeetingsCount)
	}
}

func TestPriorities_OnlyActiveTasks(t *testing.T) {
	tasks := []domain.Task{
		task("critiqueActive", domain.ImportanceCritique, false, false, nil),
		task("critiqueFinie", domain.ImportanceCritique, true, false, nil),
		task("hauteAnnulee", domain.ImportanceHaute, false, true, nil),
		task("normaleAvecDate", domain.ImportanceNormale, false, false, at(time.Hour)),
	}
	p := Priorities(tasks)
	if len(p.CriticalOrHigh) != 1 || p.CriticalOrHigh[0].ID != "critiqueActive" {
		t.Errorf("CriticalOrHigh = %v, une seule tâche active attendue", ids(p.CriticalOrHigh))
	}
	if len(p.DueSoon) != 1 || p.DueSoon[0].ID != "normaleAvecDate" {
		t.Errorf("DueSoon = %v", ids(p.DueSoon))
	}
}

// Tri par urgence : en retard d'abord, puis par échéance croissante, sans date
// en dernier (§2.8).
func TestPriorities_UrgencySort(t *testing.T) {
	tasks := []domain.Task{
		task("sansDate", domain.ImportanceCritique, false, false, nil),
		task("demain", domain.ImportanceCritique, false, false, at(24*time.Hour)),
		task("enRetard", domain.ImportanceCritique, false, false, at(-3*time.Hour)),
		task("bientot", domain.ImportanceCritique, false, false, at(30*time.Minute)),
	}
	got := ids(Priorities(tasks).CriticalOrHigh)
	want := []string{"enRetard", "bientot", "demain", "sansDate"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordre = %v, attendu %v", got, want)
		}
	}
}

func TestTopPriority(t *testing.T) {
	var tasks []domain.Task
	for i := 0; i < 8; i++ {
		tasks = append(tasks, task(string(rune('a'+i)), domain.ImportanceNormale, false, false, at(time.Duration(i)*time.Hour)))
	}
	if got := TopPriority(tasks, 5); len(got) != 5 {
		t.Errorf("%d tâches, 5 attendues", len(got))
	}
	if got := TopPriority(tasks[:2], 5); len(got) != 2 {
		t.Errorf("%d tâches, 2 attendues quand le stock est plus petit que n", len(got))
	}
}

// « Transverse / Divers » reste premier quel que soit son ordre de création (§2.1).
func TestSortProjects_LockedFirst(t *testing.T) {
	projects := []domain.Project{
		{ID: "p2", Name: "Alpha"},
		{ID: domain.DiversProjectID, Name: domain.DiversProjectName, Locked: true},
		{ID: "p3", Name: "beta"},
	}
	got := SortProjects(projects)
	if got[0].ID != domain.DiversProjectID {
		t.Errorf("premier = %s, attendu le projet verrouillé", got[0].ID)
	}
	if got[1].Name != "Alpha" || got[2].Name != "beta" {
		t.Errorf("ordre = %s, %s ; tri par nom insensible à la casse attendu", got[1].Name, got[2].Name)
	}
}

func TestSortProjects_DoesNotMutateInput(t *testing.T) {
	projects := []domain.Project{
		{ID: "p2", Name: "Alpha"},
		{ID: domain.DiversProjectID, Name: domain.DiversProjectName, Locked: true},
	}
	SortProjects(projects)
	if projects[0].ID != "p2" {
		t.Error("la tranche d'entrée a été réordonnée")
	}
}

func ids(tasks []domain.Task) []string {
	out := make([]string, len(tasks))
	for i, t := range tasks {
		out[i] = t.ID
	}
	return out
}

/* ---------------- Barre système (§2.10) ---------------- */

func TestTray_LevelFollowsImportance(t *testing.T) {
	cases := []struct {
		name  string
		tasks []domain.Task
		want  TrayLevel
	}{
		{"rien", nil, TrayLevelNormal},
		{"que des normales", []domain.Task{task("a", domain.ImportanceNormale, false, false, nil)}, TrayLevelNormal},
		{"une haute", []domain.Task{task("a", domain.ImportanceHaute, false, false, nil)}, TrayLevelHigh},
		{"une critique", []domain.Task{task("a", domain.ImportanceCritique, false, false, nil)}, TrayLevelCritical},
		{
			"le rouge prime sur l'orange",
			[]domain.Task{
				task("h", domain.ImportanceHaute, false, false, nil),
				task("c", domain.ImportanceCritique, false, false, nil),
			},
			TrayLevelCritical,
		},
		{
			"une critique terminée n'alerte plus",
			[]domain.Task{task("c", domain.ImportanceCritique, true, false, nil)},
			TrayLevelNormal,
		},
		{
			"une critique annulée non plus",
			[]domain.Task{task("c", domain.ImportanceCritique, false, true, nil)},
			TrayLevelNormal,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Tray(c.tasks, now).Level; got != c.want {
				t.Errorf("niveau = %q, attendu %q", got, c.want)
			}
		})
	}
}

func TestTray_Counts(t *testing.T) {
	tasks := []domain.Task{
		task("c1", domain.ImportanceCritique, false, false, nil),
		task("c2", domain.ImportanceCritique, false, false, nil),
		task("h1", domain.ImportanceHaute, false, false, nil),
		task("finie", domain.ImportanceCritique, true, false, nil),
	}
	s := Tray(tasks, now)
	if s.CriticalCount != 2 {
		t.Errorf("CriticalCount = %d, attendu 2", s.CriticalCount)
	}
	if s.HighCount != 1 {
		t.Errorf("HighCount = %d, attendu 1", s.HighCount)
	}
}

func TestTray_UrgentTasksListed(t *testing.T) {
	tasks := []domain.Task{
		task("urgente", domain.ImportanceCritique, false, false, at(2*time.Minute)),
		task("plusTard", domain.ImportanceCritique, false, false, at(2*time.Hour)),
	}
	s := Tray(tasks, now)
	if len(s.UrgentTasks) != 1 || s.UrgentTasks[0].ID != "urgente" {
		t.Errorf("UrgentTasks = %v, seule \"urgente\" attendue", ids(s.UrgentTasks))
	}
}

// Le clignotement reprend la condition de l'icône ⏰ (§2.3) : imminente OU en
// retard. Il vaut à tous les niveaux, y compris normal — l'horloge parle du
// temps, le niveau parle de l'importance, les deux sont indépendants.
func TestTray_BlinkingFollowsClockCondition(t *testing.T) {
	cases := []struct {
		name  string
		tasks []domain.Task
		want  bool
	}{
		{
			"en retard et critique",
			[]domain.Task{task("a", domain.ImportanceCritique, false, false, at(-3*time.Hour))},
			true,
		},
		{
			"en retard et normale : clignote quand même",
			[]domain.Task{task("a", domain.ImportanceNormale, false, false, at(-3*time.Hour))},
			true,
		},
		{
			"imminente et haute",
			[]domain.Task{task("a", domain.ImportanceHaute, false, false, at(2*time.Minute))},
			true,
		},
		{
			"échéance lointaine : pas de clignotement",
			[]domain.Task{task("a", domain.ImportanceCritique, false, false, at(72*time.Hour))},
			false,
		},
		{
			"critique sans échéance : pas de clignotement",
			[]domain.Task{task("a", domain.ImportanceCritique, false, false, nil)},
			false,
		},
		{
			"en retard mais terminée : plus rien",
			[]domain.Task{task("a", domain.ImportanceCritique, true, false, at(-3*time.Hour))},
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Tray(c.tasks, now).Blinking; got != c.want {
				t.Errorf("clignotement = %v, attendu %v", got, c.want)
			}
		})
	}
}

// La notification reste réservée au franchissement des 5 minutes : signaler à
// chaque démarrage une tâche en retard depuis trois jours serait du bruit.
func TestTray_NotifyOnlyOnImminent(t *testing.T) {
	tasks := []domain.Task{
		task("imminente", domain.ImportanceNormale, false, false, at(2*time.Minute)),
		task("enRetard", domain.ImportanceNormale, false, false, at(-3*time.Hour)),
	}
	s := Tray(tasks, now)

	// Les deux font clignoter…
	if len(s.UrgentTasks) != 2 {
		t.Errorf("UrgentTasks = %v, les deux attendues", ids(s.UrgentTasks))
	}
	// …mais une seule est notifiée.
	if len(s.TasksToNotify) != 1 || s.TasksToNotify[0].ID != "imminente" {
		t.Errorf("TasksToNotify = %v, seule \"imminente\" attendue", ids(s.TasksToNotify))
	}
}
