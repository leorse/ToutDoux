package sqlite

import (
	"errors"
	"testing"

	"toutdoux/domain"
)

// La migration 5 ajoute la couleur à des notes et réunions déjà écrites : elles
// restent sans couleur.
func TestMigration5_UpgradesFromVersion4(t *testing.T) {
	original := migrations
	migrations = original[:4]
	db := newTestDB(t)
	migrations = original

	newProject(t, NewProjectRepository(db), "p1", "Avant la migration 5")
	now := formatTime(nowUTC())
	if _, err := db.Exec(
		`INSERT INTO notes (id, project_id, title, content, hidden, order_index, created_at, updated_at) VALUES ('n1', 'p1', 'note', '', 0, 0, ?, ?)`,
		now, now,
	); err != nil {
		t.Fatalf("insertion de la note avant migration : %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO meetings (id, project_id, title, hidden, created_at, updated_at) VALUES ('m1', 'p1', 'réunion', 0, ?, ?)`,
		now, now,
	); err != nil {
		t.Fatalf("insertion de la réunion avant migration : %v", err)
	}

	if err := db.migrate(); err != nil {
		t.Fatalf("migration vers la version 5 : %v", err)
	}

	n, err := NewNoteRepository(db).Get("n1")
	if err != nil {
		t.Fatal(err)
	}
	if n.Color != "" {
		t.Errorf("couleur de la note = %q, attendu aucune", n.Color)
	}
	m, err := NewMeetingRepository(db).Get("m1")
	if err != nil {
		t.Fatal(err)
	}
	if m.Color != "" {
		t.Errorf("couleur de la réunion = %q, attendu aucune", m.Color)
	}

	if err := db.migrate(); err != nil {
		t.Fatalf("seconde migration : %v", err)
	}
}

func TestNoteRepository_SetColor(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	repo := NewNoteRepository(db)
	avant := newNote(t, repo, "n1", "p1", "Note")

	if err := repo.SetColor("n1", "menthe"); err != nil {
		t.Fatalf("couleur : %v", err)
	}
	relu, err := repo.Get("n1")
	if err != nil {
		t.Fatal(err)
	}
	if relu.Color != "menthe" {
		t.Errorf("couleur = %q, attendu « menthe »", relu.Color)
	}
	if !relu.UpdatedAt.Equal(avant.UpdatedAt) {
		t.Error("colorer une note ne doit pas toucher updated_at")
	}

	// La sauvegarde du contenu ne doit pas effacer la couleur.
	relu.Content = "<p>modifiée</p>"
	if err := repo.Update(relu); err != nil {
		t.Fatal(err)
	}
	if apres, _ := repo.Get("n1"); apres.Color != "menthe" {
		t.Errorf("couleur après mise à jour = %q, attendu « menthe »", apres.Color)
	}

	if err := repo.SetColor("inconnue", "menthe"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu domain.ErrNotFound", err)
	}
}

func TestMeetingRepository_SetColor(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	repo := NewMeetingRepository(db)
	avant := newMeeting(t, repo, "m1", "p1", "Réunion")

	if err := repo.SetColor("m1", "lilas"); err != nil {
		t.Fatalf("couleur : %v", err)
	}
	relu, err := repo.Get("m1")
	if err != nil {
		t.Fatal(err)
	}
	if relu.Color != "lilas" {
		t.Errorf("couleur = %q, attendu « lilas »", relu.Color)
	}
	if !relu.UpdatedAt.Equal(avant.UpdatedAt) {
		t.Error("colorer une réunion ne doit pas toucher updated_at")
	}
	liste, err := repo.ListByProject("p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(liste) != 1 || liste[0].Color != "lilas" {
		t.Errorf("liste = %+v, attendu une réunion lilas", liste)
	}

	if err := repo.SetColor("inconnue", "lilas"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu domain.ErrNotFound", err)
	}
}
