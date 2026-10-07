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

	// Délai d'attente sur verrou.
	//
	// Sans lui, SQLite rend SQLITE_BUSY **immédiatement** dès qu'une autre
	// connexion tient le verrou d'écriture. L'application a plusieurs lecteurs
	// concurrents — l'interface, et la barre système qui relit les tâches toutes
	// les 30 secondes (§2.10) — pendant que la sauvegarde automatique écrit. Une
	// collision est donc normale, pas exceptionnelle : il faut attendre, pas
	// échouer.
	if _, err := sqlDB.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("délai d'attente sur verrou : %w", err)
	}

	// Journalisation WAL : les lecteurs ne bloquent plus l'écrivain et
	// réciproquement. C'est exactement notre schéma d'accès. Sans effet sur une
	// base en mémoire, qui n'a pas de fichier de journal.
	if path != ":memory:" {
		if _, err := sqlDB.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("passage en WAL : %w", err)
		}
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
	{
		version: 2,
		stmts: `
-- Index sémantique (§2.12, §3.2). Table ordinaire et non virtuelle : la
-- similarité se calcule en Go à la lecture, il n'y a rien à indexer côté
-- SQLite. Le §3.2 écarte explicitement une extension vectorielle, qui
-- réintroduirait une dépendance native à charger.
CREATE TABLE IF NOT EXISTS embeddings (
    entity_id   TEXT PRIMARY KEY,   -- une entité, un vecteur
    type        TEXT NOT NULL CHECK(type IN ('note', 'meeting', 'task')),
    project_id  TEXT NOT NULL,
    vector      BLOB NOT NULL,      -- float32 sérialisés en little-endian
    dimensions  INTEGER NOT NULL,   -- garde-fou : refuser de comparer deux
                                    -- vecteurs de tailles différentes, ce qui
                                    -- arriverait si le modèle changeait
    source_hash TEXT NOT NULL,      -- empreinte du texte vectorisé, pour ne pas
                                    -- recalculer un vecteur inchangé
    updated_at  DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id)
);

CREATE INDEX IF NOT EXISTS idx_embeddings_project ON embeddings(project_id);
`,
	},
	{
		version: 3,
		stmts: `
-- Masquage des projets, notes et réunions obsolètes (§2.1, §2.6, §2.7).
-- Purement visuel : aucune requête transverse (Priorités, barre système,
-- recherche) ne filtre sur cette colonne. DEFAULT 0 rend les lignes
-- existantes visibles sans réécriture.
ALTER TABLE projects ADD COLUMN hidden BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE notes    ADD COLUMN hidden BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE meetings ADD COLUMN hidden BOOLEAN NOT NULL DEFAULT 0;
`,
	},
	{
		version: 4,
		stmts: `
-- Ordre manuel et groupes de notes (v1.2.0). Un groupe n'a pas de position
-- propre : ses notes sont contiguës dans l'ordre unique porté par
-- notes.order_index, et il se trouve là où elles sont.
CREATE TABLE IF NOT EXISTS note_groups (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    name       TEXT NOT NULL,
    created_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_note_groups_project ON note_groups(project_id);

ALTER TABLE notes ADD COLUMN group_id TEXT;
ALTER TABLE notes ADD COLUMN order_index INTEGER NOT NULL DEFAULT 0;

-- Les notes existantes gardent l'ordre affiché jusque-là : la plus récemment
-- modifiée d'abord, l'identifiant départageant les ex æquo.
UPDATE notes SET order_index = (
    SELECT COUNT(*) FROM notes n2
    WHERE n2.project_id = notes.project_id
      AND (n2.updated_at > notes.updated_at
           OR (n2.updated_at = notes.updated_at AND n2.id < notes.id))
);
`,
	},
	{
		version: 5,
		stmts: `
-- Couleur des notes et des réunions (v1.2.0) : une clé de la palette, ou la
-- chaîne vide pour « aucune ». Purement visuel, comme hidden.
ALTER TABLE notes    ADD COLUMN color TEXT NOT NULL DEFAULT '';
ALTER TABLE meetings ADD COLUMN color TEXT NOT NULL DEFAULT '';
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
