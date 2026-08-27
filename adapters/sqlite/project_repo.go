package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"toutdoux/domain"
)

// ProjectRepository implémente ports.ProjectRepository sur SQLite.
type ProjectRepository struct{ db *DB }

// NewProjectRepository câble le repository sur une base ouverte.
func NewProjectRepository(db *DB) *ProjectRepository { return &ProjectRepository{db: db} }

const projectColumns = `id, name, locked, created_at, updated_at`

func scanProject(s interface{ Scan(...any) error }) (domain.Project, error) {
	var (
		p                domain.Project
		created, updated string
	)
	if err := s.Scan(&p.ID, &p.Name, &p.Locked, &created, &updated); err != nil {
		return domain.Project{}, err
	}
	var err error
	if p.CreatedAt, err = parseTime(created); err != nil {
		return domain.Project{}, err
	}
	if p.UpdatedAt, err = parseTime(updated); err != nil {
		return domain.Project{}, err
	}
	return p, nil
}

// List rend tous les projets. L'ordre d'affichage (Divers en tête) est une
// règle de présentation : il appartient à domain/stats.SortProjects, pas au SQL.
func (r *ProjectRepository) List() ([]domain.Project, error) {
	rows, err := r.db.Query(`SELECT ` + projectColumns + ` FROM projects`)
	if err != nil {
		return nil, fmt.Errorf("liste des projets : %w", err)
	}
	defer rows.Close()

	projects := []domain.Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// Get rend un projet, ou domain.ErrNotFound.
func (r *ProjectRepository) Get(id string) (domain.Project, error) {
	row := r.db.QueryRow(`SELECT `+projectColumns+` FROM projects WHERE id = ?`, id)
	p, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Project{}, domain.ErrNotFound
	}
	return p, err
}

// Create insère un projet.
func (r *ProjectRepository) Create(p domain.Project) error {
	_, err := r.db.Exec(
		`INSERT INTO projects (`+projectColumns+`) VALUES (?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Locked, formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("création du projet : %w", err)
	}
	return nil
}

// Rename renomme un projet.
func (r *ProjectRepository) Rename(id, newName string) error {
	res, err := r.db.Exec(
		`UPDATE projects SET name = ?, updated_at = ? WHERE id = ?`,
		newName, formatTime(nowUTC()), id,
	)
	if err != nil {
		return fmt.Errorf("renommage du projet : %w", err)
	}
	return checkAffected(res)
}

// Delete supprime le projet et tout ce qui en dépend (§2.1).
//
// La cascade est explicite et transactionnelle plutôt que déléguée à
// ON DELETE CASCADE : le schéma de la spec (§3.2) ne le déclare pas, et une
// suppression partielle laisserait des tâches rattachées à un projet disparu.
func (r *ProjectRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM meeting_instances WHERE meeting_id IN (SELECT id FROM meetings WHERE project_id = ?)`, []any{id}},
		{`DELETE FROM meetings WHERE project_id = ?`, []any{id}},
		{`DELETE FROM notes WHERE project_id = ?`, []any{id}},
		{`DELETE FROM tasks WHERE project_id = ?`, []any{id}},
		{`DELETE FROM images WHERE project_id = ?`, []any{id}},
		{`DELETE FROM search_index WHERE project_id = ?`, []any{id}},
		// L'index sémantique porte une clé étrangère sur le projet : sans cette
		// ligne, la suppression échouerait sur une violation de contrainte dès
		// qu'une seule entité du projet aurait été vectorisée (§3.2).
		{`DELETE FROM embeddings WHERE project_id = ?`, []any{id}},
	}
	for _, s := range statements {
		if _, err := tx.Exec(s.query, s.args...); err != nil {
			return fmt.Errorf("cascade de suppression du projet : %w", err)
		}
	}

	res, err := tx.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("suppression du projet : %w", err)
	}
	if err := checkAffected(res); err != nil {
		return err
	}
	return tx.Commit()
}

// ExistsByName teste l'unicité insensible à la casse (§2.1).
//
// La comparaison se fait en Go plutôt qu'avec LOWER() en SQL : le LOWER() de
// SQLite ne connaît que l'ASCII, et laisserait donc coexister « Réunion » et
// « RÉUNION » — ce qui, sur un corpus francophone, est le cas nominal et non
// un cas limite.
func (r *ProjectRepository) ExistsByName(name string, excludeID string) (bool, error) {
	rows, err := r.db.Query(`SELECT id, name FROM projects`)
	if err != nil {
		return false, fmt.Errorf("vérification d'unicité : %w", err)
	}
	defer rows.Close()

	target := strings.ToLower(strings.TrimSpace(name))
	for rows.Next() {
		var id, existing string
		if err := rows.Scan(&id, &existing); err != nil {
			return false, err
		}
		if id == excludeID {
			continue
		}
		if strings.ToLower(strings.TrimSpace(existing)) == target {
			return true, nil
		}
	}
	return false, rows.Err()
}

// checkAffected traduit « aucune ligne touchée » en ErrNotFound.
func checkAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
