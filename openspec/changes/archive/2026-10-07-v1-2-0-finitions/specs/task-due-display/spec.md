# Spec Delta

## Purpose

Defines when a task's due date is rendered as a remaining time or a lateness, so that only tasks still to be done draw attention to their deadline.

## ADDED Requirements

### Requirement: No due information on a finished task
A completed or cancelled task SHALL NOT display its remaining time, its lateness or a due-date icon, in the task tree and in the task detail, whatever its due date and however much time has passed since. Its due date SHALL be kept: an active task SHALL keep displaying its due information as before, and a finished task made active again SHALL display it again.

#### Scenario: Completed after the deadline has passed
- **WHEN** a task whose due date is two days in the past is completed
- **THEN** its row in the task tree shows neither "En retard" nor a clock icon

#### Scenario: Completed in time, viewed later
- **WHEN** a task was completed before its due date and that date has since passed
- **THEN** it is not shown as late

#### Scenario: Cancelled task
- **WHEN** a task with a due date is cancelled
- **THEN** no due information is shown for it

#### Scenario: Task detail of a completed task
- **WHEN** the user selects a completed task that has a due date
- **THEN** the detail shows no remaining time, lateness or clock icon

#### Scenario: Reopening a task
- **WHEN** a completed task with a due date in the past is unchecked
- **THEN** it shows its lateness again

#### Scenario: Active task unchanged
- **WHEN** an active task has a due date in three days
- **THEN** its remaining time is displayed as before
