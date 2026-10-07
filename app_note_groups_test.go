package main

import (
	"errors"
	"reflect"
	"testing"

	"toutdoux/domain"
	"toutdoux/domain/notelayout"
)

/* ---------------- Groupes et ordre des notes (v1.2.0) ---------------- */

// projetAvecNotes crée un projet et des notes dont la liste se lit dans
// l'ordre des titres donnés. Rend l'identifiant du projet et ceux des notes,
// par titre.
func projetAvecNotes(t *testing.T, app *App, titres ...string) (string, map[string]string) {
	t.Helper()
	p, err := app.CreateProject("Projet")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	// Chaque note créée passe en tête : on crée donc à rebours.
	for i := len(titres) - 1; i >= 0; i-- {
		n, err := app.CreateNote(p.ID, titres[i])
		if err != nil {
			t.Fatal(err)
		}
		ids[titres[i]] = n.ID
	}
	return p.ID, ids
}

// disposition rend la liste sous la forme « titre » ou « titre:nom du groupe ».
func disposition(t *testing.T, app *App, projectID string) []string {
	t.Helper()
	notes, err := app.GetNotes(projectID)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := app.GetNoteGroups(projectID)
	if err != nil {
		t.Fatal(err)
	}
	noms := map[string]string{}
	for _, g := range groups {
		noms[g.ID] = g.Name
	}
	out := []string{}
	for _, n := range notes {
		if n.GroupID == nil {
			out = append(out, n.Title)
			continue
		}
		out = append(out, n.Title+":"+noms[*n.GroupID])
	}
	return out
}

func attendre(t *testing.T, app *App, projectID string, attendu ...string) {
	t.Helper()
	if got := disposition(t, app, projectID); !reflect.DeepEqual(got, attendu) {
		t.Errorf("liste = %v, attendu %v", got, attendu)
	}
}

func TestCreateNote_PlacedFirstOutsideGroups(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B")
	if _, err := app.GroupNotes(p, "G", []string{ids["A"]}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateNote(p, "Nouvelle"); err != nil {
		t.Fatal(err)
	}
	attendre(t, app, p, "Nouvelle", "A:G", "B")
}

func TestGroupNotes(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B", "C", "D")

	g, err := app.GroupNotes(p, "  Sprint 12  ", []string{ids["D"], ids["A"], ids["C"]})
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "Sprint 12" {
		t.Errorf("nom = %q, attendu sans les espaces autour", g.Name)
	}
	attendre(t, app, p, "A:Sprint 12", "C:Sprint 12", "D:Sprint 12", "B")
}

// Une note reprise à un autre groupe le quitte ; un groupe vidé disparaît.
func TestGroupNotes_TakesNotesFromAnotherGroup(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B", "C")
	if _, err := app.GroupNotes(p, "G1", []string{ids["A"], ids["B"]}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.GroupNotes(p, "G2", []string{ids["B"], ids["C"]}); err != nil {
		t.Fatal(err)
	}
	attendre(t, app, p, "A:G1", "B:G2", "C:G2")

	if _, err := app.GroupNotes(p, "G3", []string{ids["A"]}); err != nil {
		t.Fatal(err)
	}
	groups, _ := app.GetNoteGroups(p)
	if len(groups) != 2 {
		t.Errorf("%d groupes, 2 attendus — G1, vidé, doit avoir disparu", len(groups))
	}
}

func TestGroupNotes_Refusals(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A")

	if _, err := app.GroupNotes(p, "   ", []string{ids["A"]}); !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("nom blanc : erreur = %v, attendu ErrEmptyName", err)
	}
	if _, err := app.GroupNotes(p, "G", []string{"inconnue"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("aucune note du projet : erreur = %v, attendu ErrNotFound", err)
	}
	if groups, _ := app.GetNoteGroups(p); len(groups) != 0 {
		t.Errorf("%d groupes créés malgré le refus", len(groups))
	}
}

func TestRenameNoteGroup(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B")
	g, _ := app.GroupNotes(p, "Sprint 12", []string{ids["B"]})

	maj, err := app.RenameNoteGroup(g.ID, " Sprint 13 ")
	if err != nil {
		t.Fatal(err)
	}
	if maj.Name != "Sprint 13" {
		t.Errorf("nom = %q", maj.Name)
	}
	attendre(t, app, p, "A", "B:Sprint 13")

	if _, err := app.RenameNoteGroup(g.ID, ""); !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("nom vide : erreur = %v, attendu ErrEmptyName", err)
	}
	attendre(t, app, p, "A", "B:Sprint 13")
}

func TestDissolveNoteGroup(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B", "C", "D")
	g, _ := app.GroupNotes(p, "G", []string{ids["B"], ids["C"]})

	if err := app.DissolveNoteGroup(g.ID); err != nil {
		t.Fatal(err)
	}
	attendre(t, app, p, "A", "B", "C", "D")
	if groups, _ := app.GetNoteGroups(p); len(groups) != 0 {
		t.Errorf("%d groupes restants, 0 attendu", len(groups))
	}
}

func TestSetNotesLayout(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B", "C")
	g, _ := app.GroupNotes(p, "G", []string{ids["A"]})

	// C rejoint le groupe, B passe en tête.
	err := app.SetNotesLayout(p, []notelayout.Item{
		{NoteID: ids["B"]},
		{NoteID: ids["A"], GroupID: g.ID},
		{NoteID: ids["C"], GroupID: g.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	attendre(t, app, p, "B", "A:G", "C:G")

	// Disposition incohérente : refusée, rien n'est écrit.
	err = app.SetNotesLayout(p, []notelayout.Item{
		{NoteID: ids["A"], GroupID: g.ID},
		{NoteID: ids["B"]},
		{NoteID: ids["C"], GroupID: g.ID},
	})
	if !errors.Is(err, domain.ErrInvalidLayout) {
		t.Errorf("erreur = %v, attendu ErrInvalidLayout", err)
	}
	if err := app.SetNotesLayout(p, []notelayout.Item{{NoteID: ids["A"]}}); !errors.Is(err, domain.ErrInvalidLayout) {
		t.Errorf("notes manquantes : erreur = %v, attendu ErrInvalidLayout", err)
	}
	attendre(t, app, p, "B", "A:G", "C:G")

	// Sortir la dernière note d'un groupe le fait disparaître.
	err = app.SetNotesLayout(p, []notelayout.Item{{NoteID: ids["B"]}, {NoteID: ids["A"]}, {NoteID: ids["C"]}})
	if err != nil {
		t.Fatal(err)
	}
	if groups, _ := app.GetNoteGroups(p); len(groups) != 0 {
		t.Errorf("%d groupes restants, 0 attendu", len(groups))
	}
}

// Modifier une note ne la remonte plus en tête de liste.
func TestUpdateNote_KeepsPosition(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B", "C")
	if _, err := app.UpdateNote(ids["C"], "C", "<p>modifiée</p>"); err != nil {
		t.Fatal(err)
	}
	attendre(t, app, p, "A", "B", "C")
}

func TestDeleteNote_LastOfGroupRemovesGroup(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B")
	if _, err := app.GroupNotes(p, "G", []string{ids["A"]}); err != nil {
		t.Fatal(err)
	}
	if err := app.DeleteNote(ids["A"]); err != nil {
		t.Fatal(err)
	}
	if groups, _ := app.GetNoteGroups(p); len(groups) != 0 {
		t.Errorf("%d groupes restants, 0 attendu", len(groups))
	}
}

// Grouper est une affaire d'affichage : les compteurs de la sidebar n'en
// dépendent pas, et un projet à groupes se supprime comme un autre.
func TestNoteGroups_CountersAndProjectDeletion(t *testing.T) {
	app := newTestApp(t)
	p, ids := projetAvecNotes(t, app, "A", "B", "C")
	avant, err := app.GetSidebarStats(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.GroupNotes(p, "G", []string{ids["A"], ids["B"], ids["C"]}); err != nil {
		t.Fatal(err)
	}
	apres, err := app.GetSidebarStats(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(avant, apres) {
		t.Errorf("compteurs modifiés par le regroupement : %+v → %+v", avant, apres)
	}

	if err := app.DeleteProject(p); err != nil {
		t.Fatalf("suppression du projet : %v", err)
	}
	if groups, _ := app.GetNoteGroups(p); len(groups) != 0 {
		t.Errorf("%d groupes restants après suppression du projet", len(groups))
	}
}
