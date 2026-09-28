# Design

## Context

`RichEditor` (frontend/src/components/RichEditor.tsx) wraps TipTap and is used by `NotesTab.tsx` and `MeetingsTab.tsx`, each placing it in the right-hand pane of a `Split` component. No ancestor in that tree uses `transform`, `filter`, or `perspective`, so a `position: fixed` element inside `RichEditor` reliably covers the whole viewport regardless of the surrounding split/column layout. The app runs as a single Wails v2 window; Wails v2 has no supported API to open a second native window, so "full window" here means the existing window, not a separate one. See proposal.md - Why.

## Goals / Non-Goals

**Goals:**
- Add the two view modes entirely inside `RichEditor`, so `NotesTab` and `MeetingsTab` need no changes.
- Reuse the existing TipTap `editor` instance for both modes (no re-mount, no content re-fetch) so no edits are lost when toggling.

**Non-Goals:**
- No separate OS window (out of reach on Wails v2 stable).
- No change to the task description field (plain textarea, not `RichEditor`).
- No change to autosave/debounce logic in `NotesTab`/`MeetingsTab` — `onChange`/`onBlur` continue to fire exactly as they do inline.

## Decisions

- **Single component owns the toggle state**: `RichEditor` gets a local state, e.g. `viewMode: 'inline' | 'readonly-full' | 'editable-full'`, defaulting to `'inline'`. This keeps the feature self-contained per the proposal's Impact section, rather than lifting state into each tab.
- **Overlay via CSS, not a portal**: render the same JSX tree, switching the wrapper's class to `fixed inset-0 z-50 bg-white` (or similar) when a full-window mode is active, instead of using a React portal. Simpler, and avoids re-parenting the TipTap `EditorContent` (which can lose selection/focus on remount). Confirmed safe because no ancestor sets `transform`/`filter`.
- **Read-only mode reuses TipTap's `editable` toggle**: switching `viewMode` to `'readonly-full'` sets the underlying editor `editable: false` for the duration (via `editor.setEditable(false)`) and hides the toolbar block, independent of the `readOnly` prop passed by the parent. Reverting restores the prop-derived editable state.
- **Escape-to-close**: a single `keydown` listener attached while any full-window mode is active, removed on cleanup — mirrors the pattern already used elsewhere in the app for dismissing overlays (e.g. context menu).
- **Two distinct controls, not one toggle**: an "eye" button (read-only) and an "expand" button (editable), both always visible in the toolbar (toolbar itself only renders when `!readOnly`, per existing behavior) so the two scenarios in the spec map directly to two buttons rather than one control with a mode side-effect.

## Risks / Trade-offs

- [Full-window overlay could visually clash with modals/menus (e.g. `ContextMenu`) that also use high z-index] → Use a z-index high enough to sit above normal content but check `ContextMenu`'s stacking; since the overlay is opened by explicit user action and the context menu closes on outside click, conflicts are expected to be rare and non-breaking.
- [Toggling `editor.setEditable(false)` and back could interact oddly if the parent's `readOnly` prop changes while full-window is open] → On close, always resolve editability from the current `readOnly` prop rather than a cached value.
