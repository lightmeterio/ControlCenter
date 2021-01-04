# 4. SQLite as Storage

Date: 2021-01-04

## Status

Accepted

## Context

Early in the development of Control Center, we needed to choose some way to persist differents kind of data used by the application,
AKA database.

Initially the data was the content of the parsed log lines from postfix, but later it expanded to pretty much all kind of data, including
user data, insights data, application settings.

## Decision

Due the nature of the application and bad past experiences with big RDBMs (relational database management), I (Leandro, currently lead developer),
decided to use SQLite, as it has shown to be a very simple database system, still offering good performance for the operations we had in the early
development stages.

Other options were considered, such as MySQL, MariaDB and Postgres. NoSQL options were not considered due lack of knowledge on such technologies.

SQLite was chosen as it offers the simplest setup possible: no setup at all. The databases are managed by the application itself,
instead of from an external process, not requiring networking on any kind of connection: the application has direct access to the filesystem.

Direct access to the filesystem makes SQLite very fast for reading. It's basically limited by the performance of the underlying media, instead
of depending on the network or external processes.

## Consequences

SQLite is a different beast from other RDBMs, as it's not based on an client-server architecture. It makes the application setup very simple,
but limits horizontal scalability. As all databases are files located in the a filesystem the application have access, we are pretty much limited
to the processing power of the system Control Center runs on.

I don't have any data to back my opinion, but I believe that we won't reach any time soon the need to scale horizontally the database
(use multiple computers), and that SQLite will be good enough for most use cases and most, if not all, users we are currently targeting.

SQLite is also a very stable file format, so version updates are very likely to break internal (disk) representation of data. I experienced in the
past issues where upgrading MySQL would require migrating the internal disk format, making every upgrade quite risky.

Using SQlite has many downsides though:

- Lack of horizontal scalability. It scales very well in a single node with multiple CPUs, though.
- Backups are not as simple as connecting to a database server and dumping tables to text. It needs to be implemented using the SQLite (C) API
and is not a straightforward process.
- SQLite does not handle concurrency as other database systems to. Basically it's up to the application how to handle simultaneous writes to a
database. SQLite scales well for an unlimited number of readers, though.
- SQLite forces us to keep different kinds of data into different files. Although good from a archtectural perspective, as it keeps each
subprocess owning only its relevant data, it's way less simple than just "connect to a database and write stuff" approach from client-server based
databases.
- Most developers are not used to it and find a very strange (if not completely insane and wrong) decision to use SQLite in a infrastructure-kind
software that Control Center is.

As an alternative for the future, we should analyze if supporting multiple data storage backends makes sense, but I don't think it'll be needed so soon.

In a sentence, SQLite was chosen because I see the data we handle locally as "small data", where we'd delegate any "big data" operations to the cloud.
