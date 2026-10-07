# Spec Delta

## MODIFIED Requirements

### Requirement: Versioned release history
The "À propos" sub-tab SHALL list every release, most recent first. Each release SHALL show its version, its date, and its changes, and MAY show one highlighted "important" notice ahead of them. Every change SHALL be categorised as an addition or a fix, and as major or minor. Within a release, additions SHALL be listed under a heading "Ajouts" and fixes under a heading "Corrections", additions first; a heading SHALL NOT be shown when the release has no change of that kind. Under each heading, major changes SHALL come before minor ones and SHALL be visually marked as major, in a way that does not rely on colour alone. Text within a release SHALL support bold and italic emphasis; any other markup SHALL be displayed as plain text, not interpreted.

#### Scenario: Release with several changes
- **WHEN** a release lists three changes
- **THEN** its version and date are shown, followed by three bullet points

#### Scenario: Additions and fixes separated
- **WHEN** a release lists two additions and one fix
- **THEN** the two additions appear under "Ajouts" and the fix under "Corrections", "Ajouts" being first

#### Scenario: Release with fixes only
- **WHEN** a release lists only fixes
- **THEN** the heading "Corrections" is shown and no heading "Ajouts" is shown

#### Scenario: Major before minor
- **WHEN** a release lists, in that order, a minor addition and a major addition
- **THEN** the major addition is displayed first and is marked as major, and the minor one is not

#### Scenario: Release with an important notice
- **WHEN** a release defines an important notice
- **THEN** the notice is displayed highlighted, before that release's changes

#### Scenario: Release without an important notice
- **WHEN** a release defines no important notice
- **THEN** no highlighted notice is shown for it

#### Scenario: Inline emphasis
- **WHEN** a change's text contains `**gras**` and `*italique*`
- **THEN** it is displayed with bold and italic emphasis respectively, and the marker characters are not shown

#### Scenario: Other markup is not interpreted
- **WHEN** a change's text contains an HTML tag such as `<b>x</b>`
- **THEN** it is displayed literally as text and is not rendered as HTML

## ADDED Requirements

### Requirement: Every change is categorised
Every change of every release, including the releases published before this requirement, SHALL carry a kind (addition or fix) and a level (major or minor). This SHALL be enforced by the same automated check as the version consistency: it SHALL fail when a change has no text, an unknown kind or an unknown level.

#### Scenario: Valid history
- **WHEN** every change of the release history has a text, a known kind and a known level
- **THEN** the automated check passes

#### Scenario: Unknown kind
- **WHEN** a change is declared with a kind that is neither addition nor fix
- **THEN** the automated check fails and names the release concerned

#### Scenario: Missing level
- **WHEN** a change is declared without a level
- **THEN** the automated check fails
