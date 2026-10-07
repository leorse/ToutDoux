# item-colors Specification

## Purpose
Lets users give a note or a meeting one of a few predefined pastel colours, shown in the lists, so that items can be recognised at a glance.

## Requirements

### Requirement: Colour palette
The application SHALL offer exactly ten predefined pastel colours for notes and meetings, plus the absence of colour. The user SHALL NOT be able to enter a custom colour. Text displayed on any palette colour SHALL remain readable.

#### Scenario: Palette content
- **WHEN** the user opens the colour palette
- **THEN** ten colour swatches and a "no colour" choice are offered, each with an accessible name

### Requirement: Colour button in the editor toolbar
The editor toolbar of a note, and of a meeting's minutes, SHALL show a colour button placed next to the two full-screen buttons. Activating it SHALL open the palette; choosing a swatch SHALL apply it and close the palette; the palette SHALL also close on Escape or on a click outside it, without changing the colour. The palette SHALL indicate the current colour. The colour button SHALL be available in the inline editor and in the full-window editable view, and SHALL NOT be shown where the toolbar is not shown (read-only views).

#### Scenario: Choosing a colour for a note
- **WHEN** the user opens a note, activates the colour button and chooses the blue swatch
- **THEN** the palette closes and the note's row in the notes list has a blue background

#### Scenario: Removing the colour
- **WHEN** a note is blue and the user chooses "no colour" in the palette
- **THEN** the note's row is displayed as an uncoloured note

#### Scenario: Closing without choosing
- **WHEN** the palette is open and the user presses Escape
- **THEN** the palette closes, the colour is unchanged, and a full-window view, if open, stays open

#### Scenario: Position of the button
- **WHEN** the editor toolbar is displayed
- **THEN** the colour button is adjacent to the "full screen (read-only)" and "full screen" buttons

### Requirement: Colour shown in the notes list
A coloured note SHALL be displayed with its colour as the background of its row in the notes list, inside or outside a group. The note open in the editor and the notes of the multi-selection SHALL remain distinguishable from the others whatever their colour. A hidden note SHALL keep its colour, attenuated like the rest of its row.

#### Scenario: Open coloured note
- **WHEN** a green note is open in the editor
- **THEN** its row is green and is marked as the open note

#### Scenario: Coloured note in a group
- **WHEN** a pink note belongs to a group
- **THEN** its row inside the group's frame is pink, and the frame and top band are unchanged

### Requirement: Meeting colour applies to the meeting only
In the Meetings tab, the colour chosen from the editor toolbar SHALL belong to the selected meeting: it SHALL be the background of that meeting's row in the meetings list, and SHALL be the same whichever instance of the meeting is open. Rows of the instances list SHALL NOT be coloured.

#### Scenario: Colouring a meeting
- **WHEN** the user opens an instance of meeting "Comité de pilotage" and chooses yellow
- **THEN** the row "Comité de pilotage" is yellow in the meetings list and the rows of the instances list are unchanged

#### Scenario: Same colour for every instance
- **WHEN** a meeting is yellow and the user opens another of its instances
- **THEN** the palette shows yellow as the current colour

### Requirement: Colour is persistent and display-only
The colour of a note or a meeting SHALL persist across restarts. Changing it SHALL NOT change the item's content, its position in the list, its group, its hidden state, the project counters or the search results, and SHALL NOT discard text being typed in the editor. Items existing before this feature SHALL have no colour.

#### Scenario: Colour survives a restart
- **WHEN** the user colours a note and restarts the application
- **THEN** the note is displayed with the same colour

#### Scenario: Typing is not lost
- **WHEN** the user types in a note and immediately chooses a colour
- **THEN** the typed text is saved and the colour is applied

#### Scenario: Upgrade
- **WHEN** a database created with an earlier version is opened
- **THEN** every note and meeting is displayed without colour
