# Design

## Context

See proposal.md for motivation. Current state relevant to the approach:

- `App.tsx` holds `view: 'projects' | 'priorities' | 'preferences'` and `query`. "Search active" is derived (`searching = query.trim().length > 2`) and, when true, `<SearchView>` replaces whatever `view` would render. The search has no state of its own beyond `query`.
- `ouvrir(cible)` (used by search, Priorités and the tray `open-task` event) starts with `setQuery('')`: that is what ends the search on navigation today.
- A global `keydown` listener clears the query on Escape, whatever is focused. `RichEditor` and `ContextMenu` also use Escape (full screen, palette, menu).
- `SearchView` owns its results, selected result (`selectedId`), semantic flag of the displayed results and errors; it re-queries (debounced) when `query` or `semantique` change, and keeps the selection when the selected id is still in the new results. Its header has the "Fermer ⊗" button calling `onClose`.
- `ViewTab` is a plain `role="tab"` button; `TransverseBar` renders "Projets" and "Priorités".
- Search tests live in `phase3.test.tsx` ("Recherche globale (§2.9)") and run against `demoBackend`.

## Goals / Non-Goals

**Goals:**
- Navigation and search lifecycle become independent: only an explicit close or a query change touches the search.
- Returning to the tab restores selection and split width without extra bookkeeping.

**Non-Goals:**
- Several search tabs, search history, or keeping a previous search while typing a new one.
- Persisting the search across restarts.
- Any change to the search engines, ranking, excerpts, or the 3-character threshold.
- Changing how a single result is selected (click) versus opened (double-click / "Ouvrir").

## Decisions

**1. `'search'` becomes a value of `View`; "active search" stays derived from `query`.**
`searching` keeps its definition, so the tab exists iff `searching`. `view === 'search'` means the tab is the active one. An effect on the transition `searching: false → true` (and on any query change while searching) sets `view` to `'search'` and records `vuePrecedente` — the view active just before — only on the `false → true` transition. When `searching` becomes false while `view === 'search'`, `view` falls back to `vuePrecedente`.
*Alternative:* a separate `searchOpen` boolean decoupled from `query`. Rejected: two sources of truth for one fact, and the spec ties the tab to the field content.

**2. `SearchView` stays mounted while the search is active, hidden when its tab is not selected.**
Rendered as a sibling of the other views, wrapped in a container with `hidden` when `view !== 'search'`, and unmounted when `searching` becomes false. Its internal state (results, `selectedId`, `Split` width) therefore survives navigation for free, and unmounting on close guarantees the next search starts clean.
*Alternative:* lift `results`/`selectedId` into `App`. Rejected: more props and state in an already large component for no behavioural gain. The other views (Projets split, Priorités) keep their current mount/unmount behaviour; only the search view is kept alive.
Because `SearchView` stays mounted, it keeps re-querying on `query`/`semantique` changes as today; it also re-queries when its tab becomes active again (see risks).

**3. `ouvrir` no longer clears the query.** It still sets `view = 'projects'`, project, sub-tab and target. This also applies to Priorités and the tray event, which is harmless: they did not need to end a search, they only did so because `ouvrir` was shared.

**4. One `fermerRecherche()` in `App`.** Clears `query`; decision 1 handles the view fallback. Used by the tab ✕, the search-field ✕ and Escape. The "Fermer ⊗" button and `onClose` prop of `SearchView` are removed.

**5. `ViewTab` gains an optional `onClose`.** When given, the tab renders a small ✕ `<button aria-label="Fermer la recherche">` next to the label, inside a wrapper, with `stopPropagation` so closing does not select the tab. The tab's own accessible name stays "Recherche" (the ✕ is a separate button, not nested inside the `role="tab"` button, to keep valid HTML).

**6. Escape is filtered by the active view.** The global listener reads the current view (through a ref or the effect's dependency) and calls `fermerRecherche()` only when `view === 'search'`. This avoids an Escape meant for an editor or menu in Projets wiping the search.

## Risks / Trade-offs

- [Results shown after coming back may be stale if the user edited or deleted the opened item] → Re-query when the tab becomes active again (`SearchView` receives an `actif` prop and refreshes on `false → true`); the existing logic already keeps the selection if the id is still present and falls back to the first result otherwise.
- [Hidden-but-mounted `SearchView` keeps its timers/effects alive] → They only run on `query`/`semantique`/`actif` changes; cost is negligible.
- [Tests and users relying on "Fermer ⊗" or on Escape anywhere] → Tests are rewritten; the change is called out in the 1.2.1 release notes.

## Migration Plan

Frontend only, no data migration. Rollback is a plain revert.
