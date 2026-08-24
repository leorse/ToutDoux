// Package tasktree porte la structure hiérarchique des tâches, les filtres
// (§2.4) et le dépliement par défaut (§2.2).
package tasktree

import (
	"sort"

	"toutdoux/domain"
)

// RootKey est la clé sous laquelle ChildrenMap range les tâches racines,
// celles dont ParentID vaut nil.
const RootKey = ""

// parentKey rend la clé de regroupement d'une tâche.
func parentKey(t domain.Task) string {
	if t.ParentID == nil {
		return RootKey
	}
	return *t.ParentID
}

// ChildrenMap indexe les tâches par parent, chaque fratrie triée par OrderIndex.
//
// Le tri est ce qui rend l'affichage déterministe : `order_index` doit donc
// être peuplé dès l'insertion (§3.2), sinon l'ordre des frères devient arbitraire.
func ChildrenMap(tasks []domain.Task) map[string][]domain.Task {
	m := make(map[string][]domain.Task)
	for _, t := range tasks {
		k := parentKey(t)
		m[k] = append(m[k], t)
	}
	for k := range m {
		kids := m[k]
		sort.SliceStable(kids, func(i, j int) bool { return kids[i].OrderIndex < kids[j].OrderIndex })
	}
	return m
}

// Roots rend les tâches racines, triées par OrderIndex.
func Roots(tasks []domain.Task) []domain.Task { return ChildrenMap(tasks)[RootKey] }

// DescendantIDs rend l'ensemble des descendants de rootID, à toute profondeur.
//
// rootID lui-même n'en fait pas partie.
func DescendantIDs(tasks []domain.Task, rootID string) map[string]bool {
	children := ChildrenMap(tasks)
	ids := make(map[string]bool)
	stack := []string{rootID}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, kid := range children[cur] {
			if ids[kid.ID] {
				continue // garde-fou : un cycle en base ne doit pas boucler à l'infini
			}
			ids[kid.ID] = true
			stack = append(stack, kid.ID)
		}
	}
	return ids
}

// Filters est l'état des filtres de la barre d'outils (§2.4).
//
// Status et Importance sont des multi-sélections : une tâche passe si son
// statut ET son importance sont tous deux retenus.
type Filters struct {
	Status     map[domain.Status]bool     `json:"status"`
	Importance map[domain.Importance]bool `json:"importance"`
	DueOnly    bool                       `json:"dueOnly"`
}

// DefaultFilters rend l'état initial : seules les tâches actives sont
// affichées, toutes importances confondues.
func DefaultFilters() Filters {
	imp := make(map[domain.Importance]bool, len(domain.AllImportances))
	for _, i := range domain.AllImportances {
		imp[i] = true
	}
	return Filters{
		Status:     map[domain.Status]bool{domain.StatusActive: true},
		Importance: imp,
	}
}

// Matches indique si la tâche satisfait elle-même les filtres, sans considérer
// sa descendance.
func Matches(t domain.Task, f Filters) bool {
	if !f.Status[t.Status()] {
		return false
	}
	if !f.Importance[t.Importance] {
		return false
	}
	if f.DueOnly && t.DueDate == nil {
		return false
	}
	return true
}

// ComputeVisibility rend l'ensemble des tâches à afficher.
//
// Une tâche qui ne satisfait pas les filtres reste visible si l'un de ses
// descendants les satisfait (§2.4) : sans cette règle, filtrer masquerait le
// chemin menant aux tâches retenues, et l'arbre paraîtrait vide.
func ComputeVisibility(tasks []domain.Task, f Filters) map[string]bool {
	children := ChildrenMap(tasks)
	visible := make(map[string]bool)

	var visit func(t domain.Task) bool
	visit = func(t domain.Task) bool {
		anyChildVisible := false
		for _, kid := range children[t.ID] {
			if visit(kid) {
				anyChildVisible = true
			}
		}
		if Matches(t, f) || anyChildVisible {
			visible[t.ID] = true
			return true
		}
		return false
	}

	for _, root := range children[RootKey] {
		visit(root)
	}
	return visible
}

// DefaultExpanded rend les tâches d'un projet à déplier à l'ouverture (§2.2).
//
// Le critère porte sur la sous-arborescence, pas sur la tâche seule : un parent
// terminé dont un enfant reste actif est déplié, sinon cet enfant serait caché.
//
// Ce calcul n'est fait qu'au chargement et au changement de projet, jamais en
// continu (§2.2) — replier une branche sous les yeux de l'utilisateur pendant
// qu'il travaille serait pire que de laisser l'arbre trop ouvert.
func DefaultExpanded(tasks []domain.Task, projectID string) map[string]bool {
	var scoped []domain.Task
	for _, t := range tasks {
		if t.ProjectID == projectID {
			scoped = append(scoped, t)
		}
	}
	children := ChildrenMap(scoped)
	ids := make(map[string]bool)

	var subtreeHasActive func(t domain.Task) bool
	subtreeHasActive = func(t domain.Task) bool {
		childActive := false
		for _, kid := range children[t.ID] {
			if subtreeHasActive(kid) {
				childActive = true
			}
		}
		if t.Active() || childActive {
			ids[t.ID] = true
			return true
		}
		return false
	}

	for _, root := range children[RootKey] {
		subtreeHasActive(root)
	}
	return ids
}

// AncestorIDs rend la chaîne des ancêtres d'une tâche, du parent direct vers
// la racine. Sert à déplier le chemin menant à une tâche ouverte depuis la vue
// Priorités ou la Recherche (§2.8, §2.9).
func AncestorIDs(tasks []domain.Task, taskID string) []string {
	byID := make(map[string]domain.Task, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
	}
	var out []string
	seen := map[string]bool{taskID: true}
	cur, ok := byID[taskID]
	for ok && cur.ParentID != nil {
		if seen[*cur.ParentID] {
			break // cycle en base : on s'arrête plutôt que de boucler
		}
		seen[*cur.ParentID] = true
		out = append(out, *cur.ParentID)
		cur, ok = byID[*cur.ParentID]
	}
	return out
}

// Direction est un déplacement au sein d'une fratrie (§2.2).
type Direction string

const (
	DirectionUp     Direction = "up"     // « Monter »
	DirectionDown   Direction = "down"   // « Descendre »
	DirectionTop    Direction = "top"    // « Envoyer au début »
	DirectionBottom Direction = "bottom" // « Envoyer à la fin »
)

// Move réordonne une tâche parmi ses frères (§2.2).
//
// Le déplacement ne change ni le parent ni le projet : seul l'ordre au sein de
// la fratrie bouge. Tous les frères sont renumérotés de 0 à n-1, ce qui évite
// que les index dérivent après plusieurs déplacements.
//
// Rend les tâches dont l'OrderIndex a changé — vide si le mouvement était sans
// effet, une tâche déjà en tête qu'on demande de monter par exemple.
func Move(tasks []domain.Task, taskID string, dir Direction) []domain.Task {
	var target *domain.Task
	for i := range tasks {
		if tasks[i].ID == taskID {
			target = &tasks[i]
			break
		}
	}
	if target == nil {
		return nil
	}

	siblings := ChildrenMap(tasks)[parentKey(*target)]
	// ChildrenMap regroupe par parent seul ; deux projets peuvent avoir des
	// racines distinctes qui partagent la clé racine. On restreint au projet.
	filtered := siblings[:0:0]
	for _, s := range siblings {
		if s.ProjectID == target.ProjectID {
			filtered = append(filtered, s)
		}
	}
	siblings = filtered

	from := -1
	for i, s := range siblings {
		if s.ID == taskID {
			from = i
			break
		}
	}
	if from == -1 {
		return nil
	}

	to := from
	switch dir {
	case DirectionUp:
		to = max(0, from-1)
	case DirectionDown:
		to = min(len(siblings)-1, from+1)
	case DirectionTop:
		to = 0
	case DirectionBottom:
		to = len(siblings) - 1
	default:
		return nil
	}
	if to == from {
		return nil
	}

	moved := siblings[from]
	reordered := append(siblings[:from:from], siblings[from+1:]...)
	reordered = append(reordered[:to], append([]domain.Task{moved}, reordered[to:]...)...)

	changed := []domain.Task{}
	for i := range reordered {
		if reordered[i].OrderIndex != i {
			reordered[i].OrderIndex = i
			changed = append(changed, reordered[i])
		}
	}
	return changed
}

// NextOrderIndex rend l'OrderIndex à donner à une nouvelle tâche pour qu'elle
// se place en fin de fratrie (§3.2).
func NextOrderIndex(tasks []domain.Task, projectID string, parentID *string) int {
	max := -1
	key := RootKey
	if parentID != nil {
		key = *parentID
	}
	for _, t := range tasks {
		if t.ProjectID != projectID || parentKey(t) != key {
			continue
		}
		if t.OrderIndex > max {
			max = t.OrderIndex
		}
	}
	return max + 1
}
