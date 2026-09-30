# Proposal

## Why

Version 1.1.0 shipped with two editing bugs. One blocks work: while editing the minutes (CR) of a meeting instance, every autosave moves the selection to the first meeting of the list and takes the user out of the editor. The other is minor: the formatting toolbar (Gras, Italique, …) does not show the formatting at the cursor until the user types. Version 1.1.1 fixes both.

## What Changes

- **Meeting minutes keep their selection across autosave (blocking).** Saving an instance's minutes (after 2 s of inactivity, or on blur) no longer changes the selected meeting or instance, and no longer takes the editor out of the page. Root cause, found in the code: each save calls `onDataChanged`, which re-renders `App`. `App` passes `onRevelerCachee` to `MeetingsTab` as an inline arrow function, so a new function is created on every render. The initial-load effect of `MeetingsTab` depends on that callback, so it runs again and resets the selection to `liste[0]`. This regression came with the hide feature in 1.1.0.
- **The formatting toolbar follows the cursor (minor).** The pressed state of the toolbar buttons (Gras, Italique, Barré, Titre, Liste à puces, Liste numérotée, Bloc de code, Lien) updates as soon as the cursor or selection moves (click, arrow keys, selection), not only after typing. Root cause: TipTap 3 does not re-render the React component on transactions by default. Today the toolbar only refreshes because typing changes the parent's state.
- The version goes to `1.1.1` in `wails.json`, and `releases.json` gets a `1.1.1` entry listing both fixes (required by the existing release-notes check).

## Capabilities

### New Capabilities
- `rich-editor-toolbar`: the formatting toolbar of the rich editor shows the formatting at the current cursor or selection, in real time.
- `meeting-minutes-editing`: editing and autosaving a meeting instance's minutes keeps the selected meeting and instance, and leaves the user in the editor.

### Modified Capabilities
<!-- None: `app-version` and `release-notes` requirements are unchanged; bumping the version and adding a release entry is a use of those requirements, not a change to them. -->

## Impact

- `frontend/src/components/RichEditor.tsx`: the toolbar's active state is recomputed from editor state on selection and transaction changes.
- `frontend/src/App.tsx` and `frontend/src/components/MeetingsTab.tsx`: stable callback identity, and the initial selection effect no longer re-runs on parent re-renders.
- `wails.json` (`info.productVersion` → `1.1.1`), `releases.json` (new `1.1.1` entry).
- New frontend tests. No backend, API or data change.
