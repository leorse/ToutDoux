# Tasks

## 1. Colour: storage and bindings

- [x] 1.1 Add migration 5 in `adapters/sqlite/db.go` (`color TEXT NOT NULL DEFAULT ''` on `notes` and `meetings`) and a `TestMigration5_UpgradesFromVersion4` verifying that a note and a meeting written before the migration are read back with an empty colour
- [x] 1.2 Add `Color` to `domain.Note` and `domain.Meeting`, the list of the ten palette keys, a validation function and `ErrInvalidColor` in `domain`; verify with a unit test that the ten keys and the empty string are accepted and anything else rejected
- [x] 1.3 Add `SetColor` to the note and meeting repositories (ports and sqlite adapters), read the column in both scans, leave `updated_at` untouched; verify with repository tests that the colour round-trips, that `NoteRepository.Update` does not erase it, and that an unknown id returns `ErrNotFound`
- [x] 1.4 Add the bindings `SetNoteColor` and `SetMeetingColor` in the app layer; verify with app tests: colour set and returned, removed with an empty string, invalid key refused with `ErrInvalidColor`, sidebar counters and note position unchanged
- [x] 1.5 Regenerate the Wails bindings and mirror `SetNoteColor` / `SetMeetingColor` in `frontend/src/demoBackend.ts`; verify `npm run build` type-checks

## 2. Colour: interface

- [x] 2.1 Create `frontend/src/couleurs.ts` (ten keys, hex values, French labels, lookup returning nothing for an unknown key) with a test pinning the ten keys and checking the unknown-key case
- [x] 2.2 Add the optional `color` / `onColorChange` props and the colour button with its palette popover to `RichEditor.tsx`, right after the full-screen buttons; verify in `RichEditor.test.tsx`: no button without the prop, button adjacent to the full-screen buttons, ten swatches plus "Aucune couleur", current colour marked, choosing calls the handler and closes, Escape closes without calling it and keeps a full-window view open
- [x] 2.3 Wire the colour in `NotesTab.tsx` (flush pending typing, `SetNoteColor`, merge `color` into state) and render coloured rows with the open/selected rings; verify with component tests: choosing a colour colours the row, "Aucune couleur" removes it, a coloured note inside a group is coloured, the open coloured note is still marked as open, typed text is saved when a colour is chosen right after typing
- [x] 2.4 Wire the colour in `MeetingsTab.tsx` on the selected meeting and render coloured meeting rows; verify with component tests: the meeting row is coloured, instance rows are not, and the palette shows the same current colour after switching instance

## 3. Transverse project tint

- [x] 3.1 Add `--color-critique-attenuee` and `--color-haute-attenuee` to `style.css` and make `background()` in `ProjectSidebar.tsx` use them for the locked project; verify with component tests using a stubbed `GetSidebarStats`: attenuated Critique, attenuated Haute, grey without important task, ordinary project unchanged, and back to grey after the stats change

## 4. Due information on finished tasks

- [x] 4.1 Make `GetTaskView` compute due information for active tasks only; verify with app tests: completed overdue task absent from `Due`, cancelled task absent, active task present, task present again after being unchecked
- [x] 4.2 Key the "remove due date" button of `TaskDetail.tsx` on the task's due date instead of the label, and align the demo backend's task view with the new rule; verify with component tests that a completed task with a past due date shows no "En retard" and no clock icon in the tree and in the detail, and that an active one still does

## 5. Categorised release notes

- [x] 5.1 Change the release model in `app_about.go` (`Change` with `type`, `level`, `text`) and extend `checkReleases`; verify in `app_about_test.go`: valid history passes, unknown kind fails naming the version, missing level fails, empty text fails
- [x] 5.2 Convert every entry of `releases.json` to the new shape following the classification table of design.md, and rewrite the `1.2.0` entry (groups and manual order, colours, transverse tint, finished-task due date — no mention of the categorisation itself); verify `TestEmbeddedReleasesMatchVersion` passes
- [x] 5.3 Regenerate the Wails bindings, update the demo backend's `GetAppInfo`, and render "Ajouts" / "Corrections" with major-first order and the "Majeur" mark in `AProposView.tsx`; verify in `aPropos.test.tsx`: additions and fixes under their headings, no "Ajouts" heading for a fixes-only release, major before minor and marked, important notice, emphasis and literal HTML still working

## 6. Integration check

- [x] 6.1 Run `go test ./...` (from PowerShell), `npm test` and `npm run build` in `frontend`, and `openspec validate v1-2-0-finitions --strict`; verify all pass
- [ ] 6.2 Run the app (`wails dev`) and verify by hand: colour a note and a meeting and find them after a restart, instances stay neutral, the transverse project tints with a Critique then a Haute task and returns to grey, a completed overdue task shows no lateness, and the "À propos" tab shows "Ajouts" and "Corrections" for 1.2.0
