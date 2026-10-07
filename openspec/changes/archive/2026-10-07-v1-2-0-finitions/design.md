# Design

## Context

See proposal.md for motivation. Relevant current state:

- `RichEditor.tsx` owns the toolbar, including the two full-screen buttons (`👁`, `⛶`). It knows nothing about the note or meeting it edits: it receives `content`, `onChange`, `onBlur`, `readOnly`. The toolbar is hidden in the read-only full view.
- `NotesTab.tsx` and `MeetingsTab.tsx` render list rows as buttons with `bg-neutral-200` for the open item, `bg-sky-100` for multi-selected notes, `italic opacity-50` for hidden items. In `MeetingsTab` the editor edits an *instance*; the meeting has no editor of its own.
- Hiding (v1.1.0) set the pattern for a display-only attribute: a column with a default, a `SetHidden` repository method that does not touch `updated_at`, a `Set…Hidden` binding returning the entity.
- `ProjectSidebar.tsx` `background()` returns the fixed grey `--color-annulee` for a locked project before looking at `hasCritical` / `hasHigh`, which `stats.Sidebar` already computes for every project, active tasks only.
- `App.GetTaskView` fills `Due` for every task that has a due date, whatever its status; `TaskTree` and `TaskDetail` render what they receive. The sidebar icon, Priorités and the tray already ignore non-active tasks.
- `releases.json` holds `changes` as a list of strings; `app_about.go` parses and checks it (`checkReleases`), `AProposView.tsx` renders one bullet list. Migrations stop at 4.

## Goals / Non-Goals

**Goals:**
- Colour follows the hiding pattern end to end, so nothing new has to be learnt.
- `RichEditor` stays ignorant of notes and meetings.
- The due-date rule lives in one place.

**Non-Goals:**
- Custom colours, colours on projects, tasks, groups or instances.
- Filtering or sorting by colour.
- Setting a meeting's colour while it has no instance (there is no editor then, hence no button; the colour can be set as soon as an instance exists).
- Changing the sidebar due icon, Priorités or the tray.

## Decisions

### 1. Colour stored as a palette key

Migration 5: `ALTER TABLE notes ADD COLUMN color TEXT NOT NULL DEFAULT ''` and the same on `meetings`. The value is a key (`rose`, `peche`, `jaune`, `anis`, `menthe`, `turquoise`, `ciel`, `lavande`, `lilas`, `sable`) or `''` for none. `domain` holds the list of keys and rejects anything else (`ErrInvalidColor`); the frontend maps keys to hex values in one module (`couleurs.ts`) with their French labels.

- *Why a key rather than a hex value*: the palette can be retuned later without touching stored data, and an invalid value cannot be written.
- *Trade-off*: the ten keys exist in Go and in TypeScript. A key unknown to the frontend renders as no colour, so a mismatch degrades gracefully; a Go test and a frontend test each pin the list.

### 2. Bindings and repositories

`NoteRepository.SetColor(id, color)` and `MeetingRepository.SetColor(id, color)`, written like `SetHidden` (no `updated_at`, no index write). Bindings `SetNoteColor(noteID, color)` and `SetMeetingColor(meetingID, color)` validate the key and return the entity. `NoteRepository.Update` keeps writing only title and content, so autosave cannot erase a colour.

### 3. Colour button: generic props on `RichEditor`

`RichEditor` gains two optional props, `color?: string` and `onColorChange?: (color: string) => void`. When `onColorChange` is given, a `🎨` button ("Couleur") is rendered right after `⛶`; it opens a small popover (`role="menu"`) of ten swatches plus "Aucune couleur", the current one marked with `aria-checked`. Without the prop there is no button, so any other use of the editor is unchanged.

- Escape closes the popover first: the existing full-screen Escape handler ignores the key while the popover is open.
- The button uses `onMouseDown` + `preventDefault`, like its neighbours, so the editor keeps focus and no blur-flush is triggered by opening the palette.
- *Alternative considered*: an entry in the right-click menu of the list. Rejected: the request places the control in the editing area; it would also be a second way to do the same thing.

`NotesTab` passes `selected.color` and a handler that flushes pending typing, calls `SetNoteColor`, then merges only `color` into its `notes` state (same precaution as the autosave merge added in 1.2.0). `MeetingsTab` passes the *meeting's* colour and calls `SetMeetingColor` for the selected meeting; instance rows are not touched.

### 4. Rendering a coloured row

Row background becomes an inline `backgroundColor` when the item has a colour. Because the open item can no longer be recognised by its grey background, the open marker becomes independent of the background: a 2 px inset ring in `--color-selection` (the colour already used for the selected project and task), kept together with `bg-neutral-200` for uncoloured rows. Multi-selected notes get an inset sky ring in addition to `bg-sky-100` when uncoloured. Hover darkening is dropped on coloured rows.

### 5. Transverse project tint

`background()` looks at importance inside the locked case and returns one of two new theme colours, `--color-critique-attenuee` (`#e6a8a8`) and `--color-haute-attenuee` (`#e6cda8`), declared in `style.css`. Each is 35 % of the importance colour mixed with 65 % of the project's grey, written as a plain hex value. No backend change: the flags are already there, and the 30 s / `revision` refresh already applies.

- *Why mixing with the grey*: the result stays in the grey family, so the row still reads as the special project, while being distinct from both plain grey and the full colours.
- *Why a precomputed hex rather than `color-mix()`*: same result, no dependency on the WebView2 version.

### 6. Due information only for active tasks

`GetTaskView` fills `Due` only for `t.Active()` tasks. Tree and detail then show nothing for completed and cancelled tasks without any frontend rule. In `TaskDetail`, the "remove due date" button is currently rendered only when a `due` label exists; it is re-keyed on `task.dueDate` so that the stored date of a finished task can still be seen in the date field and removed.

- Cancelled tasks are included with completed ones: a lateness on an abandoned task is the same noise. Stated in the spec.
- *Alternative considered*: hiding in `TaskTree`/`TaskDetail`. Rejected: two places to keep in sync, and the backend already centralises the labels.

### 7. Release notes model

```json
{ "type": "ajout" | "correction", "level": "majeur" | "mineur", "text": "…" }
```

`Release.Changes` becomes `[]Change`. `checkReleases` rejects an empty text, an unknown `type` or `level`, naming the version. All existing entries are converted; a plain string no longer parses, so an unconverted entry fails the embedded-file test rather than being displayed uncategorised.

`AProposView` renders, per release, "Ajouts" then "Corrections" (each only if non-empty), each list stably sorted major first; a major item carries a small "Majeur" badge and bold text, a minor one is plain.

Classification of the existing history (adjustable in the file):

| Version | Change | Category |
|---|---|---|
| 1.2.0 | note groups and manual order | ajout majeur |
| 1.2.0 | colours on notes and meetings | ajout majeur |
| 1.2.0 | transverse project tint | correction mineure |
| 1.2.0 | no due information on a finished task | correction mineure |
| 1.1.1 | meeting selection no longer jumps on save | correction majeure |
| 1.1.1 | formatting buttons follow the cursor | correction mineure |
| 1.1.0 | full-screen views, hiding | ajout majeur |
| 1.1.0 | version in title bar, "À propos" tab | ajout mineur |
| 1.0.0 | all three entries | ajout majeur |

## Risks / Trade-offs

- [Palette keys duplicated in Go and TypeScript] → unknown key renders neutral; one test on each side pins the list.
- [A pastel too close to the selection or hidden styling] → open/selected state is carried by a ring, not by the background; hidden keeps `italic opacity-50`.
- [`releases.json` shape change breaks the demo backend and existing About tests] → both are updated in the same task group as the model.

## Migration Plan

Migration 5 is additive and runs at startup. Rolling back to an earlier binary: the `color` columns are ignored and colours simply do not show.
