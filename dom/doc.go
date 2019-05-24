/*
Package dom provides code for domain model declaration and registration.

Models are the primary focus of this package, they are used to define the bits, enum and object
schema types as well as service contracts. In addition to the schema type information, models hold
meta data for display, indexing, constraints and other application relevant aspects.

Models are always part of a named schema and multiple schemas are usually represented and used as a
project. A model can refer to models in other previously declared schemas within a project.

Models should be used for any data that crosses environments, such as persisted data used in both
the database and program environment or protocol data used to communicate between processes.

Models, schemas and projects have a JSON representations that can be sent to clients. Models can
have basic validation hints, but more detailed display and form logic should always use another
layer.

Easy filtering of all project parts is important for hiding irrelevant or restricted data in
different environments or even on a per-connection basis to hide sensitive information for roles
without permission.

Models need to describe references to establish model and schema dependencies. The project might be
a good place to collect some facts about model and schema references.

Using models to describe service contracts and their arguments documents the service and lets us
automatically read and route requests, and generally generate code based on it.
*/
package dom
