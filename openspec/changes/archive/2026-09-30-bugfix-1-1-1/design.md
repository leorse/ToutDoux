# Design

## Context

See proposal.md (Why / What Changes) for the two root causes. The relevant code is:

- `RichEditor.tsx` builds the editor with TipTap 3 `useEditor`. In v3, `shouldRerenderOnTransaction` is `false` by default, so a selection change does not re-render the component. The `Outil` buttons call `editor.isActive(...)` during render. Today they only refresh when a parent re-renders, which happens when typing calls `onChange` → `setNotes`.
- `MeetingsTab.tsx` has an initial-load effect with deps `[chargerReunions, cibleInstance?.meetingId, onRevelerCachee]`. When it runs, it sets the selection to `cibleInstance?.meetingId ?? liste[0]?.id`. `App.tsx` passes `onRevelerReunionCachee={() => reveler('reunions')}`, which is a new function on every `App` render. Each save calls `onDataChanged` → `signalerChangement` → `revision++` → `App` re-renders → the effect runs again → the first meeting is selected → `GetInstances` → its latest instance is selected → the editor shows another document and loses focus.
- `NotesTab` does not receive this callback and is not affected.

## Goals / Non-Goals

**Goals:**
- Fix both bugs at their root and add regression tests that fail on the current code.
- Protect `MeetingsTab` against unstable callback props from any parent, not only today's one.

**Non-Goals:**
- No change to the autosave timing, the flush rules, the toolbar's buttons or their look.
- No general audit of every inline callback in `App`. Only the ones feeding an effect's dependencies matter here.

## Decisions

### 1. Toolbar: subscribe with `useEditorState`
In `RichEditor`, compute the active state of all toolbar formats in one `useEditorState({ editor, selector })` call. It returns a flat object of booleans (`bold`, `italic`, `strike`, `heading2`, `bulletList`, `orderedList`, `codeBlock`, `link`). `Outil` receives a `pressed: boolean` prop instead of calling `editor.isActive` itself. Both `aria-pressed` and the pressed style come from that one value.

- *Why:* `useEditorState` re-renders only when the selected values change, not on every transaction. It is the pattern TipTap 3 recommends. Selection-only transactions are included, so clicks and arrow keys update the toolbar.
- *Alternative:* `shouldRerenderOnTransaction: true` on `useEditor`. It is a one-line fix, but it re-renders the whole editor wrapper on every keystroke and cursor move. Rejected for its cost and because it hides where the dependency is.

### 2. Meetings: stable callback in `App`, and the effect no longer depends on it
Two complementary changes:

1. `App.tsx`: wrap the three per-list callbacks passed to `ProjectView` (`onToggle…Caches`, `onRevelerReunionCachee`) in `useCallback` over the already-stable `basculerCaches` / `reveler`.
2. `MeetingsTab.tsx`: keep `onRevelerCachee` in a ref kept current by an effect declared before the load effect (the pattern `useAutosave` uses for `save`), and drop it from the initial-load effect's deps. The effect then only runs when the project (`chargerReunions`) or the navigation target (`cibleInstance?.meetingId`) changes, which is what it means.

- *Why both:* (1) alone fixes today's bug, but a future inline arrow would bring it back. (2) alone fixes it too, but leaves `App` producing needless changing props. Together they give defense in depth at very little cost.
- *Alternative:* in the effect, keep the current selection when it is still in the list (a functional `setMeetingId`). Rejected as the primary fix: after a real navigation, `cibleInstance.meetingId` stays set. If the effect kept running on every refresh, it would snap the selection back to the target and override later manual choices. The effect must not run on refreshes at all.

## Risks / Trade-offs

- [The `onRevelerCachee` ref could be read before it is set] → Effects run in declaration order, so the ref-updating effect runs before the load effect, which therefore always sees the current callback.
- [jsdom does not reproduce ProseMirror selection changes perfectly] → The toolbar tests move the cursor with editor commands (`setTextSelection`) and with clicks, and assert on `aria-pressed`. The manual check in the tasks covers real mouse and keyboard use.
- [Regression test for the meetings bug depends on autosave timing] → Trigger the save by blur (immediate flush) and, separately, with fake timers for the 2 s delay, so the test does not depend on real time.

## Migration Plan

No data migration. Ship as version 1.1.1 (`wails.json` + `releases.json` entry). To roll back, reinstall 1.1.0.
