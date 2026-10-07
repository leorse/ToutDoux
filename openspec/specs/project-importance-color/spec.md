# project-importance-color Specification

## Purpose
Makes the project list show, through the background colour of each project row, whether the project holds an active task of Critique or Haute importance.

## Requirements

### Requirement: Project row colour follows task importance
A project row in the project list SHALL take the Critique colour when the project has at least one active task of Critique importance, otherwise the Haute colour when it has at least one active task of Haute importance, otherwise its neutral background. Completed and cancelled tasks SHALL NOT count.

#### Scenario: Critique wins over Haute
- **WHEN** an ordinary project has one active Critique task and one active Haute task
- **THEN** its row has the Critique colour

#### Scenario: Only finished important tasks
- **WHEN** the only Critique task of a project is completed
- **THEN** its row has the neutral background

### Requirement: Attenuated colour for the transverse project
The locked "Transverse / Divers" project SHALL follow the same rule with attenuated colours: an attenuated Critique tint, or an attenuated Haute tint, each visibly different both from the corresponding colour of an ordinary project and from the grey the transverse project has when it holds no such task. Without any active Critique or Haute task it SHALL keep its grey background. The row SHALL update without changing view when a task is created, completed or changes importance.

#### Scenario: Transverse project with a Critique task
- **WHEN** "Transverse / Divers" has an active Critique task
- **THEN** its row shows the attenuated Critique tint, different from the colour of an ordinary project holding a Critique task

#### Scenario: Transverse project with a Haute task
- **WHEN** "Transverse / Divers" has an active Haute task and no active Critique task
- **THEN** its row shows the attenuated Haute tint

#### Scenario: Transverse project without important task
- **WHEN** "Transverse / Divers" has only Normale and Basse active tasks
- **THEN** its row keeps its grey background

#### Scenario: Back to grey
- **WHEN** the only Critique task of "Transverse / Divers" is marked completed
- **THEN** its row returns to the grey background without the user changing view
