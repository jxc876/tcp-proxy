# History

## Waitgroups

Use a WaitGroup for bidirectional copy completion


`ProxyConnections` uses a `sync.WaitGroup` to wait for its two copy goroutines.
Each goroutine logs its own outcome and calls `Done()` when it finishes.

Reasoning

- The function only needs to know when both copy operations are complete. It does not otherwise consume or return their results, so a WaitGroup expresses the requirement more directly than a result type, buffered channel, and receive loop.

Trade-offs

- The implementation is smaller and its completion behavior is explicit.
- Results are logged concurrently rather than processed centrally.
- A channel may be preferable later if the function needs to aggregate or
  return errors, distinguish completion order, or coordinate cancellation.
