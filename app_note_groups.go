package main

import (
	"strings"

	"github.com/google/uuid"

	"toutdoux/domain"
	"toutdoux/domain/notelayout"
)

/* ---------------- Groupes et ordre des notes (v1.2.0) ---------------- */

// GetNoteGroups rend les groupes de notes d'un projet. Leur place dans la
// liste découle de celle de leurs notes (GetNotes).
func (a *App) GetNoteGroups(projectID string) ([]domain.NoteGroup, error) {
	return a.noteGroups.ListByProject(projectID)
}

// GroupNotes crée un groupe et y rassemble les notes données, à la place de
// la première d'entre elles dans la liste.
func (a *App) GroupNotes(projectID, name string, noteIDs []string) (domain.NoteGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.NoteGroup{}, domain.ErrEmptyName
	}
	notes, err := a.notes.ListByProject(projectID)
	if err != nil {
		return domain.NoteGroup{}, err
	}
	// Les identifiants étrangers au projet sont ignorés ; s'il n'en reste
	// aucun, il n'y a rien à grouper.
	demandees := make(map[string]bool, len(noteIDs))
	for _, id := range noteIDs {
		demandees[id] = true
	}
	trouvee := false
	for _, n := range notes {
		if demandees[n.ID] {
			trouvee = true
			break
		}
	}
	if !trouvee {
		return domain.NoteGroup{}, domain.ErrNotFound
	}

	g := domain.NoteGroup{ID: uuid.NewString(), ProjectID: projectID, Name: name, CreatedAt: a.clock.Now()}
	if err := a.noteGroups.Create(g); err != nil {
		return domain.NoteGroup{}, err
	}
	layout := notelayout.Group(notelayout.FromNotes(notes), noteIDs, g.ID)
	if err := a.notes.SaveLayout(projectID, layout); err != nil {
		// Ne pas laisser derrière soi un groupe sans note.
		a.noteGroups.Delete(g.ID)
		return domain.NoteGroup{}, err
	}
	return g, nil
}

// RenameNoteGroup change le nom d'un groupe, sans toucher à ses notes.
func (a *App) RenameNoteGroup(groupID, name string) (domain.NoteGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.NoteGroup{}, domain.ErrEmptyName
	}
	if err := a.noteGroups.Rename(groupID, name); err != nil {
		return domain.NoteGroup{}, err
	}
	return a.noteGroups.Get(groupID)
}

// DissolveNoteGroup supprime un groupe : ses notes restent où elles sont et
// redeviennent des notes hors groupe.
func (a *App) DissolveNoteGroup(groupID string) error {
	g, err := a.noteGroups.Get(groupID)
	if err != nil {
		return err
	}
	notes, err := a.notes.ListByProject(g.ProjectID)
	if err != nil {
		return err
	}
	// SaveLayout emporte le groupe, désormais sans note.
	return a.notes.SaveLayout(g.ProjectID, notelayout.Dissolve(notelayout.FromNotes(notes), groupID))
}

// SetNotesLayout écrit l'ordre des notes d'un projet et leur répartition dans
// les groupes. C'est la cible de tous les glisser-déposer de la liste : le
// frontend calcule la disposition complète, on la vérifie et on l'enregistre.
func (a *App) SetNotesLayout(projectID string, layout []notelayout.Item) error {
	notes, err := a.notes.ListByProject(projectID)
	if err != nil {
		return err
	}
	groups, err := a.noteGroups.ListByProject(projectID)
	if err != nil {
		return err
	}
	noteIDs := make([]string, len(notes))
	for i, n := range notes {
		noteIDs[i] = n.ID
	}
	groupIDs := make([]string, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
	}
	if err := notelayout.Validate(layout, noteIDs, groupIDs); err != nil {
		return err
	}
	return a.notes.SaveLayout(projectID, layout)
}
