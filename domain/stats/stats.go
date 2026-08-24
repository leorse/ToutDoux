// Package stats porte les agrégations transverses : compteurs de la sidebar
// (§2.1) et vue Priorités (§2.8).
//
// Le même calcul sert au menu de la barre système (§2.10), que la spec dit
// explicitement réutiliser plutôt que de dupliquer.
package stats

import (
	"sort"
	"strings"
	"time"

	"toutdoux/domain"
	"toutdoux/domain/duedate"
)

// DueIcon est l'icône d'échéance affichée à côté d'un projet (§2.1).
type DueIcon string

const (
	DueIconNone     DueIcon = ""
	DueIconUrgent   DueIcon = "⏰" // urgente ou en retard
	DueIconUpcoming DueIcon = "🕐" // à venir, non urgente
)

// SidebarStats résume un projet pour son affichage en sidebar (§2.1).
type SidebarStats struct {
	// ActiveCount ne compte que les tâches actives : la v1 affichait le triplet
	// Total/Complétées/Actives, jugé inutilement verbeux (§2.1).
	ActiveCount   int     `json:"activeCount"`
	HasCritical   bool    `json:"hasCritical"`
	HasHigh       bool    `json:"hasHigh"`
	DueIcon       DueIcon `json:"dueIcon"`
	NotesCount    int     `json:"notesCount"`
	MeetingsCount int     `json:"meetingsCount"`
}

// Sidebar calcule les compteurs et le code couleur d'un projet (§2.1).
//
// HasHigh n'est vrai que si HasCritical est faux : le rouge prime sur l'orange,
// un projet ne porte qu'une couleur.
func Sidebar(tasks []domain.Task, notes []domain.Note, meetings []domain.Meeting, projectID string, now time.Time) SidebarStats {
	var s SidebarStats
	for _, t := range tasks {
		if t.ProjectID != projectID || !t.Active() {
			continue
		}
		s.ActiveCount++
		switch t.Importance {
		case domain.ImportanceCritique:
			s.HasCritical = true
		case domain.ImportanceHaute:
			s.HasHigh = true
		}
		// L'icône urgente l'emporte définitivement ; l'icône « à venir » ne se
		// pose que si rien de plus grave n'a encore été rencontré.
		if info := duedate.Format(t.DueDate, now); info != nil && s.DueIcon != DueIconUrgent {
			if info.Urgent || info.Overdue {
				s.DueIcon = DueIconUrgent
			} else if s.DueIcon == DueIconNone {
				s.DueIcon = DueIconUpcoming
			}
		}
	}
	if s.HasCritical {
		s.HasHigh = false
	}
	for _, n := range notes {
		if n.ProjectID == projectID {
			s.NotesCount++
		}
	}
	for _, m := range meetings {
		if m.ProjectID == projectID {
			s.MeetingsCount++
		}
	}
	return s
}

// PriorityTasks est le contenu de la vue Priorités (§2.8), transverse à tous
// les projets. C'est aussi la source du menu de la barre système (§2.10).
type PriorityTasks struct {
	CriticalOrHigh []domain.Task `json:"criticalOrHigh"`
	DueSoon        []domain.Task `json:"dueSoon"`
}

// byUrgency trie par échéance croissante, les tâches sans échéance en dernier.
func byUrgency(tasks []domain.Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i].DueDate, tasks[j].DueDate
		switch {
		case a != nil && b != nil:
			return a.Before(*b)
		case a != nil:
			return true
		case b != nil:
			return false
		default:
			return false
		}
	})
}

// Priorities rend les deux sections de la vue Priorités (§2.8).
//
// Seules les tâches actives sont retenues : une tâche terminée n'est plus une
// priorité, quelle que soit son importance.
func Priorities(tasks []domain.Task) PriorityTasks {
	var p PriorityTasks
	for _, t := range tasks {
		if !t.Active() {
			continue
		}
		if t.Importance == domain.ImportanceCritique || t.Importance == domain.ImportanceHaute {
			p.CriticalOrHigh = append(p.CriticalOrHigh, t)
		}
		if t.DueDate != nil {
			p.DueSoon = append(p.DueSoon, t)
		}
	}
	byUrgency(p.CriticalOrHigh)
	byUrgency(p.DueSoon)
	return p
}

// TopPriority rend les n tâches les plus urgentes, pour le menu de la barre
// système (§2.10, « Top 5 tâches prioritaires »).
func TopPriority(tasks []domain.Task, n int) []domain.Task {
	due := Priorities(tasks).DueSoon
	if len(due) > n {
		return due[:n]
	}
	return due
}

// SortProjects trie les projets pour la sidebar (§2.1).
//
// « Transverse / Divers » reste en première position quel que soit son ordre de
// création ; les autres suivent par nom, insensible à la casse, pour que la
// liste reste stable d'une session à l'autre.
func SortProjects(projects []domain.Project) []domain.Project {
	out := make([]domain.Project, len(projects))
	copy(out, projects)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Locked != out[j].Locked {
			return out[i].Locked
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}
