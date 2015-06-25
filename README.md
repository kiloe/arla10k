[![Circle CI](https://circleci.com/gh/kiloe/arla10k/tree/master.svg?style=svg)](https://circleci.com/gh/kiloe/arla10k/tree/master)

## What is Arla10k?

Arla10k is a framework that implements the [Arla](https://github.com/kiloe/arla) architecture using:

* Postgres + plv8 as the query store.
* SQLite for data persistence.
* Node/Express for the HTTP API.

It is an "everything in one container" implementation targeted at single host deployments. It should be ideal for application development and small production deployments. The "10k" refers to the target number of active users of your application (with a concurrency rate of ~25%) so this implementation aims to be useful for applications where you expect 25-50 requests/sec.
