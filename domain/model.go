// Package domain porte les entités et les règles métier de Tout Doux.
//
// Il ne dépend d'aucune infrastructure : ni SQLite, ni Wails, ni HTTP. C'est
// l'invariant central de l'architecture (§3.9) — SQLite implémente les contrats
// définis ici, jamais l'inverse.
package domain

import "time"

// Importance d'une tâche (§2.2). Les valeurs correspondent exactement à la
// contrainte CHECK de la colonne `importance` du schéma (§3.2).
type Importance string

const (
	ImportanceBasse    Importance = "Basse"
	ImportanceNormale  Importance = "Normale"
	ImportanceHaute    Importance = "Haute"
	ImportanceCritique Importance = "Critique"
)

// AllImportances liste les importances de la plus forte à la plus faible.
var AllImportances = []Importance{ImportanceCritique, ImportanceHaute, ImportanceNormale, ImportanceBasse}

// Valid indique si l'importance fait partie des valeurs autorisées.
func (i Importance) Valid() bool {
	switch i {
	case ImportanceBasse, ImportanceNormale, ImportanceHaute, ImportanceCritique:
		return true
	}
	return false
}

// Status est l'état d'avancement d'une tâche (§2.4).
//
// Il n'est pas stocké tel quel : la base porte deux booléens `completed` et
// `cancelled` (§3.2). Status est la vue dérivée qu'utilisent les filtres, où
// les trois états sont mutuellement exclusifs.
type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

// Project est un projet (§2.1).
type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Locked    bool      `json:"locked"` // vrai uniquement pour "Transverse / Divers"
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// DiversProjectID est l'identifiant stable du projet "Transverse / Divers"
// créé automatiquement au démarrage (§2.1). Il est fixe et non aléatoire :
// le frontend doit pouvoir le reconnaître sans le chercher par nom.
const DiversProjectID = "project-divers"

// DiversProjectName est le nom affiché du projet verrouillé (§2.1).
const DiversProjectName = "Transverse / Divers"

// Task est une tâche (§2.2). ParentID vaut nil pour une tâche racine.
type Task struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"projectId"`
	ParentID    *string    `json:"parentId"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Importance  Importance `json:"importance"`
	Completed   bool       `json:"completed"`
	Cancelled   bool       `json:"cancelled"`
	DueDate     *time.Time `json:"dueDate"`
	OrderIndex  int        `json:"orderIndex"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// Status dérive l'état exclusif de la tâche depuis ses deux booléens.
//
// `cancelled` prime sur `completed` : une tâche annulée s'affiche barrée et
// grise même si elle porte aussi `completed` (§2.2).
func (t Task) Status() Status {
	switch {
	case t.Cancelled:
		return StatusCancelled
	case t.Completed:
		return StatusCompleted
	default:
		return StatusActive
	}
}

// Active indique que la tâche n'est ni terminée ni annulée.
func (t Task) Active() bool { return !t.Completed && !t.Cancelled }

// Note est une note de projet (§2.6).
type Note struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Meeting est une réunion, rattachée à un projet (§2.7).
type Meeting struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// IndexEntry est une entrée de l'index plein texte (§3.2).
//
// C'est la forme sous laquelle le domaine voit l'index : un type plat, sans
// notion de table ni de FTS5. L'adaptateur traduit vers `search_index`.
type IndexEntry struct {
	Type      SearchType `json:"type"`
	EntityID  string     `json:"entityId"`
	ProjectID string     `json:"projectId"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	CreatedAt *time.Time `json:"createdAt"`
}

// SearchType est la nature d'une entité indexée, qui détermine l'icône du
// résultat (§2.9).
type SearchType string

const (
	SearchTypeTask    SearchType = "task"    // case à cocher + pastille d'importance
	SearchTypeNote    SearchType = "note"    // 📝
	SearchTypeMeeting SearchType = "meeting" // 📞
	SearchTypeProject SearchType = "project" // 📁
)

// MeetingInstance est une occurrence datée d'une réunion (§2.7).
type MeetingInstance struct {
	ID        string    `json:"id"`
	MeetingID string    `json:"meetingId"`
	Notes     string    `json:"notes"`
	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Embedding est le vecteur de sens d'une entité ajoutée à l'index sémantique
// (§2.12, §3.2).
//
// SourceHash est l'empreinte du texte réellement vectorisé : c'est lui qui
// permet de ne rien recalculer quand la sauvegarde automatique réenregistre un
// texte inchangé. Dimensions accompagne le vecteur pour refuser de comparer
// deux vecteurs produits par des modèles différents.
type Embedding struct {
	EntityID   string     `json:"entityId"`
	Type       SearchType `json:"type"`
	ProjectID  string     `json:"projectId"`
	Vector     []float32  `json:"vector"`
	Dimensions int        `json:"dimensions"`
	SourceHash string     `json:"sourceHash"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}
