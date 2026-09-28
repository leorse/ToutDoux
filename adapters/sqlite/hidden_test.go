package sqlite

import (
	"errors"
	"testing"
	"time"

	"toutdoux/domain"
)

func newMeeting(t *testing.T, r *MeetingRepository, id, projectID, title string) domain.Meeting {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	m := domain.Meeting{ID: id, ProjectID: projectID, Title: title, CreatedAt: now, UpdatedAt: now}
	if err := r.Create(m); err != nil {
		t.Fatalf("création de la réunion : %v", err)
	}
	return m
}

func newNote(t *testing.T, r *NoteRepository, id, projectID, title string) domain.Note {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	n := domain.Note{ID: id, ProjectID: projectID, Title: title, CreatedAt: now, UpdatedAt: now}
	if err := r.Create(n); err != nil {
		t.Fatalf("création de la note : %v", err)
	}
	return n
}

func TestProjectRepository_SetHidden(t *testing.T) {
	db := newTestDB(t)
	repo := NewProjectRepository(db)
	p := newProject(t, repo, "p1", "À masquer")

	if err := repo.SetHidden(p.ID, true); err != nil {
		t.Fatalf("masquage : %v", err)
	}
	relu, err := repo.Get(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !relu.Hidden {
		t.Error("le projet devrait être marqué caché")
	}
	if !relu.UpdatedAt.Equal(p.UpdatedAt) {
		t.Errorf("updated_at a changé : %v -> %v", p.UpdatedAt, relu.UpdatedAt)
	}

	if err := repo.SetHidden(p.ID, false); err != nil {
		t.Fatalf("réaffichage : %v", err)
	}
	relu, err = repo.Get(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if relu.Hidden {
		t.Error("le projet devrait être visible après réaffichage")
	}
}

func TestProjectRepository_SetHidden_UnknownID(t *testing.T) {
	repo := NewProjectRepository(newTestDB(t))
	if err := repo.SetHidden("inconnu", true); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu domain.ErrNotFound", err)
	}
}

func TestNoteRepository_SetHidden(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	repo := NewNoteRepository(db)
	n := newNote(t, repo, "n1", "p1", "À masquer")

	if err := repo.SetHidden(n.ID, true); err != nil {
		t.Fatalf("masquage : %v", err)
	}
	relu, err := repo.Get(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !relu.Hidden {
		t.Error("la note devrait être marquée cachée")
	}
	if !relu.UpdatedAt.Equal(n.UpdatedAt) {
		t.Errorf("updated_at a changé : %v -> %v", n.UpdatedAt, relu.UpdatedAt)
	}
}

func TestNoteRepository_SetHidden_UnknownID(t *testing.T) {
	repo := NewNoteRepository(newTestDB(t))
	if err := repo.SetHidden("inconnue", true); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu domain.ErrNotFound", err)
	}
}

func TestMeetingRepository_SetHidden(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	repo := NewMeetingRepository(db)
	m := newMeeting(t, repo, "m1", "p1", "À masquer")

	if err := repo.SetHidden(m.ID, true); err != nil {
		t.Fatalf("masquage : %v", err)
	}
	relu, err := repo.Get(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !relu.Hidden {
		t.Error("la réunion devrait être marquée cachée")
	}
	if !relu.UpdatedAt.Equal(m.UpdatedAt) {
		t.Errorf("updated_at a changé : %v -> %v", m.UpdatedAt, relu.UpdatedAt)
	}
}

func TestMeetingRepository_SetHidden_UnknownID(t *testing.T) {
	repo := NewMeetingRepository(newTestDB(t))
	if err := repo.SetHidden("inconnue", true); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu domain.ErrNotFound", err)
	}
}
