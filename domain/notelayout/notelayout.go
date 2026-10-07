// Package notelayout porte les règles de disposition de la liste des notes
// d'un projet : un ordre unique, dans lequel les notes d'un même groupe sont
// contiguës. Un groupe n'a pas de position propre, il se trouve là où sont ses
// notes.
package notelayout

import (
	"fmt"

	"toutdoux/domain"
)

// Item est une note à sa place dans la liste. GroupID est vide hors groupe.
type Item struct {
	NoteID  string `json:"noteId"`
	GroupID string `json:"groupId"`
}

// FromNotes rend la disposition de notes déjà rangées dans l'ordre de la liste.
func FromNotes(notes []domain.Note) []Item {
	layout := make([]Item, 0, len(notes))
	for _, n := range notes {
		item := Item{NoteID: n.ID}
		if n.GroupID != nil {
			item.GroupID = *n.GroupID
		}
		layout = append(layout, item)
	}
	return layout
}

// Group rassemble les notes sélectionnées dans le groupe newGroupID, dans
// l'ordre où elles apparaissaient, à la place de la première d'entre elles.
// Les notes qui appartenaient à un autre groupe le quittent.
//
// Si cette place tombe au milieu d'un groupe existant qui garde des notes de
// part et d'autre, le nouveau groupe est posé juste après lui : un groupe ne
// contient pas d'autre groupe.
func Group(layout []Item, selectedIDs []string, newGroupID string) []Item {
	selected := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = true
	}

	var rest, picked []Item
	at := -1
	for _, item := range layout {
		if selected[item.NoteID] {
			if at < 0 {
				at = len(rest)
			}
			picked = append(picked, Item{NoteID: item.NoteID, GroupID: newGroupID})
			continue
		}
		rest = append(rest, item)
	}
	if at < 0 {
		return append([]Item{}, layout...)
	}
	for at > 0 && at < len(rest) && rest[at-1].GroupID != "" && rest[at-1].GroupID == rest[at].GroupID {
		at++
	}

	out := make([]Item, 0, len(layout))
	out = append(out, rest[:at]...)
	out = append(out, picked...)
	return append(out, rest[at:]...)
}

// Dissolve sort toutes les notes du groupe sans les déplacer.
func Dissolve(layout []Item, groupID string) []Item {
	out := make([]Item, len(layout))
	for i, item := range layout {
		if item.GroupID == groupID {
			item.GroupID = ""
		}
		out[i] = item
	}
	return out
}

// Validate vérifie qu'une disposition reçue de l'extérieur est cohérente :
// exactement les notes connues, des groupes connus, et chaque groupe d'un seul
// tenant. Rend une erreur qui enveloppe domain.ErrInvalidLayout.
func Validate(layout []Item, noteIDs []string, groupIDs []string) error {
	notes := make(map[string]bool, len(noteIDs))
	for _, id := range noteIDs {
		notes[id] = true
	}
	groups := make(map[string]bool, len(groupIDs))
	for _, id := range groupIDs {
		groups[id] = true
	}
	if len(layout) != len(notes) {
		return fmt.Errorf("%w : %d notes reçues pour %d attendues", domain.ErrInvalidLayout, len(layout), len(notes))
	}

	seen := make(map[string]bool, len(layout))
	closed := map[string]bool{} // groupes dont la suite de notes est terminée
	previous := ""
	for _, item := range layout {
		if !notes[item.NoteID] || seen[item.NoteID] {
			return fmt.Errorf("%w : note %q inconnue ou en double", domain.ErrInvalidLayout, item.NoteID)
		}
		seen[item.NoteID] = true

		if item.GroupID != previous {
			if previous != "" {
				closed[previous] = true
			}
			if item.GroupID != "" {
				if !groups[item.GroupID] {
					return fmt.Errorf("%w : groupe %q inconnu", domain.ErrInvalidLayout, item.GroupID)
				}
				if closed[item.GroupID] {
					return fmt.Errorf("%w : les notes du groupe %q ne sont pas contiguës", domain.ErrInvalidLayout, item.GroupID)
				}
			}
			previous = item.GroupID
		}
	}
	return nil
}
