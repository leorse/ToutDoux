# app-version Specification

## Purpose

Gives the application a single, configurable version number and shows it to the user in the native window title, so that anyone can tell which build they are running.

## Requirements

### Requirement: Single configured version
The application version SHALL be defined in exactly one configuration file, in `MAJOR.MINOR.PATCH` form, and SHALL NOT be hard-coded in application source code. Every place that shows or embeds the version (window title, "À propos" tab, installer metadata) SHALL use that same value.

#### Scenario: Version changed in one place
- **WHEN** the version is changed in the configuration file and the application is rebuilt
- **THEN** the window title, the "À propos" tab and the installer's version metadata all show the new version without any other edit

#### Scenario: Malformed version
- **WHEN** the configured version is not of the form `MAJOR.MINOR.PATCH`
- **THEN** the automated checks fail

### Requirement: Version in the window title
The native window title SHALL read `Tout Doux (<version>)`, for example `Tout Doux (1.1.0)`. The application SHALL NOT render its own title bar inside the window content.

#### Scenario: Title at startup
- **WHEN** the application starts with configured version `1.1.0`
- **THEN** the window title is `Tout Doux (1.1.0)`

#### Scenario: No in-page title bar
- **WHEN** the application window is displayed
- **THEN** the application name appears only in the native window title, not as a line inside the window content
