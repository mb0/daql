/*
Package evt defines an interface for plain event sourcing and some generic event processors.

Event sourcing as a concept can be interpreted in various ways. Daql uses mostly 'dump' events,
that have generic create, update or delete commands. It can be used for more specific events,
those however must resolve to a sequence of generic events. Events have one central and
authoritative ledger, that assigns a revision, and with that order to all events.

There are three generic commands:
   '+' to create an new record
   '-' to hide or delete a record
   '*' to modify a specific record

Each event has a topic, key and command string and optional a argument map. Usually the topic
refers to record model name and the key to its primary key as string. The string key allows models
with uuid, integer and other character typed keys to share a ledger.

Custom commands have more meaningful names, validation and implementations.
They usually resolve to one or more generic events to allow a simple and consistent interface.

A ledger is a strictly ordered sequence of events and can be used recreate a state at any revision.

Users publish one or more events as a transaction. The events are resolved and validated,
assigned a revision and audit id and then written to the ledger.

A revision is a timestamp with millisecond granularity. It is usually the arrival
time of the event but cannot be before the latest applied revision in the persisted ledger.

Every transaction generates an audit log entry that has additional information
about the user, creation and arrival time and a map of extra information.

Servers usually update the latest state of the ledger in the same transaction that applies the
event to the ledger. This allows us to avoid event aggregates and materialized view consistency for
most operations. Stateless models and have their only persistent representation in the ledger.

To backup and restore both audit and event log are required.

Satellite servers can persist transactions failed due to recoverable errors like network outage
for later reconciliation and may choose to serve their clients the projected state of the ledger.

*/
package evt

//go:generate sh -c "go run gen.go evt.dom > evt.go"
