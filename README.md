daql
====

daql is meant to provide domain meta model and query tool using the xelf framework.

It is a work in progress.

Motivation
----------

The repetitive chore of querying, validating, tracking complex, interdependent data across
frontend, backend and database is a hassle every web developer knows well.

While ORMs and automatic user interfaces are a reason for people to use frameworks like Django or
Rails for fast prototyping, the weight of dependencies and the limits they pose on the overall
design makes them unfit for some use cases.

Rich web, mobile or desktop apps all have their own platforms. You may want to query complex
interdependent queries from all of these. Simple RPC or HTTP APIs require very specialized
endpoints or multiple queries to the backend. But what if you want pagination and sorting? APIs and
Endpoints usually grow in complexity and parameters or the clients need to implement the logic.

GraphQL tries to formalize a language for complex API queries. It is designed for clearly defined
APIs, as used by big teams and corporations.

The author is a single developer trying to create applications for a small business. The projects
data model needs a full history, should support sync and offline work and is ever changing.

It would be great to have a simple language that can easily ported and adapted to be used for
complex queries, schema declaration, code generation or even layouts.

License
-------

Copyright (c) Martin Schnabel. All rights reserved.
Use of the source code is governed by a BSD-style license that can found in the LICENSE file.
