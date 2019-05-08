/*
Package dom provides code for domain model declaration and registration.

Models are the primary focus of this package, they are used to define the flag, enum and object
schema types as well as service contracts. In addition to the schema type information, models hold
meta data for display, indexing, constraints and other application relevant aspects.

Models are always part of a named schema and multiple schemas are usually represented and used as a
project. A model can refer to models in other previously declared schemas within a project.

Models should be used for any data that crosses environments, such as persisted data used in both
the database and program environment or protocol data used to communicate between processes.

Models have a numeric version which should never be explicitly declared or updated. Instead the
model versions are automatically determined in comparison to historic model information. The schemas
and projects also have a numeric version derived in a similar fashion. The number 0 is used if no
history is provided. Otherwise we compare to the latest history version and use its version or
increment by one when there are changes.

Models, schemas and projects have a JSON representations that can be sent to clients. Models can
have basic validation hints, but more detailed display and form logic should always use another
layer.

Easy filtering of all project parts is important for hiding irrelevant or restricted data in
different environments or even on a per-connection basis to hide sensitive information for roles
without permission. A filtered node is marked as such by using a negative version number, that is
the negative of the original node's version. The version is reused to avoid not mixing up model
declarations with their filtered views.

Models need to describe references to establish model and schema dependencies. The project might be
a good place to collect some facts about model and schema references.

Using models to describe service contracts and their arguments documents the service and lets us
automatically read and route requests, and generally generate code based on it. With model versions
we can also detect and handle version mismatches, or even serve multiple versions for upgrades.
*/
package dom
