# Design

## Context

The app is a Wails v2 desktop app (Go backend in package `main` at the repo root, React frontend in `frontend/`). `main.go` sets `Title: "Tout Doux"` on `options.App`; `frontend/src/App.tsx` deliberately renders no title bar. `PreferencesView` is a single page rendered by `App` when `view === 'preferences'`. `wails.json` has no `info` block, while the Windows installer (`build/windows/installer`) already reads `INFO_PRODUCTVERSION` from it. All backend calls go through `frontend/src/api.ts`, typed from the generated `wailsjs` bindings, with `demoBackend.ts` as the browser-mode mock. See proposal.md - Why.

## Goals / Non-Goals

**Goals:**
- One file to edit to change the version, feeding title, About tab and installer (a plain `wails build` does not write the version into the exe's file properties, only the NSIS installer reads it).
- A release history that is structured, lightly formatted, and guarded against drifting from the version.

**Non-Goals:**
- No update check, no network access (the Preferences page states the app never touches the network).
- No Markdown library, no rich text engine for release notes.
- No automated version bumping or release tooling.

## Decisions

- **`wails.json` `info.productVersion` is the source of truth**, embedded with `//go:embed wails.json` in package `main` (the file sits next to `main.go`, so the embed is legal). Alternatives: `-ldflags -X main.version=...` (a forgotten flag silently yields "dev", and the installer would still need the value elsewhere) and a dedicated `VERSION` file (duplicates what Wails already reads). Keeping `X.Y.Z` three-part is required: the installer appends `.0` to build a four-part Windows version.
- **Title built before `wails.Run`**: `windowTitle(version)` returns `Tout Doux (<version>)`, computed from the embedded config in `main()`. No runtime call needed for the title, and the function is trivially unit-tested.
- **One binding, `GetAppInfo()`**, returns `{version, releases[]}`. Alternative: two bindings (`GetVersion`, `GetReleaseNotes`) — rejected, the About tab needs both together and one call keeps the mock small. Struct types carry json tags so Wails generates the TS model.
- **`releases.json` at the repo root, newest release first**, embedded next to `wails.json`. Shape: `{ "releases": [ { "version", "date", "important"?, "changes": [string] } ] }`. Structure (list, notice) comes from the JSON shape; inline style comes from a tiny markup. Alternatives: HTML strings rendered with `dangerouslySetInnerHTML` (works, but injects raw HTML and forces hand-written tags in every entry) and Markdown with a library (a new dependency for a very small need).
- **Inline markup `**bold**` / `*italic*`** parsed by a small pure function (`formaterTexte`) into React nodes, never into an HTML string. Unmatched markers and any `<`/`>` stay literal text, so there is no injection surface. The function is unit-tested on its own.
- **Guard test in Go** (`app_about_test.go`): parses the embedded files and asserts version format `X.Y.Z`, non-empty `changes`, ISO `date`, and `releases[0].version == productVersion`. The check itself is a plain function over parsed values so its failure case is testable with fake data.
- **Sub-tabs in `PreferencesView`** with local state (`'preferences' | 'about'`), reusing the tab styling of the project sub-tabs (`ViewTab`-like buttons with `role="tablist"`). The existing content moves unchanged into the first tab.

## Risks / Trade-offs

- [`wails.json` is also read by the Wails CLI; a malformed edit breaks the build] → the guard test parses it, so `go test` catches it before `wails build`.
- [Version and release notes are two files that can drift] → the guard test fails the build when the newest release differs from the version.
- [Adding an `info` block changes Windows metadata and the installer's displayed version] → intended. Only `productVersion` is set: the installer builds its uninstall registry key from company + product name, so changing `productName` would stop an existing install from being recognised as an upgrade.
- [Wails bindings are generated] → regenerate them (`wails generate module`) and commit the result, otherwise the typed `api` proxy has no `GetAppInfo`.

## Open Questions

- The content of the `1.0.0` entry (what shipped first) is a placeholder written from the git history; the author can enrich it later without any code change.
