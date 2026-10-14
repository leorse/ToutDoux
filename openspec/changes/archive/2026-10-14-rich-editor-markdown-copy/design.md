# Design

## Context

See proposal.md for motivation. Current state relevant to the approach:

- `RichEditor.tsx` is the only rich editor (Notes and Meetings tabs). TipTap 3.30 with `StarterKit` (paragraph, heading 1–6, bullet/ordered list, list item, blockquote, code block, horizontal rule, hard break, bold, italic, strike, code, link, underline) and `Image` with `allowBase64`. Content is stored as HTML; images are `data:` URIs inside it.
- On copy, ProseMirror (prosemirror-view 1.42) writes `text/html` (with its `data-pm-slice` marker, used for rich pastes between editors) and `text/plain` from `textBetween` with blank-line block separators — the cause of the "text1\n\n\n\ntext2" result.
- `editorProps` already customises pasting (`handlePaste`, `transformPastedHTML`). Toolbar buttons are built by a local helper and hidden when `readOnly` or in the read-only full view.
- A generic `ContextMenu` component (`MenuItem` with `disabled`) is used by the lists; the editor has no right-click menu, and Wails disables the WebView default menu in production builds.
- The project avoids new dependencies; pure logic lives in standalone, unit-tested modules (`notesLayout.ts`, `domain/*`).

## Goals / Non-Goals

**Goals:**
- One Markdown serialization used by every copy path (Ctrl+C, Ctrl+X, menu, button), so they cannot diverge.
- Leave the `text/html` flavour untouched, so rich paste between notes and into Word works as today.

**Non-Goals:**
- Markdown on paste (pasting Markdown text into the editor does not format it).
- A "Copier en HTML" action.
- Markdown escaping of special characters (see risks).
- Changing the stored format.

## Decisions

**1. Hand-written serializer in `frontend/src/markdown.ts`, plugged as `editorProps.clipboardTextSerializer`.**
`clipboardTextSerializer(slice)` is ProseMirror's hook for exactly the `text/plain` flavour; ProseMirror still writes `text/html` itself, and it applies to copy, cut and drag alike. The module exports `fragmentToMarkdown(fragment)` walking ProseMirror nodes by type name with the rules of the spec (block separator, list prefixes with indentation by parent marker width, marks wrapped around text runs, fences, quotes, images).
A slice opened inside a paragraph yields its inline content only; a slice opened inside list items keeps the list wrapper (ProseMirror includes the parents up to the common depth), so a partial list selection renders as list items.
*Alternatives:* `prosemirror-markdown` or `@tiptap/markdown` — new dependencies, and their defaults (`- ` bullets, escaping, loose lists) would need overriding anyway; `turndown` on the HTML — new dependency and a second parse. The node set is fixed by StarterKit, so a ~150-line serializer stays small and fully testable without a DOM.

**2. Marks: wrap maximal runs.** Adjacent text nodes sharing a mark are wrapped once (`**a b**`, not `**a** **b**`); marks are opened in a fixed order (link, bold, italic, strike, code) and closed in reverse. `code` content is not re-wrapped by other marks.

**3. Single image: take over the copy event.** In `handleDOMEvents.copy`, when the selection is a `NodeSelection` on an image, `preventDefault` and call `navigator.clipboard.write([new ClipboardItem({ 'image/png', 'text/plain', 'text/html' })])`. The PNG is obtained by drawing the data URI on a canvas (`toBlob('image/png')`) because the async Clipboard API only accepts PNG; `text/html` is the image's `<img>` tag so pasting into a note still works. Cut on a single image takes the same path then deletes the selection. If `clipboard.write` rejects, fall back to the default copy (Markdown + HTML, no PNG).

**4. Menu and button reuse the native copy path through `document.execCommand('copy')`.**
- Menu: `onContextMenu` on the editor wrapper opens `ContextMenu` with "Copier en Markdown" (disabled when `editor.state.selection.empty`); selecting it focuses the editor and runs `execCommand('copy')` on the current selection.
- Button: saves the selection, runs `editor.commands.selectAll()`, `execCommand('copy')`, then restores the saved selection (`setTextSelection` / the saved `Selection` mapped onto the unchanged doc). Shows a ✓ on the button for 1.5 s.
This guarantees the three triggers produce exactly what Ctrl+C produces, including the single-image case and the HTML flavour, with no second clipboard-writing code path. `execCommand('copy')` is deprecated but supported by WebView2/Chromium and allowed in a user-gesture handler.
*Alternative:* building the clipboard payload by hand with `navigator.clipboard.write` — would need ProseMirror's private HTML serialization for the `text/html` flavour.

**5. Button icon.** An inline SVG of the Markdown mark ("M↓"), consistent in size with the other toolbar buttons; `aria-label` and `title` "Copier tout en Markdown". Placed after the formatting buttons, before the full-screen controls. It is not a toggle and has no pressed state, so `rich-editor-toolbar` is unaffected.

**6. Separator button uses TipTap's existing command.** `HorizontalRule` is already part of `StarterKit` (its `---` / `***` input rules work today, and the line already displays correctly), so the button only calls `editor.chain().focus().setHorizontalRule().run()`, which inserts the line at the cursor or in place of the selection and puts the cursor in the following paragraph (creating one at the end of the document). No new extension, no style change. Icon: an inline SVG horizontal stroke, placed after "Bloc de code"; `aria-label`/`title` "Ligne de séparation"; not a toggle, so no pressed state.

## Risks / Trade-offs

- [No escaping: a typed `*` or a line starting with `#` can be re-read as formatting by a Markdown renderer] → Chosen for readability of the pasted text, which is the point of the change; recorded in the spec.
- [A large embedded image yields a Markdown line of several megabytes in VS Code] → Accepted by the user; the single-image PNG copy offers a lighter path.
- [`execCommand('copy')` could be removed from Chromium someday] → Isolated in one helper; replacing it with `clipboard.write` later does not change the spec.
- [jsdom has no real clipboard nor canvas] → The serializer is tested as pure functions on ProseMirror nodes built from the editor schema; component tests assert the `text/plain` data set on a dispatched `copy` event and stub `execCommand`/`clipboard.write`.

## Migration Plan

Frontend only. Rollback is a plain revert.
