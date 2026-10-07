# Proposal

## Why

A project's notes are a flat list sorted by last modification: the order keeps shifting as notes are edited, and there is no way to keep related notes together. Version 1.2.0 lets the user arrange the notes list by hand and gather notes into named groups.

## What Changes

- **Manual order of the notes list.** Notes are reordered by drag and drop and keep the position the user gave them. **BREAKING** (behaviour): the list is no longer sorted by last modification, so editing a note no longer moves it to the top. Existing notes keep the order they had at upgrade time. A new note is added at the top of the list, outside any group.
- **Multi-selection of notes** in the list (Ctrl+click to toggle, Shift+click for a range), used to pick the notes to group. The note shown in the editor stays a single note.
- **Grouping.** Right-clicking the selection offers "Regrouper" ; a popup asks for the group name and the selected notes are gathered into a new group placed where the first of them was.
- **Group rendering.** A group is a frame with a slightly thicker border than a note row and a thicker top band carrying the group name; its notes are listed inside the frame.
- **Drag and drop with groups.** Dropping a note onto a group adds it to that group; dragging a note out of its group's frame removes it from the group. A group is moved as a whole by dragging its top band. Groups do not nest.
- **Group menu** on the top band: "Renommer le groupe" (popup pre-filled with the current name) and "Dissoudre le groupe" (the notes stay, at the group's position).
- A group that loses its last note (moved out or deleted) disappears. Deleting a project deletes its groups.
- Hiding (v1.1.0) keeps working inside groups: a hidden note is filtered by the notes eye as before, and a group none of whose notes is displayed is not displayed.
- Application version becomes `1.2.0`, with a `1.2.0` entry in the release notes.

## Capabilities

### New Capabilities
- `note-ordering`: user-defined, persistent order of a project's notes list, changed by drag and drop.
- `note-groups`: multi-selection of notes, creation of named groups from a selection, group rendering, adding/removing notes by drag and drop, moving, renaming and dissolving groups.

### Modified Capabilities
(none — `item-hiding` requirements are unchanged; how hidden notes behave inside groups is specified in `note-groups`)

## Impact

- Affected code: SQLite migration 4 (new `note_groups` table, `group_id` and `order_index` columns on `notes`), `domain` entities, `ports.NoteRepository` and a new group repository, `adapters/sqlite` (note repository, project cascade delete), `App` bindings in `app.go` (groups listing, grouping, renaming, dissolving, layout write; note creation and deletion), `frontend/src/components/NotesTab.tsx` (multi-selection, drag and drop, group frames, menus), `demoBackend.ts`, regenerated `frontend/wailsjs` bindings, `wails.json` (`productVersion`), `releases.json`.
- No change to search, semantic index, Priorités, tray or sidebar counters.
- No new dependency (native HTML drag and drop, as in the task tree). Non-destructive migration.
