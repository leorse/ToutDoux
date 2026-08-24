// Package cascade porte la cohérence des statuts entre parents et enfants (§2.2).
//
// C'est la règle métier la plus délicate de la spécification, et celle qui a le
// plus de surface de régression : dans le prototype React elle n'existait pas
// comme unité, elle était diluée dans trois useCallback entrelacés avec des
// appels setTasks. L'isoler ici est le point du portage qui compte le plus.
//
// Toutes les fonctions sont pures : elles reçoivent l'état, rendent les tâches
// modifiées, et ne touchent ni à la base ni à l'UI.
package cascade

import (
	"toutdoux/domain"
	"toutdoux/domain/tasktree"
)

// index construit une copie indexée des tâches, pour muter sans toucher l'entrée.
func index(tasks []domain.Task) map[string]*domain.Task {
	byID := make(map[string]*domain.Task, len(tasks))
	for _, t := range tasks {
		copied := t
		byID[t.ID] = &copied
	}
	return byID
}

// collect rend les tâches marquées comme modifiées, dans l'ordre d'origine
// pour que le résultat soit déterministe.
func collect(tasks []domain.Task, byID map[string]*domain.Task, changed map[string]bool) []domain.Task {
	out := make([]domain.Task, 0, len(changed))
	for _, t := range tasks {
		if changed[t.ID] {
			out = append(out, *byID[t.ID])
		}
	}
	return out
}

// reactivateFrom réactive la chaîne d'ancêtres à partir de startID inclus,
// en mutant byID et en notant les changements.
//
// La remontée s'arrête au premier ancêtre déjà actif : au-delà, les statuts
// sont cohérents et les toucher effacerait de l'information.
func reactivateFrom(byID map[string]*domain.Task, startID string, changed map[string]bool) {
	cur, ok := byID[startID]
	seen := make(map[string]bool)
	for ok && (cur.Completed || cur.Cancelled) {
		if seen[cur.ID] {
			break // cycle en base : on interrompt plutôt que de boucler
		}
		seen[cur.ID] = true

		cur.Completed = false
		cur.Cancelled = false
		changed[cur.ID] = true

		if cur.ParentID == nil {
			return
		}
		cur, ok = byID[*cur.ParentID]
	}
}

// ReactivateAncestors réactive startID et ses ancêtres tant qu'ils sont
// terminés ou annulés (§2.2).
//
// À appeler quand une sous-tâche est ajoutée sous startID, ou qu'une tâche
// active y est déplacée : la nouvelle venue est active par nature, ses ancêtres
// ne peuvent donc plus être considérés comme finis.
//
// Rend les tâches modifiées ; la tranche est vide si startID était déjà actif.
func ReactivateAncestors(tasks []domain.Task, startID string) []domain.Task {
	byID := index(tasks)
	changed := make(map[string]bool)
	reactivateFrom(byID, startID, changed)
	return collect(tasks, byID, changed)
}

// SetCompleted applique un statut terminé à une tâche et propage (§2.2).
//
// Vers le bas : la tâche et toute sa descendance prennent la valeur. Cocher
// lève aussi cancelled, une tâche ne pouvant être à la fois faite et annulée.
//
// Vers le haut : chaque ancêtre est recalculé depuis ses enfants directs — il
// est terminé si et seulement si tous le sont. Un ancêtre qui ne l'est plus
// perd aussi cancelled, et pas seulement completed : c'est la correction de
// fond de la v2, la v1 ne traitait que completed et laissait des parents
// annulés au-dessus d'enfants actifs.
//
// Rend les tâches modifiées, ou ErrNotFound si taskID n'existe pas.
func SetCompleted(tasks []domain.Task, taskID string, completed bool) ([]domain.Task, error) {
	byID := index(tasks)
	target, ok := byID[taskID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	changed := make(map[string]bool)

	// Descente : la tâche et ses descendants.
	descendants := tasktree.DescendantIDs(tasks, taskID)
	apply := func(t *domain.Task) {
		if t.Completed == completed && !(completed && t.Cancelled) {
			return
		}
		t.Completed = completed
		if completed {
			t.Cancelled = false
		}
		changed[t.ID] = true
	}
	apply(target)
	for id := range descendants {
		apply(byID[id])
	}

	// Remontée : chaque ancêtre se déduit de ses enfants directs.
	children := tasktree.ChildrenMap(tasks)
	cur := target.ParentID
	seen := make(map[string]bool)
	for cur != nil {
		if seen[*cur] {
			break // cycle en base
		}
		seen[*cur] = true

		parent, ok := byID[*cur]
		if !ok {
			break
		}
		kids := children[parent.ID]
		allDone := len(kids) > 0
		for _, k := range kids {
			if !byID[k.ID].Completed {
				allDone = false
				break
			}
		}
		if parent.Completed != allDone {
			parent.Completed = allDone
			changed[parent.ID] = true
		}
		if !allDone && parent.Cancelled {
			parent.Cancelled = false
			changed[parent.ID] = true
		}
		cur = parent.ParentID
	}

	return collect(tasks, byID, changed), nil
}

// SetCancelled applique le statut annulé à une tâche (§2.2).
//
// cancelled est un état distinct de completed, pas un cas particulier :
// annuler lève donc completed.
func SetCancelled(tasks []domain.Task, taskID string, cancelled bool) ([]domain.Task, error) {
	byID := index(tasks)
	target, ok := byID[taskID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	changed := make(map[string]bool)
	if target.Cancelled != cancelled || (cancelled && target.Completed) {
		target.Cancelled = cancelled
		if cancelled {
			target.Completed = false
		}
		changed[target.ID] = true
	}
	return collect(tasks, byID, changed), nil
}

// CheckReparent valide un déplacement avant de l'appliquer (§2.2).
//
// Rend ErrCycle si la cible est la tâche elle-même ou l'un de ses descendants.
// L'échec est explicite : la spec exige un message, jamais un abandon silencieux.
func CheckReparent(tasks []domain.Task, draggedID string, newParentID *string) error {
	if newParentID == nil {
		return nil // passer à la racine ne peut pas créer de cycle
	}
	if draggedID == *newParentID {
		return domain.ErrCycle
	}
	if tasktree.DescendantIDs(tasks, draggedID)[*newParentID] {
		return domain.ErrCycle
	}
	return nil
}

// Reparent déplace une tâche sous un nouveau parent (§2.2).
//
// La tâche déplacée hérite du projet de sa cible, se place en fin de fratrie,
// et déclenche la réactivation des ancêtres — mais seulement si elle est
// elle-même active. Déplacer une tâche déjà terminée ne réactive rien : la
// cohérence du parent reste correcte, et c'est le piège classique de cette règle.
func Reparent(tasks []domain.Task, draggedID string, newParentID *string) ([]domain.Task, error) {
	if err := CheckReparent(tasks, draggedID, newParentID); err != nil {
		return nil, err
	}
	byID := index(tasks)
	dragged, ok := byID[draggedID]
	if !ok {
		return nil, domain.ErrNotFound
	}

	projectID := dragged.ProjectID
	if newParentID != nil {
		parent, ok := byID[*newParentID]
		if !ok {
			return nil, domain.ErrNotFound
		}
		projectID = parent.ProjectID
	}

	changed := make(map[string]bool)
	dragged.ParentID = newParentID
	dragged.ProjectID = projectID
	dragged.OrderIndex = tasktree.NextOrderIndex(tasks, projectID, newParentID)
	changed[dragged.ID] = true

	if dragged.Active() && newParentID != nil {
		reactivateFrom(byID, *newParentID, changed)
	}
	return collect(tasks, byID, changed), nil
}
