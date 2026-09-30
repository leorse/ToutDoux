# meeting-minutes-editing Specification

## Purpose

Ensures that editing the minutes of a meeting instance is not interrupted by its own autosave or by other data refreshes: the selected meeting and instance stay the same and the user keeps editing where they were.

## Requirements

### Requirement: Autosave keeps the meeting selection
Saving the minutes of a meeting instance, whether triggered by inactivity or by leaving the editor, SHALL NOT change the selected meeting or the selected instance, whatever the position of that meeting in the list. The saved content SHALL belong to the instance being edited.

#### Scenario: Autosave after inactivity on a meeting that is not first
- **WHEN** the user edits the minutes of an instance of the third meeting of the list and stops typing long enough for the autosave to run
- **THEN** after the save the third meeting and the same instance are still selected, and the editor still shows the text just typed

#### Scenario: Autosave on an older instance
- **WHEN** the user edits the minutes of an instance that is not the most recent one of its meeting, and the autosave runs
- **THEN** that same instance stays selected, not the most recent one

#### Scenario: Save content goes to the edited instance
- **WHEN** the autosave runs for the minutes of an instance
- **THEN** the saved minutes are stored on that instance, and no other instance or meeting is modified

### Requirement: Editing is not interrupted by data refreshes
While the user is editing the minutes of a meeting instance, a save or any other refresh of application data (such as the project sidebar counters updating) SHALL NOT take the keyboard focus out of the editor, move the cursor, or replace the content being edited.

#### Scenario: Typing continues across an autosave
- **WHEN** the user keeps typing in the minutes while an autosave runs
- **THEN** the focus stays in the editor, the cursor stays where it was, and the keystrokes typed during and after the save are kept and appear in the next save

#### Scenario: Refresh caused by another change
- **WHEN** application data is refreshed while the user is editing the minutes of an instance
- **THEN** the selected meeting and instance are unchanged and the focus stays in the editor
