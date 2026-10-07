# Spec Delta

## Purpose

Lets users gather related notes of a project into named groups, shown as framed blocks in the notes list, and manage those groups by drag and drop and from the right-click menu.

## ADDED Requirements

### Requirement: Multi-selection of notes
The notes list SHALL support selecting several notes: a plain click SHALL select that note only and open it in the editor; Ctrl+click SHALL add the note to the selection or remove it from it; Shift+click SHALL select the range of displayed notes between the note open in the editor and the clicked one. The editor SHALL keep showing a single note, and Ctrl+click and Shift+click SHALL NOT change which note is open. Selected notes SHALL be visually distinguishable from unselected ones, and the note open in the editor from the other selected notes. The multi-selection SHALL be cleared when the user switches project.

#### Scenario: Adding to the selection
- **WHEN** note A is open and the user Ctrl+clicks note C
- **THEN** A and C are selected and the editor still shows A

#### Scenario: Range selection
- **WHEN** the list shows A, B, C, D, note A is open and the user Shift+clicks C
- **THEN** A, B and C are selected and D is not

#### Scenario: Plain click resets the selection
- **WHEN** A, B and C are selected and the user clicks D without modifier
- **THEN** only D is selected and D is open in the editor

### Requirement: Grouping the selected notes
The right-click menu of a note SHALL offer a "Regrouper" entry. Right-clicking a note that is part of the selection SHALL apply it to all selected notes; right-clicking a note outside the selection SHALL first make that note the only selected one. Choosing "Regrouper" SHALL open a popup asking for the group name. On confirmation with a non-blank name, a new group with that name (trimmed) SHALL be created, containing the selected notes in their current list order, and placed where the first of them was. Notes that already belonged to a group SHALL leave it. Cancelling the popup or confirming a blank name SHALL create nothing. Two groups MAY carry the same name. Groups SHALL persist across restarts.

#### Scenario: Grouping three notes
- **WHEN** the list shows A, B, C, D, the user selects A, C and D, right-clicks one of them, chooses "Regrouper" and enters "Sprint 12"
- **THEN** the list shows a group "Sprint 12" containing A, C, D in that order, followed by B

#### Scenario: Grouping a single note
- **WHEN** nothing but note A is selected and the user right-clicks note B and chooses "Regrouper", then enters "Idées"
- **THEN** a group "Idées" containing only B is created at B's position

#### Scenario: Cancelled popup
- **WHEN** the user chooses "Regrouper" and cancels the popup, or confirms a name made only of spaces
- **THEN** no group is created and the list is unchanged

#### Scenario: Regrouping notes taken from another group
- **WHEN** group G1 contains A and B, and the user selects B and the ungrouped note C and groups them as "G2"
- **THEN** G1 contains only A and G2 contains B and C

#### Scenario: Groups survive a restart
- **WHEN** the user creates a group and restarts the application
- **THEN** the group is displayed with the same name, the same notes and the same position

### Requirement: Group rendering
A group SHALL be rendered in the notes list as a frame whose border is thicker than the separation between ordinary rows, topped by a band thicker than the other sides of the frame that displays the group name. The notes of the group SHALL be listed inside the frame, in their order within the group. A long group name SHALL be truncated rather than widen the list. Groups SHALL NOT contain other groups.

#### Scenario: Group frame
- **WHEN** a project has a group "Sprint 12" containing two notes
- **THEN** the notes list shows a framed block with "Sprint 12" in its top band and the two notes inside the frame

#### Scenario: Accessible name
- **WHEN** a group is inspected by assistive technology
- **THEN** it is exposed as a group labelled with the group name

### Requirement: Adding a note to a group by drag and drop
Dropping a note on a group SHALL add it to that group: dropped on the top band, the note SHALL become the last note of the group; dropped on a note of the group, it SHALL be placed immediately before or after that note, following the same upper-half / lower-half rule as in the rest of the list. A note that belonged to another group SHALL leave it. While a note is dragged over a group, the group SHALL be highlighted as the drop target.

#### Scenario: Dropping on the top band
- **WHEN** group G contains A and B and the user drags the ungrouped note C onto G's top band
- **THEN** G contains A, B, C and C no longer appears outside the group

#### Scenario: Dropping at a position inside the group
- **WHEN** group G contains A and B and the user drags the ungrouped note C onto the upper half of B
- **THEN** G contains A, C, B

#### Scenario: Moving between groups
- **WHEN** note A belongs to group G1 and the user drags it onto the top band of group G2
- **THEN** A belongs to G2 and no longer to G1

#### Scenario: Reordering inside a group
- **WHEN** group G contains A, B, C and the user drags C onto the upper half of A
- **THEN** G contains C, A, B

### Requirement: Removing a note from a group by drag and drop
Dragging a note of a group and dropping it outside the group's frame — on an ungrouped note or on the empty area of the list — SHALL remove it from the group and place it at the drop position in the list. The note itself SHALL NOT be modified or deleted.

#### Scenario: Dragging a note out
- **WHEN** the list shows group G containing A and B, followed by the ungrouped note C, and the user drags B onto the lower half of C
- **THEN** G contains only A and the list shows G, C, B

#### Scenario: Dragging onto the empty area
- **WHEN** the user drags a note of a group onto the empty area below the list
- **THEN** the note is removed from the group and becomes the last item of the list

### Requirement: Empty groups disappear
A group SHALL exist only while it contains at least one note. When its last note is moved out of it or deleted, the group SHALL be removed.

#### Scenario: Last note dragged out
- **WHEN** group G contains only note A and the user drags A out of the group
- **THEN** G is no longer displayed and A is an ungrouped note

#### Scenario: Last note deleted
- **WHEN** group G contains only note A and the user deletes A
- **THEN** G no longer exists

### Requirement: Moving a group
A group SHALL be movable as a whole by dragging its top band, and only from there: dragging a note of the group SHALL move that note alone. A dragged group SHALL be droppable before or after any ungrouped note or any other group, and on the empty area below the list to become last. Its notes SHALL keep their order inside it. A group SHALL NOT be droppable inside another group: released over another group, it SHALL be placed before or after that group.

#### Scenario: Moving a group below a note
- **WHEN** the list shows group G, then notes A and B, and the user drags G by its top band onto the lower half of B
- **THEN** the list shows A, B, G, and G still contains the same notes in the same order

#### Scenario: Group dropped over another group
- **WHEN** the user drags group G1 by its top band and releases it over the lower half of group G2
- **THEN** G1 is placed immediately after G2 and G2's content is unchanged

#### Scenario: Dragging a note of the group does not move the group
- **WHEN** the user drags a note of group G that contains several notes to another place in the list
- **THEN** only that note moves and G stays where it was

### Requirement: Renaming a group
The right-click menu of a group's top band SHALL offer "Renommer le groupe", which SHALL open a popup pre-filled with the current name. Confirming a non-blank name SHALL rename the group with the trimmed value; cancelling or confirming a blank name SHALL leave it unchanged. Renaming SHALL NOT change the group's notes or position.

#### Scenario: Renaming
- **WHEN** the user right-clicks the top band of group "Sprint 12", chooses "Renommer le groupe" and enters "Sprint 13"
- **THEN** the top band reads "Sprint 13" and still does after a restart

#### Scenario: Blank name refused
- **WHEN** the user confirms an empty name in the rename popup
- **THEN** the group keeps its previous name

### Requirement: Dissolving a group
The right-click menu of a group's top band SHALL offer "Dissoudre le groupe". Choosing it SHALL remove the group without deleting any note: its notes SHALL become ungrouped notes, in the same order, at the position the group occupied.

#### Scenario: Dissolving
- **WHEN** the list shows A, then group G containing B and C, then D, and the user dissolves G
- **THEN** the list shows the ungrouped notes A, B, C, D in that order

### Requirement: Group top band is not list background
Right-clicking or double-clicking a group's top band SHALL NOT trigger the actions attached to the empty area of the notes list: a double-click SHALL NOT create a note.

#### Scenario: Double-click on the top band
- **WHEN** the user double-clicks the top band of a group
- **THEN** no note is created

### Requirement: Hidden notes inside groups
Hiding a note SHALL NOT remove it from its group. The notes eye SHALL filter notes inside groups as it does outside them. A group none of whose notes is displayed SHALL NOT be displayed, and SHALL reappear with its name, content and position when at least one of its notes is displayed again.

#### Scenario: Group partly hidden
- **WHEN** group G contains A and the hidden note H, and hidden notes are not displayed
- **THEN** G is displayed with A only

#### Scenario: Group entirely hidden
- **WHEN** every note of group G is hidden and hidden notes are not displayed
- **THEN** G is not displayed; when the user makes hidden notes visible, G is displayed with its notes semi-transparent

### Requirement: Groups belong to their project
A group SHALL contain only notes of its own project. Deleting a project SHALL delete its groups along with its notes. Grouping SHALL NOT change the notes and hidden-notes counters of the project list, nor search results.

#### Scenario: Deleting a project with groups
- **WHEN** the user deletes a project that has groups
- **THEN** the deletion succeeds and neither its notes nor its groups remain

#### Scenario: Counters unchanged
- **WHEN** the user groups three notes of a project
- **THEN** the project's notes counter is unchanged
