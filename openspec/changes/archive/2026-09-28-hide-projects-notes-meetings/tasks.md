# Tasks

## 1. Data and domain

- [x] 1.1 Add migration 3 in `adapters/sqlite/db.go` with `hidden BOOLEAN NOT NULL DEFAULT 0` on `projects`, `notes` and `meetings`; verify with a test that a database at version 2 upgrades, existing rows read as visible, and running `migrate` twice is a no-op
- [x] 1.2 Add `Hidden bool` (`json:"hidden"`) to `domain.Project`, `domain.Note` and `domain.Meeting`, and read it in every repository query that builds these entities; verify existing repository tests still pass
- [x] 1.3 Add `SetHidden(id string, hidden bool) error` to the three repository ports and the SQLite repositories, without touching `updated_at` or any index; verify with repository tests (persists, reads back, `updated_at` unchanged, unknown id handled like the other methods)

## 2. Backend bindings and counters

- [x] 2.1 Add `SetProjectHidden`, `SetNoteHidden` and `SetMeetingHidden` on `App`, each returning the updated entity; `SetProjectHidden` returns `domain.ErrProjectLocked` for "Transverse / Divers"; verify with `App` tests including the locked case
- [x] 2.2 Extend `stats.SidebarStats` with `HiddenNotesCount` and `HiddenMeetingsCount`, make `NotesCount`/`MeetingsCount` count visible items only, and update `GetSidebarStats`; verify with unit tests on `stats.Sidebar` (none hidden, some hidden, all hidden, other projects ignored)
- [x] 2.3 Add tests proving hiding is display-only: after hiding a project, `GetPriorityTasks` still returns its tasks and the tray input (`tasks.ListAll`) still includes them; after hiding a note, `SearchGlobal` still finds it; verify they pass

## 3. Frontend

- [x] 3.1 Regenerate the Wails bindings (`wails generate module`) and update `frontend/src/demoBackend.ts` (hidden field, three setters, extended sidebar stats); verify `tsc --noEmit` passes in `frontend/`
- [x] 3.2 Add the icon-only `HiddenToggle` component (inline SVG open/closed eye, `aria-pressed`, `aria-label`, `title`) and hold the three "show hidden" booleans in `App` state, default closed-to-hidden (eye open), not persisted across restarts; verify with a component test for label, `aria-pressed` and click
- [x] 3.3 `ProjectSidebar`: add the toggle row, filter hidden projects by the eye state, render hidden ones with reduced opacity and italic name, add "Cacher"/"Réafficher" to the context menu (disabled for the locked project), and render the counters as `N(J) notes · M(K) réunions` with `(J)`/`(K)` only when above zero; verify with tests for each of these, including that counters do not change with the eye
- [x] 3.4 `App`: when the selected project stops being displayed (hidden, or eye opened on it), select the first displayed project or none; verify with tests for hide-selected and open-eye-on-hidden-selection
- [x] 3.5 `NotesTab`: add the toggle, filtering, reduced-opacity rendering, "Cacher"/"Réafficher" menu entries, and the selection fallback on the displayed list (including the initial selection); verify with component tests
- [x] 3.6 `MeetingsTab`: same as 3.5 for the meetings list, plus revealing a hidden target meeting (closing the meetings eye through a callback) when opened with `cibleInstance`, without the fallback replacing it; verify with component tests
- [x] 3.7 `App.ouvrir`: when the target project is hidden, close the projects eye so it is displayed and selected; verify with a test opening a task of a hidden project from Priorités while the eye is open
- [x] 3.8 Confirm the state of each toggle is independent of the other two and survives switching projects; verify with a test that closes the notes eye, changes project, and checks the notes list still shows hidden notes while the meetings eye is unchanged

## 4. Release notes and integration

- [x] 4.1 Add a line about hiding projects, notes and meetings to the existing `1.1.0` entry of `releases.json` (created by `app-version-and-about`, already archived) and verify the release-notes guard test still passes
- [x] 4.2 Run `go test ./...` and `npm test` in `frontend/`, then launch the app (`wails dev`) and confirm by hand: hide a project, a note and a meeting via right-click; toggle each eye independently; hidden items appear semi-transparent when shown; counters read `N(J)`; a hidden project's tasks still appear in Priorités
