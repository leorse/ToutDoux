# Spec Delta

## Purpose

Lets users view and edit rich text content (notes, meeting minutes) at the full size of the application window instead of a narrow column, so pasted images and long content remain readable.

## ADDED Requirements

### Requirement: Full-window read-only view
The rich text editor SHALL provide a control that displays its current content read-only, filling the entire application window, with no formatting toolbar.

#### Scenario: Opening the read-only view
- **WHEN** the user activates the read-only ("eye") control
- **THEN** the editor's content is displayed filling the whole application window, without the formatting toolbar, and cannot be edited

#### Scenario: Closing the read-only view
- **WHEN** the user presses Escape, or activates a close control, while the read-only view is open
- **THEN** the application returns to the normal column layout, with the content unchanged

### Requirement: Full-window editable view
The rich text editor SHALL provide a control that displays its current content in the same window, filling the entire application window, editable with its formatting toolbar visible, with the same editing behavior as the inline editor.

#### Scenario: Opening the editable full-window view
- **WHEN** the user activates the expand control
- **THEN** the editor's content is displayed filling the whole application window, with the formatting toolbar visible, and remains editable

#### Scenario: Editing while in the full-window view
- **WHEN** the user types or applies formatting while the editable full-window view is open
- **THEN** the same change notifications (content updates, autosave triggers) fire as when editing inline

#### Scenario: Closing the editable full-window view
- **WHEN** the user presses Escape, or activates a close control, while the editable full-window view is open
- **THEN** the application returns to the normal column layout, keeping any edits made while full-window

### Requirement: Availability scoped to rich text editors
The full-window read-only and editable views SHALL be available everywhere the rich text editor is used (Notes tab, Meetings tab), and SHALL NOT apply to plain-text fields such as the task description.

#### Scenario: Rich text editor instance
- **WHEN** a rich text editor is displayed in the Notes tab or the Meetings tab
- **THEN** both the read-only and editable full-window controls are available on it
