// Package proxy implements a small, protocol-agnostic TCP proxy.
package proxy

import (
	"errors"
	"io"
	"log"
	"net"
)

// Serve accepts TCP clients from listener and proxies each one to backendAddress.
// Each client is handled concurrently in its own goroutine.
func Serve(listener net.Listener, backendAddress string, logger *log.Logger) error {
	logger = loggerOrDefault(logger)

	for {
		connection, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		client, ok := connection.(*net.TCPConn)
		if !ok {
			logger.Printf("rejecting non-TCP connection from %s", connection.RemoteAddr())
			_ = connection.Close()
			continue
		}

		logger.Printf("accepted client=%s", client.RemoteAddr())
		go connectAndProxy(client, backendAddress, logger)
	}
}

func connectAndProxy(client *net.TCPConn, backendAddress string, logger *log.Logger) {
	resolvedBackend, err := net.ResolveTCPAddr("tcp", backendAddress)
	if err != nil {
		logger.Printf("backend address invalid client=%s backend=%s error=%v", client.RemoteAddr(), backendAddress, err)
		_ = client.Close()
		return
	}

	backend, err := net.DialTCP("tcp", nil, resolvedBackend)
	if err != nil {
		logger.Printf("backend dial failed client=%s backend=%s error=%v", client.RemoteAddr(), backendAddress, err)
		_ = client.Close()
		return
	}

	logger.Printf("connection established client=%s backend=%s", client.RemoteAddr(), backend.RemoteAddr())
	ProxyConnections(client, backend, logger)
	logger.Printf("connection closed client=%s backend=%s", client.RemoteAddr(), backend.RemoteAddr())
}

// ProxyConnections copies bytes concurrently in both directions. When one
// source reaches EOF, it half-closes the destination's write side so the
// reverse direction can finish sending any remaining data.
func ProxyConnections(client, backend *net.TCPConn, logger *log.Logger) {
	logger = loggerOrDefault(logger)
	defer client.Close()
	defer backend.Close()

	type copyResult struct {
		direction string
		err       error
	}

	results := make(chan copyResult, 2)
	copyStream := func(destination, source *net.TCPConn, direction string) {
		_, copyErr := io.Copy(destination, source)
		closeErr := destination.CloseWrite()
		if copyErr == nil {
			copyErr = closeErr
		}
		results <- copyResult{direction: direction, err: copyErr}
	}

	go copyStream(backend, client, "client-to-backend")
	go copyStream(client, backend, "backend-to-client")

	for i := 0; i < 2; i++ {
		result := <-results
		if result.err != nil {
			logger.Printf("copy stopped direction=%s error=%v", result.direction, result.err)
		}
	}
}

func loggerOrDefault(logger *log.Logger) *log.Logger {
	if logger == nil {
		return log.Default()
	}
	return logger
}
