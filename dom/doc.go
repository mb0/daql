/*
Package dom provides code for domain model declaration and registration.

Models should be used for any data that crosses environments, such as persisted data used in both
the database and program environment or protocol data used to communicate between processes.

Models can extend, reference or embed other previously defined models, just like record types can.
Cyclic dependencies, except self or ancestor references, are not allowed.
*/
package dom
