// Package sqlite implémente les repositories sur SQLite (§3.2).
//
// Le driver est modernc.org/sqlite, une transposition de SQLite en Go pur :
// aucun CGO, donc aucun compilateur C requis pour construire l'application.
// C'est ce qui rend la chaîne de build indépendante de MSYS2.
package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"toutdoux/domain"
)

// DB encapsule la connexion et les migrations.
type DB struct {
	*sql.DB
}

// Open ouvre la base, applique les migrations et garantit la présence du projet
// « Transverse / Divers » (§2.1).
//
// path vaut ":memory:" pour les tests d'intégration : la spec demande une vraie
// SQLite en mémoire plutôt qu'un mock, pour vérifier le round-trip réel (§3.10).
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("ouverture de la base : %w", err)
	}

	// SQLite ne vérifie pas les clés étrangères par défaut ; sans ce PRAGMA les
	// contraintes du schéma seraient purement décoratives.
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("activation des clés étrangères : %w", err)
	}

	// Une base ":memory:" est propre à chaque connexion : plusieurs connexions
	// verraient chacune une base vide. On force donc le pool à une connexion.
	if path == ":memory:" {
		sqlDB.SetMaxOpenConns(1)
	}

	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	if err := db.ensureDiversProject(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// migration est une étape de schéma, appliquée une fois et jamais rejouée.
type migration struct {
	version int
	stmts   string
}

// migrations liste le schéma dans l'ordre. On ajoute des entrées, on ne modifie
// jamais celles déjà livrées : une migration publiée a déjà tourné ailleurs.
var migrations = []migration{
	{
		version: 1,
		stmts: `
CREATE TABLE IF NOT EXISTS projects (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    locked     BOOLEAN DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS tasks (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL,
    parent_id   TEXT,
    name        TEXT NOT NULL,
    description TEXT,
    importance  TEXT CHECK(importance IN ('Basse', 'Normale', 'Haute', 'Critique')),
    completed   BOOLEAN DEFAULT 0,
    cancelled   BOOLEAN DEFAULT 0,
    due_date    DATETIME,
    order_index INTEGER,
    created_at  DATETIME,
    updated_at  DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id),
    FOREIGN KEY(parent_id)  REFERENCES tasks(id)
);

CREATE TABLE IF NOT EXISTS notes (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    title      TEXT NOT NULL,
    content    TEXT,
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS meetings (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    title      TEXT NOT NULL,
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS meeting_instances (
    id         TEXT PRIMARY KEY,
    meeting_id TEXT NOT NULL,
    notes      TEXT,
    timestamp  DATETIME,
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY(meeting_id) REFERENCES meetings(id)
);

CREATE TABLE IF NOT EXISTS images (
    id         TEXT PRIMARY KEY,
    project_id TEXT,
    owner_type TEXT,
    owner_id   TEXT,
    size       INT,
    format     TEXT,
    created_at DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT
);

-- Index de recherche plein texte (§3.2). FTS5 est compilé dans le driver,
-- rien à installer côté système.
CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
    type, title, content, project_id, entity_id, created_at
);

-- Les tâches sont lues projet par projet et rangées par order_index :
-- sans cet index, chaque affichage d'arbre impose un balayage complet.
CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id, order_index);
CREATE INDEX IF NOT EXISTS idx_tasks_parent  ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_notes_project ON notes(project_id);
CREATE INDEX IF NOT EXISTS idx_meetings_project ON meetings(project_id);
CREATE INDEX IF NOT EXISTS idx_instances_meeting ON meeting_instances(meeting_id, timestamp DESC);
`,
	},
}

// migrate applique les migrations non encore jouées.
func (db *DB) migrate() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY);`); err != nil {
		return fmt.Errorf("table des migrations : %w", err)
	}
	for _, m := range migrations {
		var done int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, m.version).Scan(&done); err != nil {
			return fmt.Errorf("lecture des migrations : %w", err)
		}
		if done > 0 {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(m.stmts); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d : %w", m.version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("marquage de la migration %d : %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// ensureDiversProject crée le projet verrouillé s'il n'existe pas (§2.1).
//
// Son identifiant est fixe : le frontend doit pouvoir le reconnaître sans le
// chercher par nom, et l'app y bascule quand le projet actif est supprimé.
func (db *DB) ensureDiversProject() error {
	now := formatTime(time.Now())
	_, err := db.Exec(
		`INSERT OR IGNORE INTO projects (id, name, locked, created_at, updated_at)
		 VALUES (?, ?, 1, ?, ?)`,
		domain.DiversProjectID, domain.DiversProjectName, now, now,
	)
	if err != nil {
		return fmt.Errorf("création du projet Divers : %w", err)
	}
	return nil
}

// nowUTC rend l'instant courant pour les colonnes updated_at.
//
// Ces colonnes sont de la métadonnée de persistance, pas une règle métier :
// elles n'entrent dans aucun calcul du domaine, qui reçoit son heure du port
// Clock. Les faire passer par ce port n'apporterait rien de testable.
func nowUTC() time.Time { return time.Now().UTC() }

// formatTime sérialise un instant en RFC3339 avec nanosecondes.
//
// SQLite n'a pas de type date : le format textuel est choisi ici pour être
// trié correctement par une comparaison de chaînes, ce que ferait un ORDER BY.
func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// parseTime relit un instant écrit par formatTime.
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, s)
}

// nullTime rend une valeur SQL nullable depuis un instant optionnel.
func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

// scanNullTime relit un instant optionnel.
func scanNullTime(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	t, err := parseTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
