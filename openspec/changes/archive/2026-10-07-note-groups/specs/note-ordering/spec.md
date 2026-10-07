# Spec Delta

## Purpose

Lets users arrange the notes list of a project in the order they choose, by drag and drop, and keeps that order stable over time instead of reshuffling the list whenever a note is edited.

## ADDED Requirements

### Requirement: User-defined order of the notes list
The notes list of a project SHALL be displayed in a user-defined order that persists across application restarts and project switches. Editing the title or the content of a note, hiding it or revealing it SHALL NOT change its position. The order of one project SHALL be independent of the others.

#### Scenario: Editing does not move a note
- **WHEN** the user edits the content of the third note of the list and it is saved
- **THEN** that note is still third in the list

#### Scenario: Order kept after restart
- **WHEN** the user reorders the notes and restarts the application
- **THEN** the notes list shows the same order as before the restart

### Requirement: Existing order preserved on upgrade
When the application is upgraded from a version without manual ordering, each project's notes SHALL initially appear in the order they had before the upgrade, that is most recently modified first.

#### Scenario: Upgrade from 1.1.1
- **WHEN** a database created with version 1.1.1 holds notes A, B and C, displayed in that order, and the application is upgraded
- **THEN** the notes list shows A, B, C in that order, and none of them is in a group

### Requirement: Position of a new note
A newly created note SHALL be placed at the top of its project's notes list, outside any group, and SHALL become the note shown in the editor.

#### Scenario: Creating a note
- **WHEN** the user creates a note in a project that already holds notes and groups
- **THEN** the new note is the first item of the list, is not in any group, and is opened in the editor

### Requirement: Reordering notes by drag and drop
The user SHALL be able to move a note by dragging it and dropping it on another note: dropped on the upper half of the target, the note SHALL be placed immediately before it; dropped on the lower half, immediately after it. Dropping a note on the empty area below the list SHALL place it last. While dragging, the list SHALL show where the note will be inserted. Dropping a note at the place it already occupies, or releasing it outside the list, SHALL change nothing. A plain click on a note SHALL still select it without moving it.

#### Scenario: Moving a note up
- **WHEN** the list shows A, B, C and the user drags C onto the upper half of A
- **THEN** the list shows C, A, B and that order is persisted

#### Scenario: Moving a note to the end
- **WHEN** the list shows A, B, C and the user drags A onto the empty area below the list
- **THEN** the list shows B, C, A

#### Scenario: Insertion indicator
- **WHEN** the user drags a note over the lower half of another note
- **THEN** an insertion mark is displayed just below that note

#### Scenario: Drop cancelled
- **WHEN** the user drags a note and releases it outside the notes list
- **THEN** the order is unchanged

#### Scenario: Reordering does not lose pending edits
- **WHEN** the user types in the editor of a note and immediately drags another note to a new position
- **THEN** the typed text is saved and the notes appear in the new order

### Requirement: Reordering with hidden notes filtered out
When hidden notes are not displayed, moving a note SHALL place it relative to the note it was dropped on, and the hidden notes SHALL keep their own relative order.

#### Scenario: Hidden note keeps its place
- **WHEN** the full order is A, H, B, C with H hidden and not displayed, and the user drags C onto the upper half of B
- **THEN** the full order becomes A, H, C, B
