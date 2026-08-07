# Goroutines & Channels

A few notes on goroutines :)

## Goroutines

A goroutine runs a function concurrently with the goroutine that started it:

```go
go copyStream(backend, client, "client-to-backend")
```

The `go` statement starts `copyStream` and then immediately allows the calling
goroutine to continue. 

There is no automatic return-value handoff or wait for a
goroutine to finish. 

## Channels 

Channels are one way to communicate results and coordinate
completion.

The arrow shows the direction in which a value moves:

```go
results <- value   // Send (publish) a value into the channel
result := <-results // Receive a value from the channel
```

A receive blocks when no value is available. It resumes when another goroutine
sends a value.

Whether a send blocks depends on whether the channel is *buffered* and whether
the buffer has *available capacity*.

## Unbuffered channels

Create an unbuffered channel by omitting a capacity:

```go
results := make(chan copyResult)
```

An unbuffered channel performs a direct handoff:

- A send blocks until another goroutine is ready to receive the value.
- A receive blocks until another goroutine is ready to send a value.
- The sender and receiver synchronize for every value transferred.

For example:

```go
results <- copyResult{direction: direction, err: copyErr}
```

The sending goroutine cannot continue or exit until another goroutine executes
a matching receive:

```go
result := <-results
```

## Buffered channels

Create a buffered channel by specifying its capacity:

```go
const copyOperations = 2
results := make(chan copyResult, copyOperations)
```

A buffered channel acts like a small mailbox:

- A send completes immediately while the buffer has available capacity.
- A send blocks when the buffer is full, until a receiver removes a value.
- A receive completes immediately while the buffer contains a value.
- A receive blocks when the buffer is empty, until a sender provides a value.

Buffering therefore allows a producer to publish a value and continue—or
exit—without waiting for a consumer to receive that specific value immediately.
This is true only while capacity remains available.


## Example

The proxy in this project uses a channel to communicate results.

The proxy starts two copy operations, one for each direction:

```go
go copyStream(backend, client, "client-to-backend")
go copyStream(client, backend, "backend-to-client")
```

Each copy goroutine publishes exactly one result when it finishes:

```go
results <- copyResult{direction: direction, err: copyErr}
```

The calling goroutine receives exactly two results:

```go
for i := 0; i < copyOperations; i++ {
	result := <-results
	// Handle result...
}
```

The results arrive in completion order. This may differ from the order in
which the goroutines were started. If neither copy has finished, the receive
blocks. After one finishes, its result becomes available and the loop then
waits for the other operation.

The buffer capacity equals the number of producers, and each producer sends
once. Consequently, both copy goroutines can publish their results and exit,
even if the receiving loop has not read either result yet.

An *unbuffered channel* would also work in this particular design because the
calling goroutine immediately receives twice. The difference is that each copy
goroutine would wait at its send until the receiving loop was ready. Here,
buffering mainly reduces synchronization between the workers and receiver; it
is unlikely to provide a meaningful performance improvement.


## Rules to remember

- Unbuffered channel — sender and receiver must meet for the transfer
- Buffered channel — a sender can proceed until the buffer is full
- Any receive blocks while no value is available
- Buffer capacity is not a limit on the total number of values ever sent 
- Buffer capacity is the number of unread values the channel can hold at once
- Starting a goroutine does not automatically wait for it or collect its result
- WaitGroups coordinate completion but do not carry results or errors
- Ensure every expected send has a receiver
- Ensure every expected receive can eventually get a value
- Otherwise the program may deadlock


## WaitGroups

A `sync.WaitGroup` lets one goroutine wait for a group of other goroutines to
finish. It acts like a thread-safe countdown counter:

```go
var wg sync.WaitGroup

wg.Add(2)

go func() {
	defer wg.Done()
	copyClientToBackend()
}()

go func() {
	defer wg.Done()
	copyBackendToClient()
}()

wg.Wait()
```

The lifecycle is:

1. `Add(2)` registers two operations that must finish.
2. Each operation calls `Done()` when it finishes, reducing the counter by one.
3. `Wait()` blocks until the counter reaches zero.

Calling `Done()` with `defer` ensures it runs whenever the goroutine returns:

```go
go func() {
	defer wg.Done()
	// Do work...
}()
```

Call `Add` before starting the goroutine. Calling it inside the goroutine can
create a race in which `Wait()` sees a counter of zero and returns too early:

```go
wg.Add(1)
go func() {
	defer wg.Done()
	// Do work...
}()
```

A WaitGroup communicates only that all registered operations have finished. It
does not communicate results, errors, which operation finished first, or the
order in which operations finished.

Use a WaitGroup when the goal is simply to wait for all operations. Use a
channel when the operations need to send values or errors. A channel can also
provide completion information when each goroutine sends exactly one result,
as it does in this proxy.

Common mistakes include:

- Forgetting `Done()`, which can cause `Wait()` to block forever
- Calling `Done()` too many times, which causes a negative-counter panic
- Calling `Add` inside the new goroutine instead of before starting it
- Expecting a WaitGroup to make other shared data safe; shared data may still
  require a mutex or channel
- Expecting a WaitGroup to cancel goroutines; it only waits for them
