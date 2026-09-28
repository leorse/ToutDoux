# Proposal

## Why

The application has no version number anywhere: `wails.json` has no `info` block, and the only marker is a git tag. Users cannot tell which build they run, and there is no place to record what each release changed. The window title also still carries a leftover from the web prototype era (an explicit "no HTML title bar" comment) that no longer matters.

## What Changes

- Add a single source of truth for the application version: `info.productVersion` in `wails.json`, which the Wails Windows installer already reads, and which is also embedded into the binary.
- Show the version in the native window title, in the format `Tout Doux (1.1.0)`.
- Split the Preferences view into two sub-tabs: "Préférences" (current content) and a new "À propos".
- The "À propos" tab shows the current version and the release history, newest first, read from an embedded `releases.json`. Each release has a version, a date, a list of changes (bulleted), and an optional "important" notice. Text supports minimal inline formatting (bold, italic).
- Add a test that fails when the newest `releases.json` entry does not match `productVersion` in `wails.json`.
- Remove the obsolete prototype-era comments about the title bar in `main.go` and `App.tsx`.

## Capabilities

### New Capabilities
- `app-version`: the application version is defined in one configuration file and displayed in the native window title.
- `release-notes`: the "À propos" tab in Preferences, showing the current version and the versioned release history.

### Modified Capabilities
(none — no existing spec covers the window title or the Preferences view)

## Impact

- Affected code: `wails.json` (new `info` block), `main.go` (title built from embedded config, comment cleanup), a new Go file exposing app info and release notes to the frontend, `frontend/src/components/PreferencesView.tsx` (sub-tabs), a new About component, `frontend/src/App.tsx` (comment cleanup), `frontend/src/demoBackend.ts` (mock of the new binding), regenerated `frontend/wailsjs` bindings.
- New files: `releases.json`, plus a Go test guarding version/release-notes consistency.
- No database change, no new dependency, no network access.
