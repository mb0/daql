/*
Package mig provides tools to version, record and migrate a project schema, and also provides rules
to migrate the project data.

Project dom nodes are assigned sequential version numbers, that are automatically determined based
on the node's content and its last known version. The version starts at one for new nodes and is
incremented if the old and new definition differ. The schema and project versions work in a similar
manner, only on based on the hashes of their children. This way we can avoid explicitly declaring
versions.

A project manifest contains all details needed to determine version changes. It contains the version
information for the project and all its nodes. The version information includes a sha256 hash. For
models this hash is calculated based on the default string representation, which means that any
change to the model results in a new hash. For schemas, the hash is based on the schema name and all
its model hashes, and for projects similarly the name and schema hashes. Effectively only changes to
models are automatically increment the version. Users should be able to force version updates for
any dom node.

All datasets like backups or databases should store at least the project versions. Programs involved
with data migration have the full project history to calculate any new versions, other programs only
need the project manifest.

The schema history and manifest are managed by the daql command and are written to files. Changes
need to be explicitly recorded into the project history and manifest. Data migration rules should
also be recorded for each version as part of the history. Simple migration rules can be expressed
as xelf expressions and interpreted by the daql command. Complex migration rules call any command,
usually a simple go script that migrates one or more model changes for a dataset. The daql command
should be able to generate simple rules and migration templates.
*/
package mig
