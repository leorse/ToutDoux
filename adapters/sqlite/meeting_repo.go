package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"toutdoux/domain"
)

// MeetingRepository implémente ports.MeetingRepository sur SQLite.
type MeetingRepository struct{ db *DB }

// NewMeetingRepository câble le repository sur une base ouverte.
func NewMeetingRepository(db *DB) *MeetingRepository { return &MeetingRepository{db: db} }

const meetingColumns = `id, project_id, title, hidden, created_at, updated_at`
const instanceColumns = `id, meeting_id, notes, timestamp, created_at, updated_at`

func scanMeeting(s interface{ Scan(...any) error }) (domain.Meeting, error) {
	var (
		m                domain.Meeting
		created, updated string
	)
	if err := s.Scan(&m.ID, &m.ProjectID, &m.Title, &m.Hidden, &created, &updated); err != nil {
		return domain.Meeting{}, err
	}
	var err error
	if m.CreatedAt, err = parseTime(created); err != nil {
		return domain.Meeting{}, err
	}
	if m.UpdatedAt, err = parseTime(updated); err != nil {
		return domain.Meeting{}, err
	}
	return m, nil
}

func scanInstance(s interface{ Scan(...any) error }) (domain.MeetingInstance, error) {
	var (
		i                           domain.MeetingInstance
		notes                       sql.NullString
		timestamp, created, updated string
	)
	if err := s.Scan(&i.ID, &i.MeetingID, &notes, &timestamp, &created, &updated); err != nil {
		return domain.MeetingInstance{}, err
	}
	i.Notes = notes.String
	var err error
	if i.Timestamp, err = parseTime(timestamp); err != nil {
		return domain.MeetingInstance{}, err
	}
	if i.CreatedAt, err = parseTime(created); err != nil {
		return domain.MeetingInstance{}, err
	}
	if i.UpdatedAt, err = parseTime(updated); err != nil {
		return domain.MeetingInstance{}, err
	}
	return i, nil
}

// ListByProject rend les réunions d'un projet, par titre.
func (r *MeetingRepository) ListByProject(projectID string) ([]domain.Meeting, error) {
	rows, err := r.db.Query(
		`SELECT `+meetingColumns+` FROM meetings WHERE project_id = ? ORDER BY title`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("liste des réunions : %w", err)
	}
	defer rows.Close()

	meetings := []domain.Meeting{}
	for rows.Next() {
		m, err := scanMeeting(rows)
		if err != nil {
			return nil, err
		}
		meetings = append(meetings, m)
	}
	return meetings, rows.Err()
}

// Get rend une réunion, ou domain.ErrNotFound.
func (r *MeetingRepository) Get(id string) (domain.Meeting, error) {
	row := r.db.QueryRow(`SELECT `+meetingColumns+` FROM meetings WHERE id = ?`, id)
	m, err := scanMeeting(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Meeting{}, domain.ErrNotFound
	}
	return m, err
}

// Create insère une réunion.
func (r *MeetingRepository) Create(m domain.Meeting) error {
	_, err := r.db.Exec(
		`INSERT INTO meetings (`+meetingColumns+`) VALUES (?, ?, ?, ?, ?, ?)`,
		m.ID, m.ProjectID, m.Title, m.Hidden, formatTime(m.CreatedAt), formatTime(m.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("création de la réunion : %w", err)
	}
	return nil
}

// Rename renomme une réunion.
func (r *MeetingRepository) Rename(id, newTitle string) error {
	res, err := r.db.Exec(
		`UPDATE meetings SET title = ?, updated_at = ? WHERE id = ?`,
		newTitle, formatTime(nowUTC()), id,
	)
	if err != nil {
		return fmt.Errorf("renommage de la réunion : %w", err)
	}
	return checkAffected(res)
}

// SetHidden masque ou réaffiche une réunion (§2.7).
//
// N'écrit pas updated_at : ce n'est pas une modification de contenu.
func (r *MeetingRepository) SetHidden(id string, hidden bool) error {
	res, err := r.db.Exec(`UPDATE meetings SET hidden = ? WHERE id = ?`, hidden, id)
	if err != nil {
		return fmt.Errorf("masquage de la réunion : %w", err)
	}
	return checkAffected(res)
}

// Delete supprime une réunion et ses instances (§2.7).
func (r *MeetingRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Les instances partent avant leur réunion : la clé étrangère meeting_id
	// interdit l'inverse.
	if _, err := tx.Exec(
		`DELETE FROM search_index WHERE entity_id IN (SELECT id FROM meeting_instances WHERE meeting_id = ?)`, id,
	); err != nil {
		return fmt.Errorf("nettoyage de l'index : %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM meeting_instances WHERE meeting_id = ?`, id); err != nil {
		return fmt.Errorf("suppression des instances : %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM search_index WHERE entity_id = ?`, id); err != nil {
		return fmt.Errorf("nettoyage de l'index : %w", err)
	}
	res, err := tx.Exec(`DELETE FROM meetings WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("suppression de la réunion : %w", err)
	}
	if err := checkAffected(res); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MeetingRepository) queryInstances(where string, args ...any) ([]domain.MeetingInstance, error) {
	rows, err := r.db.Query(`SELECT `+instanceColumns+` FROM meeting_instances `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("lecture des instances : %w", err)
	}
	defer rows.Close()

	out := []domain.MeetingInstance{}
	for rows.Next() {
		i, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// ListInstances rend les instances d'une réunion, la plus récente d'abord (§2.7).
func (r *MeetingRepository) ListInstances(meetingID string) ([]domain.MeetingInstance, error) {
	return r.queryInstances(`WHERE meeting_id = ? ORDER BY timestamp DESC`, meetingID)
}

// ListInstancesByProject rend toutes les instances d'un projet. Sert à la
// recherche, qui indexe les notes d'instance (§2.9).
func (r *MeetingRepository) ListInstancesByProject(projectID string) ([]domain.MeetingInstance, error) {
	return r.queryInstances(
		`WHERE meeting_id IN (SELECT id FROM meetings WHERE project_id = ?) ORDER BY timestamp DESC`,
		projectID,
	)
}

// GetInstance rend une instance, ou domain.ErrNotFound.
func (r *MeetingRepository) GetInstance(id string) (domain.MeetingInstance, error) {
	row := r.db.QueryRow(`SELECT `+instanceColumns+` FROM meeting_instances WHERE id = ?`, id)
	i, err := scanInstance(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.MeetingInstance{}, domain.ErrNotFound
	}
	return i, err
}

// CreateInstance insère une instance.
func (r *MeetingRepository) CreateInstance(i domain.MeetingInstance) error {
	_, err := r.db.Exec(
		`INSERT INTO meeting_instances (`+instanceColumns+`) VALUES (?, ?, ?, ?, ?, ?)`,
		i.ID, i.MeetingID, i.Notes, formatTime(i.Timestamp),
		formatTime(i.CreatedAt), formatTime(i.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("création de l'instance : %w", err)
	}
	return nil
}

// UpdateInstanceNotes écrit les notes d'une instance.
//
// Le timestamp n'est jamais modifié : il date la tenue de la réunion, pas la
// dernière frappe dans le compte rendu (§2.7).
func (r *MeetingRepository) UpdateInstanceNotes(id, notes string) error {
	res, err := r.db.Exec(
		`UPDATE meeting_instances SET notes = ?, updated_at = ? WHERE id = ?`,
		notes, formatTime(nowUTC()), id,
	)
	if err != nil {
		return fmt.Errorf("mise à jour de l'instance : %w", err)
	}
	return checkAffected(res)
}

// DeleteInstance supprime une instance.
func (r *MeetingRepository) DeleteInstance(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM search_index WHERE entity_id = ?`, id); err != nil {
		return fmt.Errorf("nettoyage de l'index : %w", err)
	}
	res, err := tx.Exec(`DELETE FROM meeting_instances WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("suppression de l'instance : %w", err)
	}
	if err := checkAffected(res); err != nil {
		return err
	}
	return tx.Commit()
}
