# Tasks

## 1. Closable tab

- [x] 1.1 Add an optional `onClose` to `ViewTab.tsx`: a separate ✕ button (accessible name passed by the caller, here "Fermer la recherche"), sibling of the `role="tab"` button, whose click does not select the tab; without `onClose` the rendering is unchanged — verify with a component test "clicking the ✕ calls onClose but not onClick" and that existing `ViewTab` usages still pass (`npx vitest run`)

## 2. Search lifecycle in App

- [x] 2.1 In `App.tsx`, add `'search'` to `View`, the `vuePrecedente` memory and the effect switching to `'search'` when the query starts or changes while searching, falling back to `vuePrecedente` when `searching` becomes false while on `'search'`; render the "Recherche" tab in `TransverseBar` (after "Priorités", only when `searching`, with `onClose`) — verify in `phase3.test.tsx`: "no Recherche tab without search", "typing réunion shows and selects the Recherche tab", "shortening to ré removes the tab and restores the previous view"
- [x] 2.2 Keep `SearchView` mounted while `searching` and hide it when `view !== 'search'`; remove the "Fermer ⊗" button and the `onClose` prop from `SearchView.tsx`; add an `actif` prop that re-queries when the tab becomes active again, keeping the selection if still present — verify with tests "no Fermer button in the results header" and "after opening the third result and coming back, the third result is still selected"
- [x] 2.3 Remove `setQuery('')` from `ouvrir`; add `fermerRecherche()` used by the tab ✕ and the search-field ✕ — verify with tests "double-clicking a note result opens it in Projets › Notes, keeps the query and the Recherche tab", "switching to Priorités keeps the tab", "a new query replaces the results in the single tab", "closing from the tab on the results returns to Priorités", "closing from Projets keeps Projets and the open note"
- [x] 2.4 Restrict the Escape listener to `view === 'search'` — verify with tests "Escape on the Recherche tab ends the search and restores the previous view" and "Escape in Projets after opening a result keeps the search"; replace the obsolete "Échap ferme la recherche…" and "le bouton Fermer…" tests in `phase3.test.tsx` and verify `npx vitest run` passes in full (including `finitions.test.tsx` Escape tests of the full-screen editor)

## 3. Release

- [x] 3.1 Bump `productVersion` to `1.2.1` in `wails.json` and add a `1.2.1` entry (date of release) in `releases.json` as a single minor addition ("ajout", "mineur") describing the "Recherche" tab and the persistence of the search when opening a result — verify the version checks and the "À propos" tests pass (`npx vitest run aPropos` and `go test ./...`)

## 4. Integration check

- [x] 4.1 Run the app (`wails dev`), search "réunion", open a note then a meeting from the results, come back to the tab between each, toggle 🧠, close with the tab ✕ — verify each step matches the scenarios of `specs/search-results-tab/spec.md`, and `npm run build` type-checks
