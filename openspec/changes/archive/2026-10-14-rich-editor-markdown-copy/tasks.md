# Tasks

## 1. Markdown serializer

- [x] 1.1 Create `frontend/src/markdown.ts` exporting `fragmentToMarkdown(fragment)` for every StarterKit node and mark plus images, following the rules of `specs/rich-editor-markdown-copy/spec.md` (block separator, `#` by level, `* ` / `n. ` with start number, nested indentation by parent marker width, tight lists, mark runs, underline as plain text, fenced code with language, `> ` quotes, `---`, hard break as `\`, `![alt|image](src)`, no escaping) — verify with `frontend/src/markdown.test.ts` building documents with the editor schema and covering every spec scenario plus: partial selection inside a paragraph, partial selection across two list items, bold run across adjacent text nodes, link with bold text, three-level nesting, ordered list starting at 3, empty fragment → `''` (`npx vitest run markdown`)

## 2. Editor integration

- [x] 2.1 Plug `clipboardTextSerializer` into `RichEditor.tsx` `editorProps` — verify in `RichEditor.test.tsx` that dispatching `copy` on a selected bullet list sets `text/plain` to `* text1\n* text2\n* text3` while `text/html` is still set, and that `cut` removes the selection with the same plain text
- [x] 2.2 Handle the single-image selection in `handleDOMEvents.copy`/`cut` (canvas → PNG blob, `ClipboardItem` with `image/png`, `text/plain`, `text/html`; fallback to default copy on rejection) — verify with tests stubbing `navigator.clipboard.write` and `HTMLCanvasElement.prototype.toBlob`: an image `NodeSelection` triggers one write holding the three types; a rejected write falls back without throwing; a selection with text and image does not call `clipboard.write`
- [x] 2.3 Add the editor right-click menu ("Copier en Markdown", disabled on empty selection) using `ContextMenu`, available in all view modes including read-only — verify with tests stubbing `document.execCommand`: the entry calls `execCommand('copy')`; it is disabled with an empty selection; the menu opens in the read-only full view
- [x] 2.4 Add the "Copier tout en Markdown" icon button (inline SVG, `aria-label`/`title`, ✓ for 1.5 s) to the toolbar, selecting all, copying and restoring the selection — verify with tests: the button has no visible text, triggers `execCommand('copy')` while the whole document is selected, restores the cursor position afterwards, shows the confirmation; and that `npx vitest run` passes in full (toolbar pressed-state tests unchanged)
- [x] 2.5 Add the "Ligne de séparation" icon button (inline SVG, `aria-label`/`title`, after "Bloc de code") calling `setHorizontalRule()` — verify with tests: clicking it in a paragraph puts an `<hr>` in the HTML passed to `onChange`, text typed right after lands in the paragraph following the line, the button never has `aria-pressed="true"`, and it is absent when `readOnly`

## 3. Release

- [x] 3.1 Add two minor additions to the `1.2.1` entry of `releases.json` ("ajout", "mineur"): copy as Markdown (Ctrl+C, right-click, toolbar button, images included), and the separator-line button — verify `npx vitest run aPropos` and `go test ./...` pass

## 4. Integration check

- [x] 4.1 In `wails dev`, copy from a note a bullet list, a heading with a paragraph, a numbered nested list and an image, and paste each into VS Code; copy a single image and paste it into Paint; copy a list and paste it into Word and into another note; use the right-click entry and the toolbar button; insert a separator with its button, save, reopen the note — verify each result matches the spec scenarios and `npm run build` type-checks
