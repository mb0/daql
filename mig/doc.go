/*
Package mig provides tools to record and migrate model schemas and rules for data migrations.

Model versions are sequential integers, that are automatically assigned as part of the schema
history. If there are differences between the historic model and a given model or if there is no
history model, the version is incremented. The schema and project versions work in a similar manner
based on their children's versions. This way we can avoid explicitly declaring versions.

Data stores like plain files or databases need to store model versions. Programs involved with
schema migration have the full schema history to calculate any new versions, other programs only
need a schema manifest that has the latest historic model versions and a hashes of their definition.

The schema history and manifest are automatically managed by mig and are transparently represented
as files. Changes need to be explicitly committed to the schema history and manifest. Uncommitted
schema changes are automatically represent as a new version,

Diff can be used to calculate the difference of two models as is used to extract specific changes
made.

Rules can be used to migrate data on the schema level. Specific migration routines use these rules
to migrate literals, events, or even their representations in specific environments like sql.
*/
package mig
