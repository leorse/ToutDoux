package sqlite

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"toutdoux/domain"
	"toutdoux/domain/notelayout"
)

func newNoteGroup(t *testing.T, r *NoteGroupRepository, id, projectID, name string) domain.NoteGroup {
	t.Helper()
	g := domain.NoteGroup{ID: id, ProjectID: projectID, Name: name, CreatedAt: time.Now().UTC().Truncate(time.Millisecond)}
	if err := r.Create(g); err != nil {
		t.Fatalf("création du groupe : %v", err)
	}
	return g
}

// ordre rend les identifiants des notes d'un projet, dans l'ordre de la liste.
func ordre(t *testing.T, r *NoteRepository, projectID string) []string {
	t.Helper()
	notes, err := r.ListByProject(projectID)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, n := range notes {
		ids = append(ids, n.ID)
	}
	return ids
}

// La migration 4 s'applique à une base 1.1.1 : les notes gardent l'ordre
// qu'elles avaient à l'écran (la plus récemment modifiée d'abord) et aucune
// n'est dans un groupe.
func TestMigration4_UpgradesFromVersion3(t *testing.T) {
	original := migrations
	migrations = original[:3]
	db := newTestDB(t)
	migrations = original

	newProject(t, NewProjectRepository(db), "p1", "Avant la migration 4")
	newProject(t, NewProjectRepository(db), "p2", "Autre projet")
	base := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	inserer := func(id, projet string, modifiee time.Time) {
		t.Helper()
		if _, err := db.Exec(
			`INSERT INTO notes (id, project_id, title, content, hidden, created_at, updated_at) VALUES (?, ?, ?, '', 0, ?, ?)`,
			id, projet, id, formatTime(base), formatTime(modifiee),
		); err != nil {
			t.Fatalf("insertion avant migration : %v", err)
		}
	}
	inserer("C", "p1", base)
	inserer("A", "p1", base.Add(2*time.Hour))
	inserer("B", "p1", base.Add(time.Hour))
	inserer("Z", "p2", base.Add(3*time.Hour))

	if err := db.migrate(); err != nil {
		t.Fatalf("migration vers la version 4 : %v", err)
	}

	notes, err := NewNoteRepository(db).ListByProject("p1")
	if err != nil {
		t.Fatal(err)
	}
	for i, attendu := range []string{"A", "B", "C"} {
		if notes[i].ID != attendu || notes[i].OrderIndex != i {
			t.Errorf("position %d : note %s (indice %d), attendu %s", i, notes[i].ID, notes[i].OrderIndex, attendu)
		}
		if notes[i].GroupID != nil {
			t.Errorf("la note %s ne doit être dans aucun groupe", notes[i].ID)
		}
	}
	// L'ordre d'un projet est indépendant des autres.
	if autre, _ := NewNoteRepository(db).Get("Z"); autre.OrderIndex != 0 {
		t.Errorf("indice de Z = %d, attendu 0", autre.OrderIndex)
	}

	if err := db.migrate(); err != nil {
		t.Fatalf("seconde migration : %v", err)
	}
}

// Une nouvelle note arrive en tête ; la modifier ou la cacher ne la déplace pas.
func TestNoteRepository_ManualOrder(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	notes := NewNoteRepository(db)
	newNote(t, notes, "A", "p1", "a")
	newNote(t, notes, "B", "p1", "b")
	newNote(t, notes, "C", "p1", "c")

	if got := ordre(t, notes, "p1"); !reflect.DeepEqual(got, []string{"C", "B", "A"}) {
		t.Fatalf("ordre = %v — la dernière créée doit être en tête", got)
	}

	a, _ := notes.Get("A")
	a.Content = "<p>modifiée</p>"
	if err := notes.Update(a); err != nil {
		t.Fatal(err)
	}
	if err := notes.SetHidden("B", true); err != nil {
		t.Fatal(err)
	}
	if got := ordre(t, notes, "p1"); !reflect.DeepEqual(got, []string{"C", "B", "A"}) {
		t.Errorf("ordre = %v — modifier ou cacher une note ne doit pas la déplacer", got)
	}
}

func TestNoteRepository_SaveLayout(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	notes := NewNoteRepository(db)
	groups := NewNoteGroupRepository(db)
	for _, id := range []string{"A", "B", "C"} {
		newNote(t, notes, id, "p1", id)
	}
	newNoteGroup(t, groups, "g1", "p1", "Sprint 12")
	newNoteGroup(t, groups, "g2", "p1", "Idées")

	layout := []notelayout.Item{{NoteID: "B"}, {NoteID: "A", GroupID: "g1"}, {NoteID: "C", GroupID: "g1"}}
	if err := notes.SaveLayout("p1", layout); err != nil {
		t.Fatalf("écriture de la disposition : %v", err)
	}

	relues, err := notes.ListByProject("p1")
	if err != nil {
		t.Fatal(err)
	}
	if got := notelayout.FromNotes(relues); !reflect.DeepEqual(got, layout) {
		t.Errorf("disposition relue = %v, attendu %v", got, layout)
	}

	// g2 n'a reçu aucune note : il disparaît dans la même écriture.
	restants, err := groups.ListByProject("p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(restants) != 1 || restants[0].ID != "g1" {
		t.Errorf("groupes restants = %v, attendu g1 seul", restants)
	}
}

func TestNoteGroupRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	notes := NewNoteRepository(db)
	groups := NewNoteGroupRepository(db)
	newNote(t, notes, "A", "p1", "a")
	newNoteGroup(t, groups, "g1", "p1", "Sprint 12")
	if err := notes.SaveLayout("p1", []notelayout.Item{{NoteID: "A", GroupID: "g1"}}); err != nil {
		t.Fatal(err)
	}

	if err := groups.Rename("g1", "Sprint 13"); err != nil {
		t.Fatalf("renommage : %v", err)
	}
	liste, err := groups.ListByProject("p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(liste) != 1 || liste[0].Name != "Sprint 13" {
		t.Errorf("groupes = %v, attendu « Sprint 13 »", liste)
	}

	if err := groups.Delete("g1"); err != nil {
		t.Fatalf("suppression : %v", err)
	}
	if _, err := groups.Get("g1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu ErrNotFound", err)
	}
	if a, _ := notes.Get("A"); a.GroupID != nil {
		t.Error("supprimer un groupe doit en détacher les notes, pas les emporter")
	}
	if err := groups.Rename("g1", "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("renommage d'un groupe absent : erreur = %v, attendu ErrNotFound", err)
	}
}

// Un groupe n'existe que tant qu'il contient une note.
func TestNoteRepository_DeleteLastNoteRemovesGroup(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	notes := NewNoteRepository(db)
	groups := NewNoteGroupRepository(db)
	newNote(t, notes, "A", "p1", "a")
	newNote(t, notes, "B", "p1", "b")
	newNoteGroup(t, groups, "g1", "p1", "Groupe")
	if err := notes.SaveLayout("p1", []notelayout.Item{{NoteID: "A", GroupID: "g1"}, {NoteID: "B", GroupID: "g1"}}); err != nil {
		t.Fatal(err)
	}

	if err := notes.Delete("A"); err != nil {
		t.Fatal(err)
	}
	if _, err := groups.Get("g1"); err != nil {
		t.Fatalf("le groupe garde une note, il doit subsister : %v", err)
	}
	if err := notes.Delete("B"); err != nil {
		t.Fatal(err)
	}
	if _, err := groups.Get("g1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v — le groupe vidé de sa dernière note doit disparaître", err)
	}
}

func TestProjectRepository_DeleteCascadesNoteGroups(t *testing.T) {
	db := newTestDB(t)
	projects := NewProjectRepository(db)
	notes := NewNoteRepository(db)
	groups := NewNoteGroupRepository(db)
	newProject(t, projects, "p1", "À supprimer")
	newNote(t, notes, "A", "p1", "a")
	newNoteGroup(t, groups, "g1", "p1", "Groupe")
	if err := notes.SaveLayout("p1", []notelayout.Item{{NoteID: "A", GroupID: "g1"}}); err != nil {
		t.Fatal(err)
	}

	if err := projects.Delete("p1"); err != nil {
		t.Fatalf("suppression : %v", err)
	}
	if got := ordre(t, notes, "p1"); len(got) != 0 {
		t.Errorf("%d notes restantes, 0 attendue", len(got))
	}
	if restants, _ := groups.ListByProject("p1"); len(restants) != 0 {
		t.Errorf("%d groupes restants, 0 attendu", len(restants))
	}
}
