package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"toutdoux/domain"
	"toutdoux/domain/tasktree"
)

// TaskRepository implémente ports.TaskRepository sur SQLite.
type TaskRepository struct{ db *DB }

// NewTaskRepository câble le repository sur une base ouverte.
func NewTaskRepository(db *DB) *TaskRepository { return &TaskRepository{db: db} }

const taskColumns = `id, project_id, parent_id, name, description, importance,
	completed, cancelled, due_date, order_index, created_at, updated_at`

func scanTask(s interface{ Scan(...any) error }) (domain.Task, error) {
	var (
		t                domain.Task
		parentID         sql.NullString
		description      sql.NullString
		due              sql.NullString
		created, updated string
	)
	err := s.Scan(&t.ID, &t.ProjectID, &parentID, &t.Name, &description, &t.Importance,
		&t.Completed, &t.Cancelled, &due, &t.OrderIndex, &created, &updated)
	if err != nil {
		return domain.Task{}, err
	}
	if parentID.Valid {
		p := parentID.String
		t.ParentID = &p
	}
	t.Description = description.String
	if t.DueDate, err = scanNullTime(due); err != nil {
		return domain.Task{}, err
	}
	if t.CreatedAt, err = parseTime(created); err != nil {
		return domain.Task{}, err
	}
	if t.UpdatedAt, err = parseTime(updated); err != nil {
		return domain.Task{}, err
	}
	return t, nil
}

func (r *TaskRepository) query(where string, args ...any) ([]domain.Task, error) {
	rows, err := r.db.Query(`SELECT `+taskColumns+` FROM tasks `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("lecture des tâches : %w", err)
	}
	defer rows.Close()

	tasks := []domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// ListByProject rend les tâches d'un projet, triées par order_index.
func (r *TaskRepository) ListByProject(projectID string) ([]domain.Task, error) {
	return r.query(`WHERE project_id = ? ORDER BY order_index`, projectID)
}

// ListAll rend toutes les tâches, pour les vues transverses (§2.8, §2.10).
func (r *TaskRepository) ListAll() ([]domain.Task, error) {
	return r.query(`ORDER BY project_id, order_index`)
}

// Get rend une tâche, ou domain.ErrNotFound.
func (r *TaskRepository) Get(id string) (domain.Task, error) {
	row := r.db.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, domain.ErrNotFound
	}
	return t, err
}

// Create insère une tâche.
func (r *TaskRepository) Create(t domain.Task) error {
	_, err := r.db.Exec(
		`INSERT INTO tasks (`+taskColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, nullString(t.ParentID), t.Name, t.Description, string(t.Importance),
		t.Completed, t.Cancelled, nullTime(t.DueDate), t.OrderIndex,
		formatTime(t.CreatedAt), formatTime(t.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("création de la tâche : %w", err)
	}
	return nil
}

const updateTaskSQL = `UPDATE tasks SET
	project_id = ?, parent_id = ?, name = ?, description = ?, importance = ?,
	completed = ?, cancelled = ?, due_date = ?, order_index = ?, updated_at = ?
	WHERE id = ?`

func updateTaskArgs(t domain.Task) []any {
	return []any{
		t.ProjectID, nullString(t.ParentID), t.Name, t.Description, string(t.Importance),
		t.Completed, t.Cancelled, nullTime(t.DueDate), t.OrderIndex, formatTime(nowUTC()),
		t.ID,
	}
}

// Update écrit une tâche.
func (r *TaskRepository) Update(t domain.Task) error {
	res, err := r.db.Exec(updateTaskSQL, updateTaskArgs(t)...)
	if err != nil {
		return fmt.Errorf("mise à jour de la tâche : %w", err)
	}
	return checkAffected(res)
}

// UpdateMany écrit en une transaction les tâches modifiées par une cascade.
//
// L'atomicité n'est pas un confort ici : une cascade de réactivation écrite à
// moitié produirait exactement l'incohérence parent/enfant que la règle du
// §2.2 existe pour empêcher.
func (r *TaskRepository) UpdateMany(tasks []domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(updateTaskSQL)
	if err != nil {
		return fmt.Errorf("préparation de la mise à jour : %w", err)
	}
	defer stmt.Close()

	for _, t := range tasks {
		res, err := stmt.Exec(updateTaskArgs(t)...)
		if err != nil {
			return fmt.Errorf("mise à jour de la tâche %s : %w", t.ID, err)
		}
		if err := checkAffected(res); err != nil {
			return fmt.Errorf("tâche %s : %w", t.ID, err)
		}
	}
	return tx.Commit()
}

// Delete supprime une tâche et toute sa descendance (§2.2).
//
// Les descendants sont calculés par le domaine plutôt que par un CTE récursif
// SQL : la logique d'arbre a déjà une implémentation testée, en dupliquer une
// seconde en SQL créerait deux définitions à maintenir en accord.
func (r *TaskRepository) Delete(id string) error {
	all, err := r.ListAll()
	if err != nil {
		return err
	}
	ids := append([]string{id}, keys(tasktree.DescendantIDs(all, id))...)

	// Suppression du plus profond vers le plus haut : la clé étrangère
	// parent_id interdit d'effacer un parent encore référencé, et l'ordre de
	// parcours d'une map est délibérément aléatoire en Go — s'y fier ferait
	// échouer la suppression une fois sur deux, de façon indéboguable.
	sort.SliceStable(ids, func(i, j int) bool {
		return len(tasktree.AncestorIDs(all, ids[i])) > len(tasktree.AncestorIDs(all, ids[j]))
	})

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, taskID := range ids {
		if _, err := tx.Exec(`DELETE FROM search_index WHERE entity_id = ?`, taskID); err != nil {
			return fmt.Errorf("nettoyage de l'index de recherche : %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM tasks WHERE id = ?`, taskID); err != nil {
			return fmt.Errorf("suppression de la tâche %s : %w", taskID, err)
		}
	}
	return tx.Commit()
}

// keys rend les clés d'un ensemble sous forme de tranche.
func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// nullString rend une valeur SQL nullable depuis un pointeur de chaîne.
func nullString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
