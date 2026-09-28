# Tasks

## 1. Version source and window title

- [x] 1.1 Add an `info` block to `wails.json` (`productVersion` "1.1.0" only, leaving `productName` at its default to keep the installer's uninstall key stable) and verify the file is valid JSON and the installer still reads `INFO_PRODUCTVERSION` (`wails build -nsis` metadata, or inspect `build/windows/installer` usage)
- [x] 1.2 Add `app_about.go` in package `main`: embed `wails.json`, parse `info.productVersion`, expose `windowTitle(version)` returning `Tout Doux (<version>)`; add `app_about_test.go` covering the title format, a valid config, and a malformed or missing version, and verify `go test .` passes
- [x] 1.3 Use the embedded version in `main.go` to set `Title`, and remove the prototype-era comment about the system title bar there; verify `go build` succeeds and the title test still passes

## 2. Release notes data and guard

- [x] 2.1 Create `releases.json` at the repo root (newest first) with a `1.1.0` entry describing this change (version shown in the title, "À propos" tab) and a `1.0.0` baseline entry, each with `version`, `date`, `changes[]` and, where useful, `important`; verify it parses
- [x] 2.2 Embed `releases.json` and add `GetAppInfo()` on `App` returning `{version, releases[]}` with json tags; verify with a Go test that the call returns the embedded data
- [x] 2.3 Add the guard check to `app_about_test.go` (version format `X.Y.Z`, ISO dates, non-empty `changes`, `releases[0].version == productVersion`) as a plain function over parsed values, plus a test proving it fails on a mismatched fake release list; verify `go test .` passes and the mismatch test fails as intended when the check is bypassed

## 3. Frontend

- [x] 3.1 Regenerate the Wails bindings (`wails generate module`) and add `GetAppInfo` to `frontend/src/demoBackend.ts`; verify `tsc --noEmit` passes in `frontend/`
- [x] 3.2 Add the pure inline-markup function (`**bold**`, `*italic*`, everything else literal text, no HTML) with unit tests for bold, italic, both together, unmatched markers, and an HTML tag rendered as text; verify `npm test` passes
- [x] 3.3 Add an About component showing the current version and the release history (version, date, bullet list, highlighted `important` notice when present) built with the markup function; verify with component tests using `setBackend` (several changes, with and without `important`, order preserved)
- [x] 3.4 Split `PreferencesView` into "Préférences" (default, unchanged content) and "À propos" sub-tabs with `role="tablist"`; verify with tests for the default tab and switching, and that the existing Preferences tests still pass
- [x] 3.5 Remove the prototype-era doc comment about the missing title bar in `frontend/src/App.tsx` (or reduce it to one line on the native title); verify `npm test` passes

## 4. Integration

- [x] 4.1 Run `go test ./...` and `npm test` in `frontend/`, then launch the app (`wails dev`) and confirm by hand that the window title reads `Tout Doux (1.1.0)`, the Preferences view has both sub-tabs, and "À propos" shows the version and both releases; also confirm bumping `productVersion` alone makes the guard test fail
