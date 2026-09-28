# Proposal

## Why

Projects, notes and meetings that are no longer current keep piling up in their lists, burying the ones in use. There is currently no way to get them out of the way short of deleting them, which is destructive and cascades to tasks.

## What Changes

- Projects, notes and meetings gain a persistent "hidden" state, set and cleared from the right-click menu ("Cacher" / "Réafficher"). Hiding is purely visual: tasks of a hidden project keep appearing in Priorités, the system tray, notifications and search, exactly as before.
- Each of the three lists (projects, notes, meetings) gets its own icon-only eye toggle, following the icon filter pattern of the task filters:
  - open eye (default): hidden items are not shown;
  - closed eye: all items are shown, hidden ones rendered semi-transparent.
  The state persists across project switches and resets on restart.
- The project sidebar counters become `N(J) Notes et M(K) Réunions`: N/M visible notes/meetings, J/K hidden ones. `(J)` and `(K)` are shown only when greater than 0.
- Selection falls back to the first visible item when the selected item becomes hidden or the eye closes on it. Opening a search result or priority task that targets a hidden project or meeting reveals it (the relevant eye closes) so the target is visible. Notes are unaffected: the existing navigation only opens the Notes tab and does not select a specific note.
- The locked "Transverse / Divers" project cannot be hidden.

## Capabilities

### New Capabilities
- `item-hiding`: hiding and revealing projects, notes and meetings, with per-list visibility filters and hidden-aware counters.

### Modified Capabilities
(none — no existing main spec covers these lists)

## Impact

- Affected code: SQLite migration 3 (a `hidden` column on `projects`, `notes`, `meetings`), the `domain` entities and repositories, `App` bindings (hide/reveal per entity, extended `GetSidebarStats`), `ProjectSidebar.tsx`, `NotesTab.tsx`, `MeetingsTab.tsx`, `App.tsx` (search/priority navigation revealing hidden targets), `demoBackend.ts`, regenerated `frontend/wailsjs` bindings.
- No change to priorities, tray, notification or search queries.
- No new dependency; no destructive migration (existing rows default to visible).
