# rich-editor-markdown-copy Specification

## Purpose

Makes content copied from the rich text editor (notes, meeting minutes) arrive as Markdown when pasted into a plain-text editor, keeping lists, headings, emphasis and images, through Ctrl+C, a right-click entry, or a toolbar button copying the whole content.

## Requirements

### Requirement: Copied text is Markdown
When content is copied or cut from a rich text editor (Notes tab, Meetings tab, inline or full-window, editable or read-only), the plain-text form put on the clipboard SHALL be the Markdown rendering of the selected content. The formatted form put on the clipboard for applications that accept formatted content SHALL remain as before this change. Pasting into a rich text editor SHALL behave as before this change.

#### Scenario: Bullet list pasted into a text editor
- **WHEN** the user selects a bullet list of three items "text1", "text2", "text3", presses Ctrl+C and pastes into a plain-text editor
- **THEN** the pasted text is exactly `* text1\n* text2\n* text3`

#### Scenario: Cut
- **WHEN** the user selects a heading and presses Ctrl+X
- **THEN** the heading is removed from the editor and the clipboard plain text is its Markdown rendering

#### Scenario: Formatted paste unchanged
- **WHEN** the user copies a bullet list from a note and pastes it into another note
- **THEN** a bullet list is inserted, as before this change

#### Scenario: Selection inside a paragraph
- **WHEN** the user selects the words "très **important**" in the middle of a paragraph and copies them
- **THEN** the clipboard plain text is `très **important**`, with no surrounding blank lines

### Requirement: Markdown rendering rules
The Markdown rendering SHALL follow these rules:
- Consecutive blocks SHALL be separated by exactly one blank line; there SHALL be no trailing blank lines.
- A heading of level n SHALL be rendered as n `#` characters, a space, and its text.
- A bullet list item SHALL start with `* `; a numbered list item SHALL start with its number followed by `. `, numbering from the list's start number. Items of the same list SHALL NOT be separated by blank lines. A nested list SHALL be indented under its parent item by the width of the parent's marker.
- Bold SHALL be rendered `**text**`, italic `*text*`, strike `~~text~~`, inline code `` `text` ``, and a link `[text](url)`. Underline, which has no Markdown form, SHALL be rendered as its plain text.
- A code block SHALL be rendered as a fenced block (```` ``` ````), with its language after the opening fence when one is set, and its content unchanged.
- A quote SHALL be rendered with each of its lines prefixed by `> `.
- A horizontal rule SHALL be rendered `---`.
- A hard line break inside a block SHALL be rendered as a line ending with a backslash (`\`).
- An image SHALL be rendered `![alt](src)`, where `src` is the image's embedded data (`data:image/…;base64,…`) and `alt` is its alternative text, or `image` when it has none.
- Text SHALL be copied as typed: characters that Markdown could interpret are not escaped.

#### Scenario: Headings
- **WHEN** the selection contains a level-2 heading "Titre" followed by a paragraph "Texte"
- **THEN** the Markdown is `## Titre\n\nTexte`

#### Scenario: Numbered list with nested bullets
- **WHEN** the selection is a numbered list "Un" (with a nested bullet "a"), "Deux"
- **THEN** the Markdown is `1. Un\n   * a\n2. Deux`

#### Scenario: Image
- **WHEN** the selection contains a paragraph "Avant" and a pasted image without alternative text
- **THEN** the Markdown is `Avant\n\n![image](data:image/png;base64,…)` with the image's full embedded data

#### Scenario: Code block
- **WHEN** the selection is a code block containing `a  *  b`
- **THEN** the Markdown is ```` ```\na  *  b\n``` ````

### Requirement: Single image also copied as an image
When the selection is exactly one image and nothing else, copying SHALL also put that image on the clipboard as a PNG image, in addition to the Markdown text, so that applications that accept images receive the image itself.

#### Scenario: Copying a selected image
- **WHEN** the user clicks an image in a note to select it and presses Ctrl+C
- **THEN** the clipboard holds the image as PNG and the Markdown `![image](data:…)` as plain text

### Requirement: Copy as Markdown from the context menu
Right-clicking inside a rich text editor SHALL open a menu offering "Copier en Markdown". Choosing it SHALL copy the current selection exactly as Ctrl+C does. When the selection is empty, the entry SHALL be shown disabled. The menu SHALL be available in every view mode of the editor, including the read-only views.

#### Scenario: Copy from the menu
- **WHEN** the user selects a bullet list, right-clicks it and chooses "Copier en Markdown"
- **THEN** the clipboard plain text is the Markdown of the list, as with Ctrl+C

#### Scenario: Nothing selected
- **WHEN** the user right-clicks in the editor without any selection
- **THEN** "Copier en Markdown" is shown disabled

### Requirement: Copy the whole content from the toolbar
The formatting toolbar of the rich text editor SHALL offer a button showing only an icon (no text label), with the accessible name and tooltip "Copier tout en Markdown". Activating it SHALL copy the whole content of the editor as Ctrl+C would after selecting everything, SHALL leave the user's selection and cursor where they were, and SHALL show a brief visual confirmation on the button. The button SHALL be present wherever the formatting toolbar is shown.

#### Scenario: Copy all
- **WHEN** a note holds a heading "Titre" and a bullet list "a", "b", the cursor is inside "a", and the user clicks "Copier tout en Markdown"
- **THEN** the clipboard plain text is `## Titre\n\n* a\n* b`, the cursor is still inside "a", and the button briefly shows a confirmation

#### Scenario: Empty content
- **WHEN** the editor is empty and the user clicks "Copier tout en Markdown"
- **THEN** the clipboard plain text is empty and no error is shown
