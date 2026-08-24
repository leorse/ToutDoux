package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"toutdoux/domain"
)

// NoteRepository implémente ports.NoteRepository sur SQLite.
type NoteRepository struct{ db *DB }

// NewNoteRepository câble le repository sur une base ouverte.
func NewNoteRepository(db *DB) *NoteRepository { return &NoteRepository{db: db} }

const noteColumns = `id, project_id, title, content, created_at, updated_at`

func scanNote(s interface{ Scan(...any) error }) (domain.Note, error) {
	var (
		n                domain.Note
		content          sql.NullString
		created, updated string
	)
	if err := s.Scan(&n.ID, &n.ProjectID, &n.Title, &content, &created, &updated); err != nil {
		return domain.Note{}, err
	}
	n.Content = content.String
	var err error
	if n.CreatedAt, err = parseTime(created); err != nil {
		return domain.Note{}, err
	}
	if n.UpdatedAt, err = parseTime(updated); err != nil {
		return domain.Note{}, err
	}
	return n, nil
}

// ListByProject rend les notes d'un projet, la plus récemment modifiée d'abord.
func (r *NoteRepository) ListByProject(projectID string) ([]domain.Note, error) {
	rows, err := r.db.Query(
		`SELECT `+noteColumns+` FROM notes WHERE project_id = ? ORDER BY updated_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("liste des notes : %w", err)
	}
	defer rows.Close()

	notes := []domain.Note{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// Get rend une note, ou domain.ErrNotFound.
func (r *NoteRepository) Get(id string) (domain.Note, error) {
	row := r.db.QueryRow(`SELECT `+noteColumns+` FROM notes WHERE id = ?`, id)
	n, err := scanNote(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Note{}, domain.ErrNotFound
	}
	return n, err
}

// Create insère une note.
func (r *NoteRepository) Create(n domain.Note) error {
	_, err := r.db.Exec(
		`INSERT INTO notes (`+noteColumns+`) VALUES (?, ?, ?, ?, ?, ?)`,
		n.ID, n.ProjectID, n.Title, n.Content, formatTime(n.CreatedAt), formatTime(n.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("création de la note : %w", err)
	}
	return nil
}

// Update écrit le titre et le contenu d'une note.
//
// Chaque appel repousse updated_at : c'est cette colonne que la sauvegarde
// automatique restitue à l'utilisateur sous la forme « Note sauvegardée à
// 14:32:15 » (§2.6).
func (r *NoteRepository) Update(n domain.Note) error {
	res, err := r.db.Exec(
		`UPDATE notes SET title = ?, content = ?, updated_at = ? WHERE id = ?`,
		n.Title, n.Content, formatTime(nowUTC()), n.ID,
	)
	if err != nil {
		return fmt.Errorf("mise à jour de la note : %w", err)
	}
	return checkAffected(res)
}

// Delete supprime une note.
func (r *NoteRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM search_index WHERE entity_id = ?`, id); err != nil {
		return fmt.Errorf("nettoyage de l'index de recherche : %w", err)
	}
	res, err := tx.Exec(`DELETE FROM notes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("suppression de la note : %w", err)
	}
	if err := checkAffected(res); err != nil {
		return err
	}
	return tx.Commit()
}
