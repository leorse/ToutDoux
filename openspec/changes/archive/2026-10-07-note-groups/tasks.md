# Tasks

## 1. Schema and domain

- [x] 1.1 Add migration 4 in `adapters/sqlite/db.go` (`note_groups` table, `notes.group_id`, `notes.order_index`, backfill by `updated_at DESC` per project) and add `TestMigration4_UpgradesFromVersion3` in `migration_test.go` verifying that notes of a version-3 database come out with `order_index` following their former display order and a null `group_id`
- [x] 1.2 Add `GroupID *string` / `OrderIndex int` to `domain.Note` and a `domain.NoteGroup` entity in `domain/model.go`; verify `go build ./...` passes
- [x] 1.3 Create `domain/notelayout` (`Group`, `Dissolve`, `Validate`) with table-driven tests covering: grouping A, C, D out of A, B, C, D; grouping a single note; regrouping a note taken from another group; dissolving in place; validation rejecting a missing note, an unknown group and a non-contiguous group — verify `go test ./domain/notelayout` passes

## 2. Persistence

- [x] 2.1 Update `NoteRepository` (port and sqlite adapter): read the two new columns, `ListByProject` ordered by `order_index, id`, `Create` inserting at `MIN(order_index) - 1`, `Update` and `SetHidden` leaving order and group untouched; verify with repository tests that a new note comes first and that updating or hiding a note does not change the list order
- [x] 2.2 Add `NoteRepository.SaveLayout(projectID, layout)` (single transaction: rewrite `order_index` and `group_id`, delete the project's groups without notes); verify with tests that the order and membership are read back and that an emptied group is deleted
- [x] 2.3 Add the `NoteGroupRepository` port and its sqlite adapter (`ListByProject`, `Create`, `Rename`, `Delete`); verify with a create/rename/list/delete repository test
- [x] 2.4 Make `NoteRepository.Delete` remove the group when the deleted note was its last, and add `DELETE FROM note_groups WHERE project_id = ?` to `ProjectRepository.Delete`; verify with tests "last note deleted removes the group" and "deleting a project with groups leaves no note and no group"

## 3. Application bindings

- [x] 3.1 Wire the group repository in `App` and add `GetNoteGroups`, `GroupNotes`, `RenameNoteGroup`, `DissolveNoteGroup`, `SetNotesLayout` in `app.go`; add `app_note_groups_test.go` covering: grouping places the group at the first selected note, blank name refused for group and rename, rename persists, dissolve keeps the notes in place, `SetNotesLayout` rejects an invalid layout, `GetSidebarStats` notes counters unchanged after grouping — verify `go test ./...` passes
- [x] 3.2 Regenerate the Wails bindings (`wails generate module`) and verify `frontend/wailsjs/go/main/App.d.ts` and `models.ts` expose the five bindings, `NoteGroup`, `groupId` and `orderIndex`
- [x] 3.3 Mirror the bindings in `frontend/src/demoBackend.ts` (ordered notes, groups, new note first, empty-group cleanup on layout write and note deletion) and export them from `api.ts`; verify `npm run build` type-checks

## 4. Frontend layout logic

- [x] 4.1 Create `frontend/src/notesLayout.ts` (`moveNote`, `moveGroup`, `toLayout`, and the derivation of render blocks from ordered notes with the hidden filter) with `notesLayout.test.ts` covering every drag scenario of the `note-ordering` and `note-groups` specs, including: move up, move to end, drop at own place is a no-op, hidden note keeps its place, drop on band appends, drop inside a group, move between groups, drag out of a group, group moved after a note, group released over another group lands before/after it, fully hidden group is skipped — verify `npx vitest run notesLayout` passes

## 5. Notes list UI

- [x] 5.1 Render groups in `NotesTab.tsx` (thicker frame, thicker top band with truncated name, `role="group"` labelled by the name, notes inside; band excluded from background double-click and right-click) and load groups alongside notes; verify with component tests "group frame shows its name and notes", "accessible group name", "double-click on the band creates no note", "partly hidden / entirely hidden group"
- [x] 5.2 Implement the multi-selection (click, Ctrl+click, Shift+click, distinct styles for selected and open note, reset on project change); verify with component tests for the three selection scenarios and that the editor keeps showing the open note
- [x] 5.3 Add "Regrouper" to the note menu (right-click outside the selection first reduces it to that note; `window.prompt` for the name; blank or cancelled does nothing) and the band menu "Renommer le groupe" / "Dissoudre le groupe"; verify with component tests: grouping three notes, grouping a single note, cancelled prompt, rename, blank rename refused, dissolve
- [x] 5.4 Implement note drag and drop (draggable rows, upper/lower half, insertion mark, group highlight, drop on band, drop on list background, release outside changes nothing) calling `save.flush()` then `SetNotesLayout`; make `enregistrer` merge only title, content and `updatedAt`; verify with component tests using a stubbed `getBoundingClientRect`: reorder, add to group via band, drag out of group, and "typed text is saved and the new order is kept"
- [x] 5.5 Implement group drag from the top band only (before/after notes and groups, never inside a group, end of list); verify with component tests "group moved below a note keeps its content" and "dragging a note of a group moves only that note"
- [x] 5.6 Keep the new-note flow consistent: the created note is first, ungrouped and opened; verify with a component test in a project that already has a group

## 6. Version and release notes

- [x] 6.1 Set `productVersion` to `1.2.0` in `wails.json` and add the `1.2.0` entry at the top of `releases.json` (manual order of notes — editing no longer moves a note to the top —, groups, drag and drop); verify the existing version/release-notes tests (`app_about_test.go`, `aPropos.test.tsx`) pass

## 7. Integration check

- [x] 7.1 Run `go test ./...`, `npm test` and `npm run build` in `frontend`, and `openspec validate note-groups --strict`; verify all pass
- [x] 7.2 Run the app (`wails dev`) on a copy of a 1.1.1 database and verify by hand: order preserved after upgrade, group three notes, drag a note in and out, move a group by its band, rename, dissolve, restart and find the same layout
