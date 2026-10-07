package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"toutdoux/domain"
)

/* ---------------- Couleur des notes et des réunions (v1.2.0) ---------------- */

func TestSetNoteColor(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B")
	avant, err := app.GetSidebarStats(p)
	if err != nil {
		t.Fatal(err)
	}

	maj, err := app.SetNoteColor(ids["B"], "ciel")
	if err != nil {
		t.Fatal(err)
	}
	if maj.Color != "ciel" {
		t.Errorf("couleur = %q, attendu « ciel »", maj.Color)
	}

	// Purement visuel : ni la place dans la liste ni les compteurs ne bougent,
	// et la sauvegarde automatique ne doit pas effacer la couleur.
	attendre(t, app, p, "A", "B")
	if apres, _ := app.GetSidebarStats(p); !reflect.DeepEqual(avant, apres) {
		t.Errorf("compteurs modifiés par la couleur : %+v → %+v", avant, apres)
	}
	if _, err := app.UpdateNote(ids["B"], "B", "<p>texte</p>"); err != nil {
		t.Fatal(err)
	}
	notes, _ := app.GetNotes(p)
	if notes[1].Color != "ciel" {
		t.Errorf("couleur après modification du contenu = %q, attendu « ciel »", notes[1].Color)
	}

	maj, err = app.SetNoteColor(ids["B"], "")
	if err != nil {
		t.Fatal(err)
	}
	if maj.Color != "" {
		t.Errorf("couleur = %q, attendu aucune", maj.Color)
	}
}

func TestSetNoteColor_Refusals(t *testing.T) {
	app := newTestApp(t)
	_, ids := projetAvecNotes(t, app, "A")

	if _, err := app.SetNoteColor(ids["A"], "#ff0000"); !errors.Is(err, domain.ErrInvalidColor) {
		t.Errorf("couleur hors palette : erreur = %v, attendu ErrInvalidColor", err)
	}
	if _, err := app.SetNoteColor("inconnue", "rose"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("note inconnue : erreur = %v, attendu ErrNotFound", err)
	}
}

func TestSetMeetingColor(t *testing.T) {
	app := newTestApp(t)
	p, _ := app.CreateProject("Projet")
	m, err := app.CreateMeeting(p.ID, "Comité")
	if err != nil {
		t.Fatal(err)
	}
	if m.Color != "" {
		t.Errorf("une réunion neuve ne doit pas avoir de couleur, obtenu %q", m.Color)
	}

	maj, err := app.SetMeetingColor(m.ID, "jaune")
	if err != nil {
		t.Fatal(err)
	}
	if maj.Color != "jaune" {
		t.Errorf("couleur = %q, attendu « jaune »", maj.Color)
	}
	// Renommer ne doit pas effacer la couleur.
	if _, err := app.RenameMeeting(m.ID, "Comité de pilotage"); err != nil {
		t.Fatal(err)
	}
	liste, _ := app.GetMeetings(p.ID)
	if len(liste) != 1 || liste[0].Color != "jaune" {
		t.Errorf("réunions = %+v, attendu une réunion jaune", liste)
	}

	if _, err := app.SetMeetingColor(m.ID, "fuchsia"); !errors.Is(err, domain.ErrInvalidColor) {
		t.Errorf("couleur hors palette : erreur = %v, attendu ErrInvalidColor", err)
	}
	if _, err := app.SetMeetingColor("inconnue", "rose"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("réunion inconnue : erreur = %v, attendu ErrNotFound", err)
	}
}

func TestValidItemColor(t *testing.T) {
	if len(domain.ItemColors) != 10 {
		t.Errorf("%d couleurs dans la palette, 10 attendues", len(domain.ItemColors))
	}
	for _, c := range append([]string{""}, domain.ItemColors...) {
		if !domain.ValidItemColor(c) {
			t.Errorf("%q devrait être acceptée", c)
		}
	}
	for _, c := range []string{"rouge", "#fecaca", "Rose", " rose"} {
		if domain.ValidItemColor(c) {
			t.Errorf("%q ne devrait pas être acceptée", c)
		}
	}
}

/* ---------------- Échéance d'une tâche terminée (v1.2.0) ---------------- */

// Une tâche terminée ou annulée n'a plus de délai affiché : ni temps restant,
// ni retard. La date, elle, est conservée, et le délai revient si la tâche
// redevient active.
func TestGetTaskView_NoDueInfoOnFinishedTasks(t *testing.T) {
	app := newTestApp(t)
	p, _ := app.CreateProject("Projet")

	avecEcheance := func(nom string, dans time.Duration) string {
		t.Helper()
		res, err := app.CreateTask(p.ID, "", nom)
		if err != nil {
			t.Fatal(err)
		}
		due := app.clock.Now().Add(dans).Format(time.RFC3339)
		if _, err := app.UpdateTask(res.Task.ID, TaskPatch{DueDate: &due}); err != nil {
			t.Fatal(err)
		}
		return res.Task.ID
	}
	active := avecEcheance("Active", 72*time.Hour)
	terminee := avecEcheance("Terminée en retard", -48*time.Hour)
	annulee := avecEcheance("Annulée", -48*time.Hour)

	if _, err := app.ToggleTaskCompleted(terminee); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ToggleTaskCancelled(annulee); err != nil {
		t.Fatal(err)
	}

	tous := FilterSelection{
		Status:     []string{"active", "completed", "cancelled"},
		Importance: []string{"Basse", "Normale", "Haute", "Critique"},
	}
	vue, err := app.GetTaskView(p.ID, tous)
	if err != nil {
		t.Fatal(err)
	}
	if vue.Due[active] == nil {
		t.Error("une tâche active garde son délai")
	}
	if info := vue.Due[terminee]; info != nil {
		t.Errorf("tâche terminée : délai %q affiché, aucun attendu", info.Text)
	}
	if info := vue.Due[annulee]; info != nil {
		t.Errorf("tâche annulée : délai %q affiché, aucun attendu", info.Text)
	}
	for _, task := range vue.Tasks {
		if task.ID == terminee && task.DueDate == nil {
			t.Error("la date d'échéance d'une tâche terminée doit être conservée")
		}
	}

	// Décochée, la tâche retrouve son retard.
	if _, err := app.ToggleTaskCompleted(terminee); err != nil {
		t.Fatal(err)
	}
	vue, err = app.GetTaskView(p.ID, tous)
	if err != nil {
		t.Fatal(err)
	}
	if info := vue.Due[terminee]; info == nil || !info.Overdue {
		t.Errorf("tâche rouverte : délai = %+v, attendu un retard", info)
	}
}
