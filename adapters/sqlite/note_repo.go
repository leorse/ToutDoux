package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"toutdoux/domain"
	"toutdoux/domain/notelayout"
)

// NoteRepository implémente ports.NoteRepository sur SQLite.
type NoteRepository struct{ db *DB }

// NewNoteRepository câble le repository sur une base ouverte.
func NewNoteRepository(db *DB) *NoteRepository { return &NoteRepository{db: db} }

const noteColumns = `id, project_id, title, content, hidden, color, group_id, order_index, created_at, updated_at`

func scanNote(s interface{ Scan(...any) error }) (domain.Note, error) {
	var (
		n                domain.Note
		content, groupID sql.NullString
		created, updated string
	)
	if err := s.Scan(&n.ID, &n.ProjectID, &n.Title, &content, &n.Hidden, &n.Color, &groupID, &n.OrderIndex, &created, &updated); err != nil {
		return domain.Note{}, err
	}
	n.Content = content.String
	if groupID.Valid {
		n.GroupID = &groupID.String
	}
	var err error
	if n.CreatedAt, err = parseTime(created); err != nil {
		return domain.Note{}, err
	}
	if n.UpdatedAt, err = parseTime(updated); err != nil {
		return domain.Note{}, err
	}
	return n, nil
}

// ListByProject rend les notes d'un projet dans l'ordre choisi par l'utilisateur.
func (r *NoteRepository) ListByProject(projectID string) ([]domain.Note, error) {
	rows, err := r.db.Query(
		`SELECT `+noteColumns+` FROM notes WHERE project_id = ? ORDER BY order_index, id`,
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

// Create insère une note en tête de la liste de son projet, hors groupe.
//
// L'indice est pris juste sous le plus petit existant : pas de renumérotation,
// et un indice négatif ne gêne pas jusqu'au prochain SaveLayout.
func (r *NoteRepository) Create(n domain.Note) error {
	_, err := r.db.Exec(
		`INSERT INTO notes (id, project_id, title, content, hidden, created_at, updated_at, order_index)
		 VALUES (?, ?, ?, ?, ?, ?, ?, (SELECT COALESCE(MIN(order_index), 1) - 1 FROM notes WHERE project_id = ?))`,
		n.ID, n.ProjectID, n.Title, n.Content, n.Hidden, formatTime(n.CreatedAt), formatTime(n.UpdatedAt), n.ProjectID,
	)
	if err != nil {
		return fmt.Errorf("création de la note : %w", err)
	}
	return nil
}

// SetHidden masque ou réaffiche une note (§2.6).
//
// N'écrit pas updated_at : ce n'est pas une modification de contenu.
func (r *NoteRepository) SetHidden(id string, hidden bool) error {
	res, err := r.db.Exec(`UPDATE notes SET hidden = ? WHERE id = ?`, hidden, id)
	if err != nil {
		return fmt.Errorf("masquage de la note : %w", err)
	}
	return checkAffected(res)
}

// SetColor pose ou retire la couleur d'une note.
//
// N'écrit pas updated_at : ce n'est pas une modification de contenu.
func (r *NoteRepository) SetColor(id string, color string) error {
	res, err := r.db.Exec(`UPDATE notes SET color = ? WHERE id = ?`, color, id)
	if err != nil {
		return fmt.Errorf("couleur de la note : %w", err)
	}
	return checkAffected(res)
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

// SaveLayout réécrit l'ordre et les groupes des notes d'un projet, puis
// supprime les groupes du projet qui n'ont plus aucune note.
func (r *NoteRepository) SaveLayout(projectID string, layout []notelayout.Item) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, item := range layout {
		var groupID any
		if item.GroupID != "" {
			groupID = item.GroupID
		}
		if _, err := tx.Exec(
			`UPDATE notes SET order_index = ?, group_id = ? WHERE id = ? AND project_id = ?`,
			i, groupID, item.NoteID, projectID,
		); err != nil {
			return fmt.Errorf("écriture de la disposition des notes : %w", err)
		}
	}
	if _, err := tx.Exec(
		`DELETE FROM note_groups WHERE project_id = ?
		 AND id NOT IN (SELECT group_id FROM notes WHERE project_id = ? AND group_id IS NOT NULL)`,
		projectID, projectID,
	); err != nil {
		return fmt.Errorf("nettoyage des groupes vides : %w", err)
	}
	return tx.Commit()
}

// Delete supprime une note, et son groupe si elle en était la dernière.
func (r *NoteRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var groupID sql.NullString
	if err := tx.QueryRow(`SELECT group_id FROM notes WHERE id = ?`, id).Scan(&groupID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("lecture du groupe de la note : %w", err)
	}

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
	if groupID.Valid {
		if _, err := tx.Exec(
			`DELETE FROM note_groups WHERE id = ? AND NOT EXISTS (SELECT 1 FROM notes WHERE group_id = ?)`,
			groupID.String, groupID.String,
		); err != nil {
			return fmt.Errorf("nettoyage du groupe vide : %w", err)
		}
	}
	return tx.Commit()
}
