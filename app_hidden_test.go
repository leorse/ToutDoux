package main

import (
	"errors"
	"testing"

	"toutdoux/domain"
)

/* ---------------- Bindings de masquage (§2.1, §2.6, §2.7) ---------------- */

func TestSetProjectHidden(t *testing.T) {
	app := newTestApp(t)
	p, err := app.CreateProject("Obsolète")
	if err != nil {
		t.Fatal(err)
	}

	maj, err := app.SetProjectHidden(p.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !maj.Hidden {
		t.Error("le projet devrait être marqué caché")
	}

	maj, err = app.SetProjectHidden(p.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if maj.Hidden {
		t.Error("le projet devrait être réaffiché")
	}
}

// Le projet verrouillé ne peut pas être caché, comme il ne peut être ni
// renommé ni supprimé (§2.1).
func TestSetProjectHidden_LockedProject(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.SetProjectHidden(domain.DiversProjectID, true); !errors.Is(err, domain.ErrProjectLocked) {
		t.Errorf("erreur = %v, attendu ErrProjectLocked", err)
	}
}

func TestSetNoteHidden(t *testing.T) {
	app := newTestApp(t)
	p, _ := app.CreateProject("Projet")
	n, err := app.CreateNote(p.ID, "Note")
	if err != nil {
		t.Fatal(err)
	}

	maj, err := app.SetNoteHidden(n.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !maj.Hidden {
		t.Error("la note devrait être marquée cachée")
	}
}

func TestSetMeetingHidden(t *testing.T) {
	app := newTestApp(t)
	p, _ := app.CreateProject("Projet")
	m, err := app.CreateMeeting(p.ID, "Réunion")
	if err != nil {
		t.Fatal(err)
	}

	maj, err := app.SetMeetingHidden(m.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !maj.Hidden {
		t.Error("la réunion devrait être marquée cachée")
	}
}

/* ---------------- Masquage sans effet ailleurs (§2.1) ---------------- */

// Une tâche critique d'un projet caché reste dans les Priorités et dans la
// lecture que fait la barre système (tasks.ListAll) : le masquage est
// purement visuel.
func TestHiddenProject_StillInPrioritiesAndTray(t *testing.T) {
	app := newTestApp(t)
	p, err := app.CreateProject("À cacher")
	if err != nil {
		t.Fatal(err)
	}
	res, err := app.CreateTask(p.ID, "", "Tâche critique")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateTask(res.Task.ID, TaskPatch{Importance: strPtr(string(domain.ImportanceCritique))}); err != nil {
		t.Fatal(err)
	}

	if _, err := app.SetProjectHidden(p.ID, true); err != nil {
		t.Fatal(err)
	}

	priorites, err := app.GetPriorityTasks()
	if err != nil {
		t.Fatal(err)
	}
	if !contientTache(priorites.CriticalOrHigh, res.Task.ID) {
		t.Error("la tâche du projet caché devrait rester dans les Priorités")
	}

	toutes, err := app.tasks.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if !contientTache(toutes, res.Task.ID) {
		t.Error("la tâche du projet caché devrait rester lue par la barre système (ListAll)")
	}
}

// La recherche continue de trouver une note cachée (§2.6, §2.9).
func TestHiddenNote_StillInSearch(t *testing.T) {
	app := newTestApp(t)
	p, err := app.CreateProject("Projet")
	if err != nil {
		t.Fatal(err)
	}
	n, err := app.CreateNote(p.ID, "Réunion de cadrage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateNote(n.ID, n.Title, "contenu très particulier à chercher"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.SetNoteHidden(n.ID, true); err != nil {
		t.Fatal(err)
	}

	resultats, err := app.SearchGlobal("particulier")
	if err != nil {
		t.Fatal(err)
	}
	trouve := false
	for _, r := range resultats {
		if r.Note != nil && r.Note.ID == n.ID {
			trouve = true
		}
	}
	if !trouve {
		t.Error("la note cachée devrait rester trouvable par la recherche")
	}
}

func strPtr(s string) *string { return &s }

func contientTache(tasks []domain.Task, id string) bool {
	for _, t := range tasks {
		if t.ID == id {
			return true
		}
	}
	return false
}
