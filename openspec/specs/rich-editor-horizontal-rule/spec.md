# rich-editor-horizontal-rule Specification

## Purpose

Offers, in the formatting toolbar of the rich text editor, a button inserting a horizontal separator line, which could so far only be created by typing `---` or `***`.

## Requirements

### Requirement: Separator button
The formatting toolbar of the rich text editor SHALL offer a button showing only an icon (no text label), with the accessible name and tooltip "Ligne de séparation", wherever the formatting toolbar is shown. Activating it SHALL insert a horizontal separator line at the cursor, or in place of the selection when there is one, identical to the line produced by typing `---`. The cursor SHALL then be in the block following the line, a new empty paragraph being created when the line ends the content. The button SHALL never be shown as pressed. The existing `---` and `***` shortcuts SHALL keep working.

#### Scenario: Inserting a separator
- **WHEN** the cursor is at the end of the paragraph "Partie 1" and the user clicks "Ligne de séparation", then types "Partie 2"
- **THEN** the content is the paragraph "Partie 1", a separator line, then the paragraph "Partie 2", and it is saved as such

#### Scenario: Never pressed
- **WHEN** the cursor is placed just after a separator line
- **THEN** the "Ligne de séparation" button is not shown as pressed

#### Scenario: Read-only
- **WHEN** the editor is read-only
- **THEN** no "Ligne de séparation" button is shown
