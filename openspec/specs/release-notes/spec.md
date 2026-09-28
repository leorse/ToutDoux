# release-notes Specification

## Purpose

Lets users see, from within the application, which version they run and what changed in every release, in a dedicated "À propos" tab of the Preferences view.

## Requirements

### Requirement: Preferences sub-tabs
The Preferences view SHALL offer two sub-tabs, "Paramétrage" and "À propos". "Paramétrage" SHALL be shown by default and SHALL keep the same content the Preferences view had before this change (the sub-tab is a rename only, not a new heading duplicating it).

#### Scenario: Default sub-tab
- **WHEN** the user opens the Preferences view
- **THEN** the "Paramétrage" sub-tab is shown, with the same content as before this change

#### Scenario: Switching to About
- **WHEN** the user selects the "À propos" sub-tab
- **THEN** the About content is shown in place of the preferences content

### Requirement: Current version in About
The "À propos" sub-tab SHALL display the current application version.

#### Scenario: Version displayed
- **WHEN** the user opens the "À propos" sub-tab of an application configured with version `1.1.0`
- **THEN** the text `1.1.0` is visible in it

### Requirement: Versioned release history
The "À propos" sub-tab SHALL list every release, most recent first. Each release SHALL show its version, its date, and its list of changes as bullet points, and MAY show one highlighted "important" notice ahead of the list. Text within a release SHALL support bold and italic emphasis; any other markup SHALL be displayed as plain text, not interpreted.

#### Scenario: Release with several changes
- **WHEN** a release lists three changes
- **THEN** its version and date are shown, followed by three bullet points

#### Scenario: Release with an important notice
- **WHEN** a release defines an important notice
- **THEN** the notice is displayed highlighted, before that release's list of changes

#### Scenario: Release without an important notice
- **WHEN** a release defines no important notice
- **THEN** no highlighted notice is shown for it

#### Scenario: Inline emphasis
- **WHEN** a change's text contains `**gras**` and `*italique*`
- **THEN** it is displayed with bold and italic emphasis respectively, and the marker characters are not shown

#### Scenario: Other markup is not interpreted
- **WHEN** a change's text contains an HTML tag such as `<b>x</b>`
- **THEN** it is displayed literally as text and is not rendered as HTML

### Requirement: Release notes consistent with the version
The most recent release entry SHALL carry the same version as the configured application version. This SHALL be enforced by an automated check that fails otherwise.

#### Scenario: Newest release matches
- **WHEN** the newest release entry has version `1.1.0` and the configured version is `1.1.0`
- **THEN** the automated check passes

#### Scenario: Newest release out of date
- **WHEN** the configured version is `1.0.2` but the newest release entry is `1.0.1`
- **THEN** the automated check fails
