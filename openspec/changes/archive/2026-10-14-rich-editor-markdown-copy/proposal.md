# Proposal

## Why

Copying from a rich editor (note, meeting minutes) and pasting into a plain-text editor such as VS Code gives unusable text: every list item, heading or paragraph turns into a bare line separated by several blank lines, bullets and heading levels are lost, and images vanish. The user wants the copied text to be Markdown, so that structure survives the paste.

While in the editors: a horizontal separator line can already be created by typing `---` (or `***` then Enter), but nothing in the toolbar offers it, so it stays undiscoverable.

## What Changes

- **Copy as Markdown.** The plain-text form that the rich editor puts on the clipboard (Ctrl+C, Ctrl+X) becomes Markdown: `* ` bullet lists, `1. ` numbered lists, `#` headings by level, `**bold**`, `*italic*`, `~~strike~~`, `` `code` ``, fenced code blocks, `> ` quotes, links, `---` rules, and tight lists without blank lines between items.
- **Images in the Markdown.** Images are written as `![image](data:image/…;base64,…)`, so that a Markdown preview (VS Code) shows them.
- **Single image.** When the selection is exactly one image, the image itself (PNG) is also put on the clipboard, so it can be pasted into an image editor, a chat or a mail.
- **Right-click menu in the editor** with "Copier en Markdown", which copies the selection exactly as Ctrl+C does.
- **Copy-all toolbar button:** an icon button (no text) in the formatting toolbar copies the whole content of the editor as Markdown, without selecting anything. A short visual confirmation shows the copy succeeded.
- The formatted (HTML) form that the editor already puts on the clipboard is left as it is today: pasting into another note or into Word keeps the formatting, as before. No "Copier en HTML" entry is added (explicitly out of scope).
- Pasting into the editor is unchanged.
- **Separator button:** an icon button (no text) in the formatting toolbar inserts a horizontal separator line at the cursor, the same line `---` already produces.
- Release notes: two minor additions in the `1.2.1` entry (version not yet released, same branch): copy as Markdown, and the separator button.

## Capabilities

### New Capabilities
- `rich-editor-markdown-copy`: the Markdown form of copied rich-editor content (selection and whole content), its rules per element, the single-image case, and the two ways to trigger it besides Ctrl+C (context menu, toolbar button).
- `rich-editor-horizontal-rule`: the toolbar button inserting a horizontal separator line.

### Modified Capabilities
(none — `rich-editor-toolbar` only governs the pressed state of formatting buttons; neither new button is a toggle, and both are specified in the new capabilities)

## Impact

- Affected code: `frontend/src/components/RichEditor.tsx` (clipboard text serializer, copy handling for a single image, context menu, copy-all and separator toolbar buttons), a new pure module `frontend/src/markdown.ts` serializing ProseMirror content to Markdown with its tests, `frontend/src/components/RichEditor.test.tsx`, `releases.json`.
- No backend, database or binding change. Stored content stays HTML.
- No new dependency.
