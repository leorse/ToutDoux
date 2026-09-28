# Design

## Context

Projects, notes and meetings live in SQLite tables `projects`, `notes`, `meetings`; schema changes are append-only entries in `migrations` (`adapters/sqlite/db.go`, currently versions 1-2). Repositories implement ports in `ports/ports.go`. `App` methods in package `main` are the Wails bindings; the frontend calls them only through `api.ts` (typed from generated `wailsjs`, mocked by `demoBackend.ts`). `ListProjects`, `GetNotes`, `GetMeetings` return full lists, ordered by the backend (`stats.SortProjects`); the sidebar counters come from the pure function `stats.Sidebar`. `NotesTab` and `MeetingsTab` are remounted on every project change (`key={projectId}`), and task filters already live in `App` state so they persist across projects. Priorités and the tray read `tasks.ListAll()` regardless of project. See proposal.md - Why.

## Goals / Non-Goals

**Goals:**
- A `hidden` flag on the three entities, purely presentational: no query feeding Priorités, tray, notifications or search changes.
- A per-list eye filter whose state survives project switches.

**Non-Goals:**
- No archiving semantics, no auto-hiding of completed projects, no bulk hide.
- No "hidden since" date.
- No change to search results (hidden items stay findable and are not marked in results).
- No new note navigation: opening a search hit on a note still only opens the Notes tab.

## Decisions

- **`hidden BOOLEAN NOT NULL DEFAULT 0` on the three tables**, added by migration 3 with three `ALTER TABLE ... ADD COLUMN`. Existing rows become visible; the migration is non-destructive and follows the append-only rule. Alternative: `archived_at DATETIME NULL` — rejected, no use for the date.
- **`SetHidden(id string, hidden bool) error` on `ProjectRepository`, `NoteRepository` and `MeetingRepository`**, rather than overloading `Update`/`Rename`. It does not touch `updated_at` (hiding is not a content edit) and does not touch the search or semantic indexes.
- **Three `App` bindings**: `SetProjectHidden`, `SetNoteHidden`, `SetMeetingHidden`, each returning the updated entity. `SetProjectHidden` refuses the locked project with `domain.ErrProjectLocked`, the same guard as rename/delete.
- **Lists stay complete on the backend; filtering is in the frontend.** `ListProjects`/`GetNotes`/`GetMeetings` return everything with a `hidden` field, because the eye state is UI state and the backend ordering rule stays the single definition of order (hidden items are not reordered). Alternative: an `includeHidden` parameter on each call — rejected, it would push view state into the backend and force re-fetching on every eye click.
- **Counters computed in `stats.Sidebar`**: `NotesCount`/`MeetingsCount` become *visible* counts and `HiddenNotesCount`/`HiddenMeetingsCount` are added. Pure function, unit-tested. The frontend renders `N(J)` with `(J)` only when `J > 0`, keeping the existing separators and wording (`N(J) notes · M(K) réunions`).
- **Eye state lives in `App`**, next to the task filters: `{ projets, notes, reunions }` booleans meaning "show hidden", default `false`, passed to `ProjectSidebar`, `NotesTab`, `MeetingsTab`. Lifting is required because the tabs remount on project change; state is not persisted across restarts, like the task filters.
- **One small `HiddenToggle` component**, icon-only (inline SVG, open eye / closed eye), `aria-pressed`, `aria-label`, `title`, styled like `BoutonFiltre` in `TaskFilters.tsx`. Its accessible label describes the action ("Afficher les éléments cachés" / "Masquer les éléments cachés"). It sits in a slim row above each list, outside the list's create-on-double-click background (verify against `surLeFond`).
- **Appearance of a hidden item**: reduced opacity (`opacity-50`) and italic title, background untouched. This keeps the criticality colour of a hidden project (its tasks stay in Priorités) and does not collide with the flat grey of "Transverse / Divers".
- **Selection rule computed on the displayed list**: when the selected id is not in the displayed list (item hidden, or eye opened on it), select the first displayed item, or none. Initial selection after a load also uses the displayed list.
- **Revealing a navigation target**: `App.ouvrir` closes the projects eye when the target project is hidden (it knows the project list). `MeetingsTab`, which already receives the target meeting/instance, closes the meetings eye through a callback when the target meeting is hidden, and skips the fallback while doing so. Notes are excluded: no note-level target exists today.
- **Context menu**: "Cacher" / "Réafficher" reuse the existing `ContextMenu` action items; for the locked project "Cacher" is present but `disabled`, matching how "Renommer"/"Supprimer" are handled.

## Risks / Trade-offs

- [Existing selection code assumes the first list item is selectable] → route it through the displayed list; covered by tests for hide-selected and eye-open-on-hidden.
- [Counters change meaning: `notesCount` was total] → nothing else reads it; grep before changing and update the demo mock.
- [A hidden project with critical tasks still colours Priorités/tray, which may surprise] → intended by the requirements; documented in the spec.
- [Reveal-on-navigation flips a filter the user did not touch] → deliberate and limited to the exact list of the target; the alternative is an unreachable target.
- [Bindings are generated] → regenerate `wailsjs` after adding the three methods and the new fields, and commit the result.
- [Release notes coupling] → this feature ships as part of `1.1.0`, not a new version bump: this change adds a line to the existing `1.1.0` entry of `releases.json`, created by the now-archived `app-version-and-about`.

## Migration Plan

Migration 3 runs at startup like migrations 1-2. It only adds columns with a default, so rollback is simply not using them; no data is rewritten.
