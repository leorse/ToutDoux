# Design

## Context

See proposal.md for motivation. Current state relevant to the approach:

- `notes` has no position column; `NoteRepository.ListByProject` sorts by `updated_at DESC` (`adapters/sqlite/note_repo.go`). `domain.Note` carries no order or group field.
- Migrations are a versioned list in `adapters/sqlite/db.go` (last one: 3), each run in a transaction; `migration_test.go` already tests an upgrade from the previous version.
- Project deletion is an explicit list of `DELETE` statements in `ProjectRepository.Delete`; there is no reliance on foreign-key cascades.
- Pure business rules live in `domain/<package>` (`tasktree`, `cascade`, `stats`) and are unit-tested without a database; `app.go` only orchestrates.
- `NotesTab.tsx` keeps `notes` and a single `selectedId`, renders a flat `<ul>` of buttons, uses `ContextMenu` for right-click and `window.prompt` / `window.confirm` for every name and confirmation in the app. Autosave (`useAutosave`, 2 s) writes through `UpdateNote` and replaces the note in state with the returned one.
- Drag and drop exists only in `TaskTree.tsx`: native HTML5 events, id passed in `dataTransfer` under a custom MIME type.
- `demoBackend.ts` mirrors every binding for the browser demo and for the frontend tests.

## Goals / Non-Goals

**Goals:**
- One persisted representation from which the list order, the group membership and the group positions all derive, so they cannot contradict each other.
- Every layout change written atomically.
- Drag-and-drop arithmetic testable without a DOM.

**Non-Goals:**
- Dragging several selected notes at once: the multi-selection only feeds "Regrouper"; a drag moves the dragged note alone.
- Nested groups, collapsing a group, group colours.
- Keyboard reordering.
- Manual ordering or grouping of meetings or projects.
- A custom modal component: popups stay `window.prompt`.

## Decisions

### 1. Flat order on notes; a group is a contiguous run

`notes` gets `order_index INTEGER NOT NULL DEFAULT 0` and `group_id TEXT NULL`. A new table `note_groups(id, project_id, name, created_at)` holds the name only. The list is `ORDER BY order_index, id`. Invariant: the notes of a group are contiguous in that order. A group has no position of its own: it sits where its notes are.

- *Why*: there is one ordering to maintain, and a group can never be "between" positions or hold a stale position after its notes moved. Hidden notes keep their slot inside the run, so a group whose notes are all hidden reappears at the same place for free. "Empty group disappears" becomes a simple cleanup (no row with that `group_id`).
- *Alternative considered*: a top-level order shared by groups and ungrouped notes plus an inner order per group. Closer to the visual tree, but two orderings and a polymorphic position to keep consistent, for no behaviour the specs ask for.
- No SQL foreign key on `group_id`: the codebase deletes dependants explicitly and the invariant is enforced in the domain layer (decision 3).

### 2. Migration 4

```sql
CREATE TABLE note_groups (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, name TEXT NOT NULL, created_at DATETIME NOT NULL);
ALTER TABLE notes ADD COLUMN group_id TEXT;
ALTER TABLE notes ADD COLUMN order_index INTEGER NOT NULL DEFAULT 0;
UPDATE notes SET order_index = (
  SELECT COUNT(*) FROM notes n2
  WHERE n2.project_id = notes.project_id
    AND (n2.updated_at > notes.updated_at OR (n2.updated_at = notes.updated_at AND n2.id < notes.id)));
```

The backfill reproduces the pre-upgrade display order (most recently modified first), with `id` as a deterministic tie-break. Additive only; an older binary ignores the new columns.

### 3. Layout rules in a pure domain package, one transactional write

New package `domain/notelayout` working on an ordered slice of `{NoteID, GroupID}`:

- `Group(layout, selectedIDs, newGroupID)`: gathers the selected notes, in list order, at the position of the first of them.
- `Dissolve(layout, groupID)`: clears the group id, positions unchanged.
- `Validate(layout, knownNoteIDs, knownGroupIDs)`: same set of notes as stored, known groups only, each group contiguous.

New port `NoteGroupRepository` (`ListByProject`, `Create`, `Rename`, `Delete`) and `NoteRepository.SaveLayout(projectID, layout)`, which in one transaction rewrites `order_index` (0..n-1) and `group_id` for the project's notes and deletes the project's groups left without notes.

- *Why a full rewrite*: a project holds tens of notes; rewriting all rows is trivial and removes any gap/renumbering logic. Putting the "delete emptied groups" step in the same transaction makes the "Empty groups disappear" requirement hold after every operation, including regrouping notes taken from another group.

### 4. Bindings

| Binding | Behaviour |
|---|---|
| `GetNoteGroups(projectID)` | groups of the project (`id`, `projectId`, `name`) |
| `GroupNotes(projectID, name, noteIDs)` | trims and rejects a blank name (`ErrEmptyName`), creates the group, applies `notelayout.Group`, `SaveLayout` |
| `RenameNoteGroup(groupID, name)` | trims, rejects blank |
| `DissolveNoteGroup(groupID)` | `notelayout.Dissolve` + `SaveLayout` (which removes the group) |
| `SetNotesLayout(projectID, layout)` | `Validate` then `SaveLayout`; used by every drag and drop |

`domain.Note` gains `GroupID *string` (`groupId`) and `OrderIndex int` (`orderIndex`). `CreateNote` inserts with `order_index = MIN(order_index) - 1` for the project (no renumbering, negative values are fine until the next `SaveLayout`). `DeleteNote` removes the group if the note was its last. `NoteRepository.Update` and `SetHidden` must keep writing only their own columns so that editing or hiding never moves a note. `ProjectRepository.Delete` gains `DELETE FROM note_groups WHERE project_id = ?`.

- *Why the drag result is computed in the frontend*: the drop target and its upper/lower half are DOM facts, and with the eye closed the frontend still holds the full list (hidden notes included), so it can produce the complete layout. The backend only validates and stores. Grouping and dissolving are computed in Go because they need no DOM information and the demo backend can reuse the same simple rule.

### 5. Frontend structure

- `frontend/src/notesLayout.ts`, pure functions on the full notes array: `moveNote(notes, noteId, target)` with `target` = `{before|after: noteId}` (the note takes the target's group), `{intoGroup: groupId}` (last of the group) or `{end: true}` (ungrouped, last); `moveGroup(notes, groupId, target)` with `target` = before/after an ungrouped note, before/after a group, or end; `toLayout(notes)`. Inserting relative to the target in the *full* order gives the hidden-notes behaviour of the `note-ordering` spec without special cases.
- `NotesTab.tsx` derives its render items from the ordered notes: consecutive notes with the same `groupId` form one group block; a block with no displayed note is skipped. A group renders as `<li>` containing a `role="group"` `aria-labelledby` frame (`border-2`), a top band (`border-t-4`-like thicker band, truncated name) and an inner `<ul>`.
- Selection: `selectedId` stays the open note; a `Set<string>` holds the multi-selection. Shift range is computed over the displayed notes from the open note. The set is reset on project change and pruned to displayed notes.
- Drag and drop: native HTML5, as in `TaskTree`. Rows are `draggable` and carry `text/note-id`; the band is `draggable` and carries `text/note-group-id`; the dragged kind is also kept in component state since `dataTransfer` data is unreadable during `dragover`. Upper/lower half comes from `getBoundingClientRect` at `dragover`; the insertion mark is a 2 px line on the hovered edge; the hovered group gets a highlight ring when a note is dragged over it. A group dragged over another group resolves to before/after the *group*, never inside. Drops on the `<ul>` background mean "end".
- The band stops propagation of `contextmenu` and carries `data-ligne` so `surLeFond` ignores double-clicks on it.
- Each drop: `await save.flush()` → `SetNotesLayout` → `recharger()`. No `onDataChanged()`: counters do not depend on layout.

### 6. Popups

`window.prompt('Nom du groupe ?')` and `window.prompt('Nouveau nom ?', nom)`, like every other name entry in the app. A dedicated modal would be a new pattern for one feature.

## Risks / Trade-offs

- [Autosave response overwrites fresh layout fields: `enregistrer` replaces the note in state with the `UpdateNote` result, which may predate a layout write] → flush is awaited before the layout write, and `enregistrer` merges only `title`, `content`, `updatedAt` into the existing state entry.
- [Native drag and drop is poorly simulated in jsdom (`getBoundingClientRect` returns zeros)] → the arithmetic is covered by `notesLayout.test.ts`; component tests stub the rect and cover one case per drop kind.
- [Frontend and demo backend could send a layout breaking contiguity] → `Validate` rejects it; the frontend reloads the list on error.
- [Breaking habit: the list no longer floats the last edited note to the top] → stated in the release notes; the upgrade order equals the last displayed order.
- [Two windows/tray writing concurrently] → not applicable, notes are only edited from the main window.

## Migration Plan

Migration 4 runs at startup like the previous ones, inside a transaction. Rollback to 1.1.1: the old binary still works (it ignores `group_id`, `order_index` and `note_groups`, and sorts by `updated_at`); groups created meanwhile are simply not shown and reappear on re-upgrade. Deleting a project with 1.1.1 would leave orphan rows in `note_groups`, which are harmless.
