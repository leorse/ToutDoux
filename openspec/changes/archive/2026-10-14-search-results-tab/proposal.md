# Proposal

## Why

Today the search results replace the whole main area, and opening a result (double-click or "Ouvrir") clears the search: to look at a second result the user has to type the query again. The results should stay at hand while the user visits them, the way an open tab stays open.

## What Changes

- **"Recherche" tab.** Search results are shown in a new view tab, placed after "Projets" and "Priorités" in the transverse bar. The tab exists only while a search is active; it appears (and becomes the active view) as soon as a search is launched, and disappears when the search ends.
- **Same results display.** Inside the tab, the results keep their current form: count, 🧠 badge, results list on the left, preview on the right with highlighted excerpt and "Ouvrir" button, semantic scores.
- **Opening a result no longer ends the search.** Opening a result navigates to it exactly as today (Projets view, project, native sub-tab, target selected), but the query is kept and the "Recherche" tab stays, with its results and its selected result. Coming back to the tab shows the results as they were left.
- **Display persistence only.** The search survives view switches, but not a restart: nothing is stored in the database. It ends when the user closes the tab, launches a new search (which replaces it) or quits the application.
- **Closing lives on the tab.** The tab carries a ✕ close button (icon only, no text). **BREAKING** (UI): the "Fermer ⊗" button of the results header is removed.
- **Escape** closes the search only while the "Recherche" tab is the active view; elsewhere it no longer clears the search (so pressing Escape in an editor after opening a result does not lose it).
- Application version becomes `1.2.1`, with a `1.2.1` entry in the release notes.

## Capabilities

### New Capabilities
- `search-results-tab`: where the global search results are displayed (a closable "Recherche" view tab), when that tab exists, how it persists across navigation, and how it is closed.

### Modified Capabilities
(none — no existing spec covers the search; the search engine itself, keyword and semantic, is unchanged)

## Impact

- Affected code: `frontend/src/App.tsx` (view state gains `'search'`, search lifecycle separated from navigation, `ouvrir` no longer clears the query, Escape handling, `TransverseBar` renders the extra tab), `frontend/src/components/ViewTab.tsx` (optional close button), `frontend/src/components/SearchView.tsx` (header button removed, view kept mounted so its selection survives), `frontend/src/phase3.test.tsx` (search tests rewritten), `wails.json` (`productVersion`), `releases.json`.
- No backend, binding, database or dependency change. `SearchGlobal` / `SearchSemantic` are untouched.
