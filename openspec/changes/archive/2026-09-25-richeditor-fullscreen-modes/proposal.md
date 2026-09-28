# Proposal

## Why

In Notes and Meetings, the rich text editor sits in the right-hand column of a resizable split view, squeezed next to the sidebar and the rest of the layout. Pasted images render at that column's narrow width, making them hard to read. Users need a way to view and edit that content at the full size of the application window.

## What Changes

- Add two full-window overlay modes to `RichEditor`, triggered from its toolbar:
  - An "eye" button that opens a read-only full-window view (toolbar hidden, no editing).
  - An "expand" button that opens an editable full-window view (toolbar visible, same editing behavior as inline).
- Both modes overlay the entire application window (not a separate OS window — Wails v2 does not support multi-window) and can be closed via a close control or the Escape key, returning to the normal column layout.
- Applies to both places `RichEditor` is used: the Notes tab and the Meetings tab. The task description field (plain textarea, no image support) is out of scope.

## Capabilities

### New Capabilities
- `rich-editor-view-modes`: full-window read-only and editable overlay modes for the rich text editor used in Notes and Meetings.

### Modified Capabilities
(none — no existing specs cover the rich editor's toolbar or layout)

## Impact

- Affected code: `frontend/src/components/RichEditor.tsx` (new overlay state and toolbar buttons), consumers `NotesTab.tsx` and `MeetingsTab.tsx` (no API change expected, since the toggle is internal to `RichEditor`).
- No backend/API changes; no new dependencies.
