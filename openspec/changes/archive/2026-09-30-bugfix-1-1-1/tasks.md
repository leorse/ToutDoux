# Tasks

## 1. Meeting minutes keep their selection across autosave (blocking)

- [x] 1.1 Write a failing regression test (e.g. `frontend/src/reunionsSauvegarde.test.tsx`, rendering `<App />` on the demo backend): open the Réunions tab, select a meeting that is **not** first in the list and an instance that is **not** its most recent one, type in the minutes, then trigger the save (a) by blur and (b) by the 2 s delay with fake timers; assert the same meeting and instance stay selected (`bg-neutral-200` row), the editor still shows the typed text, and the save went to that instance. Verify the test fails on the current code.
- [x] 1.2 In `App.tsx`, wrap the per-list callbacks passed to `ProjectView` (`onToggleNotesCachees`, `onToggleReunionsCachees`, `onRevelerReunionCachee`) in `useCallback` over `basculerCaches` / `reveler`; verify with `tsc --noEmit` in `frontend/`.
- [x] 1.3 In `MeetingsTab.tsx`, keep `onRevelerCachee` in a ref kept current by an effect declared before the initial-load effect, and remove it from that effect's deps (keep `chargerReunions` and `cibleInstance?.meetingId`); verify the 1.1 test passes and the existing hidden-meeting navigation tests in `hiding.test.tsx` still pass.
- [x] 1.4 Add a test that renders `MeetingsTab` in a wrapper that passes a **new inline** `onRevelerCachee` and re-renders on every `onDataChanged`, then checks that a save leaves the selection unchanged; verify it passes, which shows the tab no longer depends on the parent keeping its callbacks stable.
- [x] 1.5 Add a test that focus stays in the editor across a save: while the minutes editor is focused, trigger the delayed save and assert `document.activeElement` is still inside the editor and further typed text is appended; verify it passes.

## 2. Formatting toolbar follows the cursor (minor)

- [x] 2.1 Write failing tests for `RichEditor` (e.g. `frontend/src/components/RichEditor.test.tsx`) with content `<p>plain <strong>bold</strong> <em><strong>both</strong></em></p><ul><li>item</li></ul>`: move the cursor with editor selection commands (no typing) into bold, into plain text, into "both" and into the list item, and assert `aria-pressed` of Gras / Italique / Liste à puces each time; repeat one case in the editable full-window view. Verify they fail on the current code.
- [x] 2.2 In `RichEditor.tsx`, compute all toolbar active states with a single `useEditorState({ editor, selector })` and pass a `pressed` boolean to `Outil`, which uses it for both `aria-pressed` and the pressed style instead of calling `editor.isActive`; verify the 2.1 tests pass and the existing `texteEnrichi.test.tsx` and rich-editor view-mode tests still pass.

## 3. Version 1.1.1

- [x] 3.1 Set `info.productVersion` to `1.1.1` in `wails.json` and add a `1.1.1` entry dated on the release day at the top of `releases.json`, listing both fixes (the meeting minutes no longer jump to the first meeting on save; the formatting buttons follow the cursor); verify `go test ./...` passes, including the release-notes consistency check in `app_about_test.go`.

## 4. Integration

- [x] 4.1 Run `go test ./...` and `npm test` in `frontend/`, then launch the app (`wails dev`) and confirm by hand: in a meeting other than the first, edit the minutes of an older instance, wait for "Sauvegardé à …" and keep typing without the selection or focus moving; in a note and in minutes, click and arrow into bold/italic/list text and see the buttons light up without typing, inline and in full-window edit mode.
- [x] 4.2 (Found while running 4.1, outside the two specs.) In `SearchView.tsx`, show similarity scores only for results that actually came from the semantic search: right after toggling 🧠, the keyword results stay on screen for the debounce delay without a score, which crashed the render on the demo backend (`r.score.toFixed` in `phase5.test.tsx`) and showed `0.000` in the real app; verify `npm test` reports no unhandled error.
