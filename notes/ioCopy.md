# How `io.Copy` Works

In Go, `io.Copy(destination, source)` transfers bytes from a source that implements
`io.Reader` to a destination that implements `io.Writer`.

Its signature is:

```go
func Copy(dst Writer, src Reader) (written int64, err error)
```

It returns the number of bytes successfully written and the first non-EOF error it
encounters.

## Conceptual implementation

At a high level, `io.Copy` repeatedly reads a chunk from the source and writes that
chunk to the destination:

```go
buffer := make([]byte, 32*1024)

for {
    n, readErr := source.Read(buffer)

    if n > 0 {
        written, writeErr := destination.Write(buffer[:n])

        if writeErr != nil {
            return totalWritten, writeErr
        }
        if written != n {
            return totalWritten, io.ErrShortWrite
        }

        totalWritten += int64(written)
    }

    if readErr == io.EOF {
        return totalWritten, nil
    }
    if readErr != nil {
        return totalWritten, readErr
    }
}
```

The real implementation contains additional safeguards and optimized paths, but
this is the central idea.

## Fast paths

Before using the generic buffered loop, `io.Copy` checks whether either endpoint
provides a specialized copy operation:

```go
if source implements io.WriterTo {
    return source.WriteTo(destination)
}

if destination implements io.ReaderFrom {
    return destination.ReadFrom(source)
}
```

These methods may avoid the generic buffer or use more efficient operating-system
facilities. The source's `WriterTo` method takes precedence over the destination's
`ReaderFrom` method.

## Important behavior

- The generic copy loop normally uses a 32 KiB buffer.
- Memory usage remains roughly constant regardless of the total amount copied.
- `io.EOF` from the source means successful completion, so it is returned as a
  `nil` error.
- A read is allowed to return both data and an error. `io.Copy` writes the returned
  data before handling the error.
- The byte count represents bytes successfully written, not merely bytes read.
- A write that reports fewer bytes than supplied without returning an error results
  in `io.ErrShortWrite`.
- `io.Copy` transfers an unstructured byte stream. It does not preserve or interpret
  packets, application messages, lines, or protocol frames.
- It does not add cancellation or timeouts. Those behaviors must be supplied by the
  reader or writer, such as through network deadlines or context-aware wrappers.

## Use in a TCP proxy

A TCP connection is full duplex, so a proxy usually runs one copy operation in each
direction:

```go
go io.Copy(clientConn, upstreamConn) // upstream -> client
io.Copy(upstreamConn, clientConn)    // client -> upstream
```

Each call blocks while it reads from one socket and writes to the other. Together,
the calls relay traffic in both directions until a connection closes or an error
occurs.

In production code, both copy operations commonly run in coordinated goroutines so
that completion or failure in one direction can trigger appropriate half-closing or
cleanup of the connections.

## Mental model

Think of `io.Copy` as a reusable pump:

```text
source (io.Reader) -> reusable byte buffer -> destination (io.Writer)
```

The pump does not need to know what the bytes mean. 

That is why the same function works with files, TCP connections, compressed streams, in-memory buffers, and many other Go I/O types.
