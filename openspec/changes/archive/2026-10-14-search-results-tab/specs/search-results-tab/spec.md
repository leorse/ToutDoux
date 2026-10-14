# Spec Delta

## Purpose

Shows the global search results in a dedicated, closable "Recherche" view tab that stays available while the user navigates to the results, until the search is closed, replaced or the application quits.

## ADDED Requirements

### Requirement: Search tab existence
The transverse bar SHALL show a "Recherche" view tab after the "Projets" and "Priorités" tabs if and only if a search is active. A search SHALL be active when the search field holds at least 3 non-blank characters. When no search is active, the "Recherche" tab SHALL NOT be present at all (neither hidden nor disabled).

#### Scenario: No search
- **WHEN** the application starts, or the search field holds fewer than 3 characters
- **THEN** the view tabs are "Projets" and "Priorités" only

#### Scenario: Search launched
- **WHEN** the user types "réunion" in the search field while the "Projets" view is active
- **THEN** a "Recherche" tab appears after "Priorités" and becomes the active view, showing the results for "réunion"

#### Scenario: Query shortened below the minimum
- **WHEN** a search for "réunion" is active and the user erases the field down to "ré"
- **THEN** the "Recherche" tab disappears and the view that was active before the search is shown again

### Requirement: Results display in the search tab
When the "Recherche" tab is the active view, it SHALL occupy the main area below the transverse bar and SHALL display the results in the same form as before this change: the result count, the semantic-mode badge when the semantic mode is on, the results list, and the preview of the selected result with its highlighted excerpt and its "Ouvrir" action. The results header SHALL NOT contain a close button.

#### Scenario: Same content as before
- **WHEN** a search for "réunion" is active and the "Recherche" tab is selected
- **THEN** the result count, the results list and the preview of the selected result are shown, and the project sidebar and task tree are not

#### Scenario: No close button in the header
- **WHEN** the "Recherche" tab is shown
- **THEN** no "Fermer" button is present in the results header

### Requirement: Search persists across navigation
Opening a result (double-click on a result or the "Ouvrir" action of the preview) SHALL navigate to it as before this change: the "Projets" view becomes active with the result's project, native sub-tab and item selected. Opening a result SHALL NOT clear the search field nor end the search: the "Recherche" tab SHALL remain present. Switching to another view tab SHALL NOT end the search either. When the user selects the "Recherche" tab again, it SHALL show the same query, the same mode, and the same selected result as when it was left, provided that result is still part of the results.

#### Scenario: Opening a note from the results
- **WHEN** a search for "réunion" is active and the user double-clicks a note result
- **THEN** the "Projets" view is active with that note open in the "Notes" sub-tab, the search field still holds "réunion" and the "Recherche" tab is still present

#### Scenario: Coming back to the results
- **WHEN** the user had selected the third result, opened it, then selects the "Recherche" tab again
- **THEN** the results for the same query are shown and the third result is still selected

#### Scenario: Switching to Priorités
- **WHEN** a search is active and the user selects the "Priorités" tab
- **THEN** the "Priorités" view is shown and the "Recherche" tab is still present

### Requirement: Display persistence only
The search SHALL NOT be stored in the database or in any persistent storage. It SHALL end when the user closes it, and SHALL be replaced when the user changes the query (a new search). After the application is restarted, no search SHALL be active.

#### Scenario: New search replaces the previous one
- **WHEN** a search for "réunion" is active, the user opened a result, then changes the query to "budget"
- **THEN** a single "Recherche" tab is present, it becomes the active view and shows the results for "budget"

#### Scenario: Restart
- **WHEN** a search is active and the user quits and restarts the application
- **THEN** the search field is empty and no "Recherche" tab is present

### Requirement: Closing the search from the tab
The "Recherche" tab SHALL carry a close control showing only a cross (✕), with no text, and with an accessible name "Fermer la recherche". Activating it SHALL end the search: the search field SHALL be emptied and the tab SHALL disappear. If the "Recherche" tab was the active view, the view that was active before the search SHALL be shown; otherwise the active view SHALL stay unchanged. Activating the close control SHALL NOT first select the "Recherche" tab. The cross of the search field SHALL have the same effect.

#### Scenario: Closing while on the results
- **WHEN** the user was on "Priorités", launched a search, and clicks the ✕ of the "Recherche" tab
- **THEN** the tab disappears, the search field is empty and the "Priorités" view is shown

#### Scenario: Closing from another view
- **WHEN** a search is active, the "Projets" view is shown with a note open, and the user clicks the ✕ of the "Recherche" tab
- **THEN** the tab disappears, the search field is empty and the "Projets" view stays shown with the same note open

### Requirement: Escape closes the search only on its tab
Pressing Escape SHALL close the search, as the tab's close control does, only when the "Recherche" tab is the active view. When another view is active, Escape SHALL NOT end the search.

#### Scenario: Escape on the results
- **WHEN** the "Recherche" tab is the active view and the user presses Escape
- **THEN** the search ends and the view that was active before the search is shown

#### Scenario: Escape elsewhere
- **WHEN** a search is active, the user opened a result and presses Escape in the "Projets" view
- **THEN** the search field still holds the query and the "Recherche" tab is still present
