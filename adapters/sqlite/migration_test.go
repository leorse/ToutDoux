package sqlite

import "testing"

// La migration 3 ajoute la colonne `hidden` à trois tables déjà en production
// (§2.1, §2.6, §2.7) : elle doit s'appliquer à une base qui ne connaît encore
// que les migrations 1 et 2, sans perdre les lignes déjà écrites.
func TestMigration3_UpgradesFromVersion2(t *testing.T) {
	original := migrations
	migrations = original[:2] // simule une base qui ne connaît que les migrations 1 et 2
	db := newTestDB(t)
	migrations = original

	// Insertion directe : avant la migration 3, la colonne `hidden` n'existe pas
	// encore, donc ProjectRepository.Create (qui la référence) ne peut pas servir ici.
	now := formatTime(nowUTC())
	if _, err := db.Exec(
		`INSERT INTO projects (id, name, locked, created_at, updated_at) VALUES (?, ?, 0, ?, ?)`,
		"p1", "Avant la migration 3", now, now,
	); err != nil {
		t.Fatalf("insertion avant migration : %v", err)
	}

	if err := db.migrate(); err != nil {
		t.Fatalf("migration vers la version 3 : %v", err)
	}

	relu, err := NewProjectRepository(db).Get("p1")
	if err != nil {
		t.Fatalf("relecture après migration : %v", err)
	}
	if relu.Hidden {
		t.Error("un projet créé avant la migration 3 doit rester visible")
	}

	// Rejouer la migration ne doit rien casser (§ note d'implémentation :
	// « on ajoute des entrées, on ne modifie jamais celles déjà livrées »).
	if err := db.migrate(); err != nil {
		t.Fatalf("seconde migration : %v", err)
	}
}
