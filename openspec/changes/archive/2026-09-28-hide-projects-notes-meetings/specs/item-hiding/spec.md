# Spec Delta

## Purpose

Lets users hide projects, notes and meetings that are no longer current so they stop cluttering their lists, without deleting anything and without changing what the rest of the application computes from them.

## ADDED Requirements

### Requirement: Hiding and revealing items
Projects, notes and meetings SHALL be hideable and revealable from their right-click menu, through a "Cacher" entry on a visible item and a "Réafficher" entry on a hidden one. The hidden state SHALL persist across application restarts. The locked "Transverse / Divers" project SHALL NOT be hideable: its "Cacher" entry SHALL be present but disabled.

#### Scenario: Hiding a note
- **WHEN** the user right-clicks a note and chooses "Cacher"
- **THEN** the note is marked hidden and remains so after restarting the application

#### Scenario: Revealing a hidden meeting
- **WHEN** hidden meetings are displayed and the user right-clicks one and chooses "Réafficher"
- **THEN** the meeting is no longer hidden

#### Scenario: Locked project
- **WHEN** the user right-clicks "Transverse / Divers"
- **THEN** the "Cacher" entry is disabled

### Requirement: Hiding is display-only
Hiding an item SHALL NOT change what is computed from it. Tasks of a hidden project SHALL keep appearing in the Priorités view, in the system tray menu, icon and notifications, and in search results, exactly as if the project were visible. Hiding SHALL NOT delete or modify any content.

#### Scenario: Priorities of a hidden project
- **WHEN** a project holding a critical task is hidden
- **THEN** that task still appears in the Priorités view and still counts for the system tray

#### Scenario: Search finds hidden items
- **WHEN** the user searches for text contained in a hidden note
- **THEN** the note appears in the search results

### Requirement: Per-list visibility toggle
The projects list, the notes list and the meetings list SHALL each have their own icon-only toggle, with no text label, controlling whether hidden items are shown in that list. With the eye open (the default), hidden items are not shown. With the eye closed, all items are shown and hidden ones are rendered semi-transparent. Each toggle SHALL keep its state when the user switches between projects, and SHALL be reset to open when the application restarts. Toggling one list SHALL NOT affect the other two.

#### Scenario: Default state
- **WHEN** the application starts
- **THEN** all three toggles show an open eye and no hidden item is listed

#### Scenario: Showing hidden projects
- **WHEN** the user closes the eye on the projects list
- **THEN** hidden projects appear in the list, semi-transparent, alongside visible ones, and the notes and meetings lists are unchanged

#### Scenario: State kept across projects
- **WHEN** the notes eye is closed and the user selects another project
- **THEN** the notes list of the other project also shows its hidden notes

#### Scenario: Accessible label
- **WHEN** a toggle is inspected by assistive technology
- **THEN** it exposes a text label describing its action, even though none is displayed

### Requirement: Hidden-aware project counters
The project list SHALL display for each project its notes and meetings counters as `N(J)` and `M(K)`, where N and M are the numbers of visible notes and meetings and J and K the numbers of hidden ones. The `(J)` and `(K)` parts SHALL be shown only when greater than zero. The counters SHALL NOT depend on the state of the visibility toggles.

#### Scenario: Hiding a note updates counters
- **WHEN** a project shows 4 notes with none hidden and the user hides one of them
- **THEN** its counter reads 3 notes with 1 hidden, that is `3(1)`

#### Scenario: No hidden items
- **WHEN** a project has no hidden notes
- **THEN** its notes counter shows only N, without parentheses

#### Scenario: Toggle does not change counters
- **WHEN** the user closes or opens the notes eye
- **THEN** the counters stay the same

### Requirement: Selection stays on a displayed item
When the selected item stops being displayed, because it was just hidden or because the eye opened on it, the selection SHALL move to the first displayed item of that list, or to none if the list is empty. When the user opens a hidden project or meeting through the search or the Priorités view, that item SHALL be revealed (the relevant eye closes) instead of being replaced by another selection.

#### Scenario: Hiding the selected project
- **WHEN** the selected project is hidden while the projects eye is open
- **THEN** the first displayed project becomes selected

#### Scenario: Opening the eye on a hidden selection
- **WHEN** a hidden project is selected with the eye closed and the user opens the eye
- **THEN** the first displayed project becomes selected

#### Scenario: Opening a task of a hidden project from Priorités
- **WHEN** the user opens a task belonging to a hidden project from the Priorités view while the projects eye is open
- **THEN** the projects eye closes, the hidden project appears semi-transparent, and it is the selected project

#### Scenario: Opening a hidden meeting from search
- **WHEN** the user opens a hidden meeting from the search results while the meetings eye is open
- **THEN** the meetings eye closes and that meeting is displayed and selected
