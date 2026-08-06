# About

A minimal TCP (Layer 4) load balancer written as a learning exercise. 

It accepts incoming TCP connections and proxies to a local backend. 

Raw bytes are copied in both directions. 

The destinations are intentionally hardcoded in this version:
- The load balancer listes for incoming connections on `:8080`
- The demo backend runs on `127.0.0.1:9001` and simply echos back


## Try it

Terminal 1 — starts the a demo backend on port `9001`:

```sh
go run ./cmd/demo-backend
```

Terminal 2 — starts the load balancer and proxies to the backend:

```sh
go run ./cmd/load-balancer
```

Terminal 3 — connects a TCP client using netcat:

```sh
nc 127.0.0.1 8080
```

Type a line and press Enter, Press Control-C to stop `nc` and the two servers.

The response should be prefixed with `demo-backend:`. 

This shows that the bytes traveled through the load balancer to the backend and back. 



## Run the tests

```sh
go test ./...
go test -race ./...
```

The integration tests use real loopback TCP sockets to cover various scenarios:
- arbitrary byte forwarding
- concurrent connections
- half-closes
- backend closure
- backend dial failures


## Implementation Notes

Key files:

- [load-balancer](./load-balancer/main.go) — listens on `:8080`
- [demo-backend](./cmd/demo-backend/main.go) — demo backend, runs on `:9001`
- [proxy.go](./internal/proxy/proxy.go) — concurrent bidirectional copying

The proxy uses two concurrent `io.Copy` operations:
* Client → backend
* Backend → client

When either copy reaches EOF, `CloseWrite()` is called. 

The reverse stream is allowed to finish to avoid truncating a final response.


## Future Enhancements

This first milestone intentionally has one hardcoded backend. 

Other features are defferend, ex:
- Backend pools
- Balancing algorithms (ex: Round Robin)
- Health checks
- Configurable timeouts
- Limits
- Metrics
- Graceful process draining
- HTTP-aware routing
