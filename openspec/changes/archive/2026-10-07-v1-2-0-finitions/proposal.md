# Proposal

## Why

Four small finishing touches for version 1.2.0, before it is released: notes and meetings cannot be told apart at a glance in their lists, the "Transverse / Divers" project never signals that it holds an important task, a completed task keeps drifting into "En retard" as time passes, and the release history mixes new features and bug fixes in one flat list.

## What Changes

- **Colour on notes and meetings.** A colour button in the editor toolbar, next to the two full-screen buttons, opens a palette of ten predefined pastel colours plus "no colour". The chosen colour becomes the background of the note, or of the meeting, in its list. In the Meetings tab the colour belongs to the meeting: the list of instances stays neutral. Colour is display-only, like hiding.
- **"Transverse / Divers" reacts to importance.** When it holds an active Critique or Haute task, its row in the project list takes an attenuated tint of the corresponding colour instead of its fixed grey, so that it still reads as the special project.
- **No due information on a finished task.** A completed or cancelled task no longer shows its remaining time, its lateness or the clock icon, in the task tree and in the task detail. The due date itself is kept.
- **Categorised release notes.** Each change of a release is an addition ("ajout") or a fix ("correction"), and major or minor. The "À propos" tab shows additions and fixes under separate headings, major ones first and visually marked. The existing history is classified.
- **Release notes of 1.2.0** rewritten in the new form: note groups and manual order (major addition), colours (major addition), tint of the transverse project (minor fix), no due information on a finished task (minor fix). The categorisation of the release notes is itself not listed. The version stays `1.2.0`.

## Capabilities

### New Capabilities
- `item-colors`: choosing one of ten pastel colours for a note or a meeting from the editor toolbar, and showing it in the lists.
- `project-importance-color`: background colour of a project row according to the importance of its active tasks, including the attenuated variant for the locked "Transverse / Divers" project.
- `task-due-display`: when the remaining time or lateness of a task's due date is shown.

### Modified Capabilities
- `release-notes`: the "Versioned release history" requirement changes — a release's changes are categorised (addition/fix, major/minor) and displayed grouped by category; the consistency check also validates the categories.

## Impact

- Affected code: SQLite migration 5 (`color` column on `notes` and `meetings`), `domain` (`Note.Color`, `Meeting.Color`, palette), `ports` and `adapters/sqlite` note and meeting repositories, `App` bindings (`SetNoteColor`, `SetMeetingColor`, `GetTaskView`, `GetAppInfo` / release model in `app_about.go`), `releases.json` (new shape, all entries), frontend `RichEditor.tsx` (colour button), `NotesTab.tsx`, `MeetingsTab.tsx`, `ProjectSidebar.tsx`, `TaskDetail.tsx`, `AProposView.tsx`, `style.css`, `demoBackend.ts`, regenerated `frontend/wailsjs` bindings.
- `releases.json` changes shape (`changes` becomes a list of objects): **BREAKING** for the file format only, read by nothing but the application.
- No change to search, semantic index, Priorités, tray, counters, note groups or ordering. No new dependency. Non-destructive migration.
