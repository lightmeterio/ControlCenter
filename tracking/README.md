# Postfix log tracking

As known as RayChaser. Or Bruce Wayne.

## TODO

### Document the architecture of this beast.

### Simplify the publishing interface

The idea is to output `events` instead of `Results`.

The main issue is that `Results` is unreadable by humans and not extensible for the multi-server/relay support.

For instance, it's not able to represent the different hosts/queues a message went through.

In the new approach, the Tracker will receive as argument an `EventsListeners` that looks like more or less like this:

```go
type EventsListeners struct {
  DeliveryAttemptsListener interface{ PublishDeliveryAttemptEvent(DeliveryAttemptEvent) }
  ExpiredAttemptsListener interface{ PublishExpiredAttemptEvent(DeliveryAttemptEvent) }
  RelayedBounceListener interface{ RelayedBounceEvent(RelayedBounceEvent) }
  // ... and so on...
}
```

And the `*Event` objects are high level enough to be human readable, as well as making the `switch` of event types much easier,
as currently they all come in the form of a `Results` object.

## Debugging

You should build (or debug/run) the tracking code with the build flag `tracking_debug_util`,
which will cause each `action` to be executed in its own transaction.

```
-tags=tracking_debug_util,dev
```

By doing this you can immediately see the new database state after each action by opening the database
in a different process (sqlitebrowser, for instance).

This code is not enabled by default because it makes the tracking terribly slow. SQLite can only execute a
few transactions per second, and each `action` correspond to one line.

With transactions, on the other hand, we can process dozens of thousands of lines per second.
