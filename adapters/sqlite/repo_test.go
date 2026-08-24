package sqlite

import (
	"errors"
	"testing"
	"time"

	"toutdoux/domain"
)

// Ces tests tournent contre une vraie SQLite en mémoire, sans mock ni port
// simulé (§3.10, niveau 2). L'objectif est le round-trip : ce qu'on écrit doit
// se relire à l'identique. Il n'est pas de vérifier que SQLite fonctionne.

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func ptr(s string) *string { return &s }

func newProject(t *testing.T, r *ProjectRepository, id, name string) domain.Project {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	p := domain.Project{ID: id, Name: name, CreatedAt: now, UpdatedAt: now}
	if err := r.Create(p); err != nil {
		t.Fatalf("création du projet : %v", err)
	}
	return p
}

func newTask(t *testing.T, r *TaskRepository, id, projectID string, parent *string) domain.Task {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	task := domain.Task{
		ID: id, ProjectID: projectID, ParentID: parent, Name: id,
		Importance: domain.ImportanceNormale, CreatedAt: now, UpdatedAt: now,
	}
	if err := r.Create(task); err != nil {
		t.Fatalf("création de la tâche %s : %v", id, err)
	}
	return task
}

// Le projet verrouillé est créé au premier démarrage, avec un id stable (§2.1).
func TestOpen_CreatesDiversProject(t *testing.T) {
	db := newTestDB(t)
	repo := NewProjectRepository(db)

	p, err := repo.Get(domain.DiversProjectID)
	if err != nil {
		t.Fatalf("le projet Divers doit exister : %v", err)
	}
	if !p.Locked {
		t.Error("le projet Divers doit être verrouillé")
	}
	if p.Name != domain.DiversProjectName {
		t.Errorf("nom = %q, attendu %q", p.Name, domain.DiversProjectName)
	}
}

// Rouvrir la base ne doit pas créer un second projet Divers ni rejouer les
// migrations.
func TestOpen_IsIdempotent(t *testing.T) {
	db := newTestDB(t)
	if err := db.migrate(); err != nil {
		t.Fatalf("seconde migration : %v", err)
	}
	if err := db.ensureDiversProject(); err != nil {
		t.Fatalf("second ensureDiversProject : %v", err)
	}
	projects, err := NewProjectRepository(db).List()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Errorf("%d projets, 1 attendu", len(projects))
	}
}

func TestProjectRepository_RoundTrip(t *testing.T) {
	repo := NewProjectRepository(newTestDB(t))
	created := newProject(t, repo, "p1", "Migration 2026")

	got, err := repo.Get("p1")
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	if got.Name != created.Name {
		t.Errorf("nom = %q, attendu %q", got.Name, created.Name)
	}
	if got.Locked {
		t.Error("un projet ordinaire ne doit pas être verrouillé")
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("créé le %v, attendu %v — la date ne survit pas à l'aller-retour", got.CreatedAt, created.CreatedAt)
	}
}

func TestProjectRepository_NotFound(t *testing.T) {
	repo := NewProjectRepository(newTestDB(t))
	if _, err := repo.Get("fantome"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu ErrNotFound", err)
	}
	if err := repo.Rename("fantome", "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("renommage : erreur = %v, attendu ErrNotFound", err)
	}
	if err := repo.Delete("fantome"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("suppression : erreur = %v, attendu ErrNotFound", err)
	}
}

// L'unicité doit être insensible à la casse y compris sur les accents, ce que
// le LOWER() ASCII de SQLite ne saurait pas faire (§2.1).
func TestProjectRepository_ExistsByName_AccentAwareCase(t *testing.T) {
	repo := NewProjectRepository(newTestDB(t))
	newProject(t, repo, "p1", "Réunion Générale")

	for _, name := range []string{"Réunion Générale", "réunion générale", "RÉUNION GÉNÉRALE", "  Réunion Générale  "} {
		exists, err := repo.ExistsByName(name, "")
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("%q devrait être détecté comme doublon", name)
		}
	}
	exists, err := repo.ExistsByName("Autre chose", "")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("un nom distinct ne doit pas être signalé comme doublon")
	}
}

// Un projet qu'on renomme ne doit pas entrer en conflit avec lui-même.
func TestProjectRepository_ExistsByName_ExcludesSelf(t *testing.T) {
	repo := NewProjectRepository(newTestDB(t))
	newProject(t, repo, "p1", "Migration")

	exists, err := repo.ExistsByName("Migration", "p1")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("le projet en cours de renommage doit être exclu de la vérification")
	}
}

// La suppression d'un projet emporte tâches et notes (§2.1).
func TestProjectRepository_DeleteCascades(t *testing.T) {
	db := newTestDB(t)
	projects := NewProjectRepository(db)
	tasks := NewTaskRepository(db)
	notes := NewNoteRepository(db)

	newProject(t, projects, "p1", "À supprimer")
	newTask(t, tasks, "t1", "p1", nil)
	newTask(t, tasks, "t2", "p1", ptr("t1"))
	now := time.Now().UTC()
	if err := notes.Create(domain.Note{ID: "n1", ProjectID: "p1", Title: "note", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}

	if err := projects.Delete("p1"); err != nil {
		t.Fatalf("suppression : %v", err)
	}

	remaining, err := tasks.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Errorf("%d tâches restantes, 0 attendue", len(remaining))
	}
	got, err := notes.ListByProject("p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("%d notes restantes, 0 attendue", len(got))
	}
}

func TestTaskRepository_RoundTripWithNullables(t *testing.T) {
	db := newTestDB(t)
	NewProjectRepository(db)
	tasks := NewTaskRepository(db)
	newProject(t, NewProjectRepository(db), "p1", "Projet")

	due := time.Date(2026, 8, 25, 14, 30, 0, 0, time.UTC)
	now := time.Now().UTC()
	original := domain.Task{
		ID: "t1", ProjectID: "p1", ParentID: nil,
		Name: "Préparer la réunion", Description: "avec accents : échéance",
		Importance: domain.ImportanceCritique, DueDate: &due, OrderIndex: 3,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := tasks.Create(original); err != nil {
		t.Fatalf("création : %v", err)
	}

	got, err := tasks.Get("t1")
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	if got.ParentID != nil {
		t.Errorf("ParentID = %v, nil attendu pour une racine", got.ParentID)
	}
	if got.DueDate == nil || !got.DueDate.Equal(due) {
		t.Errorf("échéance = %v, attendu %v", got.DueDate, due)
	}
	if got.Name != original.Name || got.Description != original.Description {
		t.Errorf("texte accentué altéré : %q / %q", got.Name, got.Description)
	}
	if got.Importance != domain.ImportanceCritique {
		t.Errorf("importance = %q", got.Importance)
	}
	if got.OrderIndex != 3 {
		t.Errorf("order_index = %d, attendu 3", got.OrderIndex)
	}

	// Une tâche sans échéance doit relire nil, pas l'instant zéro.
	newTask(t, tasks, "t2", "p1", nil)
	t2, err := tasks.Get("t2")
	if err != nil {
		t.Fatal(err)
	}
	if t2.DueDate != nil {
		t.Errorf("échéance = %v, nil attendu", t2.DueDate)
	}
}

// La contrainte CHECK du schéma doit rejeter une importance inventée (§3.2).
func TestTaskRepository_RejectsInvalidImportance(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	tasks := NewTaskRepository(db)

	now := time.Now().UTC()
	err := tasks.Create(domain.Task{
		ID: "t1", ProjectID: "p1", Name: "x", Importance: "Urgentissime",
		CreatedAt: now, UpdatedAt: now,
	})
	if err == nil {
		t.Error("la contrainte CHECK doit rejeter une importance hors des quatre valeurs")
	}
}

// La clé étrangère doit refuser une tâche rattachée à un projet inexistant.
func TestTaskRepository_ForeignKeyEnforced(t *testing.T) {
	tasks := NewTaskRepository(newTestDB(t))
	now := time.Now().UTC()
	err := tasks.Create(domain.Task{
		ID: "t1", ProjectID: "projet-inexistant", Name: "x",
		Importance: domain.ImportanceNormale, CreatedAt: now, UpdatedAt: now,
	})
	if err == nil {
		t.Error("PRAGMA foreign_keys doit être actif et rejeter cette insertion")
	}
}

func TestTaskRepository_ListByProjectOrdersByIndex(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	tasks := NewTaskRepository(db)

	now := time.Now().UTC()
	for _, spec := range []struct {
		id    string
		order int
	}{{"c", 2}, {"a", 0}, {"b", 1}} {
		if err := tasks.Create(domain.Task{
			ID: spec.id, ProjectID: "p1", Name: spec.id, OrderIndex: spec.order,
			Importance: domain.ImportanceNormale, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := tasks.ListByProject("p1")
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"a", "b", "c"} {
		if got[i].ID != want {
			t.Errorf("position %d = %s, attendu %s", i, got[i].ID, want)
		}
	}
}

// Suppression en cascade sur trois niveaux : l'ordre doit respecter la clé
// étrangère parent_id, quelle que soit la profondeur.
func TestTaskRepository_DeleteCascadesDeepTree(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	tasks := NewTaskRepository(db)

	newTask(t, tasks, "racine", "p1", nil)
	newTask(t, tasks, "enfant", "p1", ptr("racine"))
	newTask(t, tasks, "petit", "p1", ptr("enfant"))
	newTask(t, tasks, "arriere", "p1", ptr("petit"))
	newTask(t, tasks, "voisine", "p1", nil)

	if err := tasks.Delete("racine"); err != nil {
		t.Fatalf("suppression en cascade : %v", err)
	}

	remaining, err := tasks.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].ID != "voisine" {
		t.Errorf("restantes = %v, seule \"voisine\" attendue", remaining)
	}
}

// UpdateMany est la primitive qu'utilise la cascade : tout ou rien.
func TestTaskRepository_UpdateManyIsAtomic(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	tasks := NewTaskRepository(db)

	a := newTask(t, tasks, "a", "p1", nil)
	b := newTask(t, tasks, "b", "p1", nil)

	a.Completed = true
	b.Completed = true
	fantome := domain.Task{ID: "fantome", ProjectID: "p1", Name: "x", Importance: domain.ImportanceNormale}

	if err := tasks.UpdateMany([]domain.Task{a, b, fantome}); err == nil {
		t.Fatal("une tâche inexistante dans le lot doit faire échouer l'écriture")
	}

	// Aucune des deux tâches valides ne doit avoir été écrite.
	for _, id := range []string{"a", "b"} {
		got, err := tasks.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Completed {
			t.Errorf("%s a été écrite alors que la transaction devait être annulée", id)
		}
	}
}

func TestTaskRepository_UpdateManyEmpty(t *testing.T) {
	tasks := NewTaskRepository(newTestDB(t))
	if err := tasks.UpdateMany(nil); err != nil {
		t.Errorf("un lot vide ne doit pas être une erreur : %v", err)
	}
}

func TestNoteRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	notes := NewNoteRepository(db)

	// La création est datée dans le passé à dessein. L'horloge système de
	// Windows n'avance que par paliers de l'ordre de la milliseconde : deux
	// appels à time.Now() séparés par quelques instructions rendent la même
	// valeur, et comparer à « maintenant » donnerait un test qui échoue selon
	// la vitesse de la machine.
	created := time.Now().UTC().Add(-time.Hour)
	if err := notes.Create(domain.Note{
		ID: "n1", ProjectID: "p1", Title: "Compte rendu",
		Content: "<p>échéance repoussée</p>", CreatedAt: created, UpdatedAt: created,
	}); err != nil {
		t.Fatalf("création : %v", err)
	}

	got, err := notes.Get("n1")
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	if got.Content != "<p>échéance repoussée</p>" {
		t.Errorf("contenu = %q — le HTML de l'éditeur riche doit être stocké tel quel (§3.4)", got.Content)
	}

	got.Title = "Compte rendu v2"
	got.Content = "<p>mis à jour</p>"
	if err := notes.Update(got); err != nil {
		t.Fatalf("mise à jour : %v", err)
	}
	after, err := notes.Get("n1")
	if err != nil {
		t.Fatal(err)
	}
	if after.Title != "Compte rendu v2" || after.Content != "<p>mis à jour</p>" {
		t.Errorf("après mise à jour : %q / %q", after.Title, after.Content)
	}
	if !after.UpdatedAt.After(created) {
		t.Error("updated_at doit être repoussé : c'est l'horodatage affiché par la sauvegarde auto (§2.6)")
	}
	if !after.CreatedAt.Equal(created) {
		t.Errorf("created_at = %v, attendu %v — la mise à jour ne doit pas y toucher", after.CreatedAt, created)
	}
}

func TestNoteRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	newProject(t, NewProjectRepository(db), "p1", "Projet")
	notes := NewNoteRepository(db)

	now := time.Now().UTC()
	if err := notes.Create(domain.Note{ID: "n1", ProjectID: "p1", Title: "x", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := notes.Delete("n1"); err != nil {
		t.Fatalf("suppression : %v", err)
	}
	if _, err := notes.Get("n1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu ErrNotFound", err)
	}
	if err := notes.Delete("n1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("seconde suppression : erreur = %v, attendu ErrNotFound", err)
	}
}

// FTS5 doit être disponible et répondre sur du texte accentué : c'est ce dont
// dépend la recherche du §2.9.
func TestSearchIndex_Fts5AvailableWithAccents(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(
		`INSERT INTO search_index (type, title, content, project_id, entity_id, created_at)
		 VALUES ('note', 'Réunion client', 'échéance repoussée', 'p1', 'n1', '2026-08-21')`,
	)
	if err != nil {
		t.Fatalf("insertion dans l'index : %v", err)
	}
	for _, q := range []string{"réunion", "Réunion", "échéance"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM search_index WHERE search_index MATCH ?`, q).Scan(&n); err != nil {
			t.Fatalf("MATCH %q : %v", q, err)
		}
		if n != 1 {
			t.Errorf("MATCH %q : %d résultats, 1 attendu", q, n)
		}
	}
}
