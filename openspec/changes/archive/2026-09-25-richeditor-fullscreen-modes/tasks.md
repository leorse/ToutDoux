# Tasks

## 1. `RichEditor` view-mode state and overlay

- [x] 1.1 Add `viewMode` state (`'inline' | 'readonly-full' | 'editable-full'`) to `RichEditor`, defaulting to `'inline'`, and verify it compiles with no change in default (inline) rendering
- [x] 1.2 Render the existing editor tree inside a wrapper that switches to a full-viewport overlay (`fixed inset-0 z-50 ...`) when `viewMode !== 'inline'`, and verify manually that toggling the state visually covers the whole window over Notes and Meetings layouts
- [x] 1.3 Add an `Escape` keydown listener, active only while a full-window mode is open, that resets `viewMode` to `'inline'`, and verify it does not fire when `viewMode === 'inline'` (e.g. does not interfere with other Escape handling elsewhere)
- [x] 1.4 Add a visible close control in the overlay (in addition to Escape) that resets `viewMode` to `'inline'`, and verify it is reachable without a keyboard

## 2. Read-only ("eye") control

- [x] 2.1 Add an "eye" toolbar button that sets `viewMode = 'readonly-full'`, calling `editor.setEditable(false)` and hiding the toolbar while active, and verify content displays full-window and cannot be edited (typing has no effect)
- [x] 2.2 On closing the read-only view, restore `editor`'s editable state from the current `readOnly` prop (not a cached value), and verify editing still works afterward when the parent is not read-only

## 3. Editable ("expand") control

- [x] 3.1 Add an "expand" toolbar button that sets `viewMode = 'editable-full'`, keeping the toolbar visible and the editor editable, and verify typing/formatting works the same as inline
- [x] 3.2 Verify `onChange`/`onBlur` callbacks still fire while in `editable-full` mode (e.g. by checking the Notes/Meetings autosave status text updates while the overlay is open)
- [x] 3.3 Verify edits made in `editable-full` mode are preserved after closing back to `'inline'` (content matches what was typed)

## 4. Scope check

- [x] 4.1 Confirm both controls render for `RichEditor` instances in `NotesTab` and `MeetingsTab` with no changes required in those files, and verify by running the existing frontend test suite (`npm test` in `frontend/`) plus a manual check in both tabs
- [x] 4.2 Confirm the task description field (`TaskDetail.tsx` textarea) is untouched and has no full-window controls
