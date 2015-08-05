[![Circle CI](https://circleci.com/gh/kiloe/arla10k/tree/master.svg?style=svg)](https://circleci.com/gh/kiloe/arla10k/tree/master)

## What is Arla10k?

Arla10k is a framework that implements all server-side components of the [Arla](https://github.com/kiloe/arla) spec using:

* A flat file JSON log for data persistence.
* Postgres + plv8 as the ephemeral query store.
* A small Go HTTP server for the API layer (and optionally serving static files for a client app).

It is an "everything in one container" implementation targeted at single host deployments. It should be ideal for application development and small production deployments. The "10k" refers to the target number of active users of your application (with a concurrency rate of ~25%) so this implementation aims to be useful for applications where you expect 25-50 requests/sec.

## Quick Arla Recap...

You configure an Arla application using Javascript. You define your schema, the possible mutation functions and how your application will deal with authentication by declaring calling the `arla.configure({...})` function.

Client applications are expected to interact with an Arla datastore via four main HTTP API points:

* `POST /authenticate` used to retrieve an access_token to make other API calls.
* `POST /register` used to create a new user.
* `POST /query` used to fetch data by executing AQL (a GraphQL-like language).
* `POST /exec`  used to apply changes to the data (mutations).

## Getting Started
