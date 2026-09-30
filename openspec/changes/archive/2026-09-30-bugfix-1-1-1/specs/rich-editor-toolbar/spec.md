# Spec Delta

## Purpose

Makes the formatting toolbar of the rich text editor (notes, meeting minutes) show, in real time, which formatting applies at the cursor or selection, so the user can see the current formatting without typing.

## ADDED Requirements

### Requirement: Toolbar reflects formatting at the cursor
Each formatting button of the rich editor toolbar (Gras, Italique, Barré, Titre, Liste à puces, Liste numérotée, Bloc de code, Lien) SHALL be shown as pressed exactly when that formatting applies at the current cursor position or selection. The pressed state SHALL update as soon as the cursor or selection moves, whether by mouse click, keyboard navigation or selection, without the user having to type or change the content. This applies in the inline editor and in the editable full-window view.

#### Scenario: Cursor moved into bold text
- **WHEN** the cursor is in plain text and the user clicks inside a bold word, without typing
- **THEN** the "Gras" button is shown as pressed

#### Scenario: Cursor moved out of bold text
- **WHEN** the cursor is inside a bold word and the user moves it into plain text with the arrow keys, without typing
- **THEN** the "Gras" button is no longer shown as pressed

#### Scenario: Block formatting follows the cursor
- **WHEN** the user moves the cursor from a paragraph into a bullet list item, without typing
- **THEN** the "Liste à puces" button is shown as pressed, and it stops being pressed when the cursor goes back to the paragraph

#### Scenario: Several formats at once
- **WHEN** the cursor is placed in text that is both bold and italic
- **THEN** both the "Gras" and "Italique" buttons are shown as pressed, and the other inline format buttons are not

#### Scenario: Editable full-window view
- **WHEN** the editable full-window view is open and the user moves the cursor into italic text
- **THEN** the "Italique" button of its toolbar is shown as pressed

#### Scenario: Pressed state is exposed to assistive technology
- **WHEN** a formatting button is shown as pressed
- **THEN** it also reports itself as pressed to assistive technology, and reports not pressed otherwise
