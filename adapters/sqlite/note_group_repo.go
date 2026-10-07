package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"toutdoux/domain"
)

// NoteGroupRepository implémente ports.NoteGroupRepository sur SQLite.
type NoteGroupRepository struct{ db *DB }

// NewNoteGroupRepository câble le repository sur une base ouverte.
func NewNoteGroupRepository(db *DB) *NoteGroupRepository { return &NoteGroupRepository{db: db} }

const noteGroupColumns = `id, project_id, name, created_at`

func scanNoteGroup(s interface{ Scan(...any) error }) (domain.NoteGroup, error) {
	var (
		g       domain.NoteGroup
		created string
	)
	if err := s.Scan(&g.ID, &g.ProjectID, &g.Name, &created); err != nil {
		return domain.NoteGroup{}, err
	}
	var err error
	if g.CreatedAt, err = parseTime(created); err != nil {
		return domain.NoteGroup{}, err
	}
	return g, nil
}

// ListByProject rend les groupes d'un projet. Leur ordre d'affichage ne vient
// pas d'ici : il découle de la place de leurs notes.
func (r *NoteGroupRepository) ListByProject(projectID string) ([]domain.NoteGroup, error) {
	rows, err := r.db.Query(
		`SELECT `+noteGroupColumns+` FROM note_groups WHERE project_id = ? ORDER BY created_at, id`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("liste des groupes de notes : %w", err)
	}
	defer rows.Close()

	groups := []domain.NoteGroup{}
	for rows.Next() {
		g, err := scanNoteGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// Get rend un groupe, ou domain.ErrNotFound.
func (r *NoteGroupRepository) Get(id string) (domain.NoteGroup, error) {
	row := r.db.QueryRow(`SELECT `+noteGroupColumns+` FROM note_groups WHERE id = ?`, id)
	g, err := scanNoteGroup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NoteGroup{}, domain.ErrNotFound
	}
	return g, err
}

// Create insère un groupe, encore sans note.
func (r *NoteGroupRepository) Create(g domain.NoteGroup) error {
	_, err := r.db.Exec(
		`INSERT INTO note_groups (`+noteGroupColumns+`) VALUES (?, ?, ?, ?)`,
		g.ID, g.ProjectID, g.Name, formatTime(g.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("création du groupe de notes : %w", err)
	}
	return nil
}

// Rename change le nom d'un groupe.
func (r *NoteGroupRepository) Rename(id, newName string) error {
	res, err := r.db.Exec(`UPDATE note_groups SET name = ? WHERE id = ?`, newName, id)
	if err != nil {
		return fmt.Errorf("renommage du groupe de notes : %w", err)
	}
	return checkAffected(res)
}

// Delete supprime un groupe et en détache les notes, qui restent à leur place.
func (r *NoteGroupRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE notes SET group_id = NULL WHERE group_id = ?`, id); err != nil {
		return fmt.Errorf("détachement des notes du groupe : %w", err)
	}
	res, err := tx.Exec(`DELETE FROM note_groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("suppression du groupe de notes : %w", err)
	}
	if err := checkAffected(res); err != nil {
		return err
	}
	return tx.Commit()
}
