package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

const testTimeout = 3 * time.Second

func TestServeCopiesArbitraryBytesInBothDirections(t *testing.T) {
	backend := listenTCP(t)
	defer backend.Close()

	backendDone := make(chan error, 1)
	go func() {
		connection, err := backend.AcceptTCP()
		if err != nil {
			backendDone <- err
			return
		}
		defer connection.Close()

		payload := make([]byte, 8)
		if _, err := io.ReadFull(connection, payload); err != nil {
			backendDone <- err
			return
		}
		_, err = connection.Write(append([]byte("reply:"), payload...))
		backendDone <- err
	}()

	proxyAddress, stopProxy := startProxy(t, backend.Addr().String())
	defer stopProxy()

	client := dialTCP(t, proxyAddress)
	defer client.Close()
	setDeadline(t, client)

	payload := []byte{0x00, 0x01, 0x7f, 0x80, 0xfe, 0xff, 'G', 'o'}
	if _, err := client.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	response := make([]byte, len("reply:")+len(payload))
	if _, err := io.ReadFull(client, response); err != nil {
		t.Fatalf("read response: %v", err)
	}
	if want := append([]byte("reply:"), payload...); !bytes.Equal(response, want) {
		t.Fatalf("response = %v, want %v", response, want)
	}

	if err := <-backendDone; err != nil {
		t.Fatalf("backend: %v", err)
	}
}

func TestServeKeepsConcurrentClientStreamsSeparate(t *testing.T) {
	backend := listenTCP(t)
	defer backend.Close()

	backendDone := make(chan struct{})
	go func() {
		defer close(backendDone)
		var handlers sync.WaitGroup
		for i := 0; i < 12; i++ {
			connection, err := backend.AcceptTCP()
			if err != nil {
				return
			}
			handlers.Add(1)
			go func() {
				defer handlers.Done()
				defer connection.Close()
				_, _ = io.Copy(connection, connection)
			}()
		}
		handlers.Wait()
	}()

	proxyAddress, stopProxy := startProxy(t, backend.Addr().String())
	defer stopProxy()

	var clients sync.WaitGroup
	errors := make(chan error, 12)
	for i := 0; i < 12; i++ {
		clients.Add(1)
		go func(id int) {
			defer clients.Done()
			client, err := net.DialTCP("tcp", nil, resolveForTest(proxyAddress))
			if err != nil {
				errors <- err
				return
			}
			defer client.Close()
			_ = client.SetDeadline(time.Now().Add(testTimeout))

			message := []byte(fmt.Sprintf("client-%02d", id))
			if _, err := client.Write(message); err != nil {
				errors <- err
				return
			}
			response := make([]byte, len(message))
			if _, err := io.ReadFull(client, response); err != nil {
				errors <- err
				return
			}
			if !bytes.Equal(response, message) {
				errors <- fmt.Errorf("client %d received %q, want %q", id, response, message)
			}
		}(i)
	}
	clients.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Error(err)
		}
	}
	waitFor(t, backendDone, "concurrent backend handlers")
}

func TestClientHalfCloseAllowsFinalBackendResponse(t *testing.T) {
	backend := listenTCP(t)
	defer backend.Close()

	backendDone := make(chan error, 1)
	go func() {
		connection, err := backend.AcceptTCP()
		if err != nil {
			backendDone <- err
			return
		}
		defer connection.Close()

		request, err := io.ReadAll(connection)
		if err == nil {
			_, err = connection.Write(append([]byte("after-eof:"), request...))
		}
		backendDone <- err
	}()

	proxyAddress, stopProxy := startProxy(t, backend.Addr().String())
	defer stopProxy()

	client := dialTCP(t, proxyAddress)
	defer client.Close()
	setDeadline(t, client)

	if _, err := client.Write([]byte("request")); err != nil {
		t.Fatalf("write request: %v", err)
	}
	if err := client.CloseWrite(); err != nil {
		t.Fatalf("half-close client: %v", err)
	}

	response, err := io.ReadAll(client)
	if err != nil {
		t.Fatalf("read final response: %v", err)
	}
	if want := "after-eof:request"; string(response) != want {
		t.Fatalf("response = %q, want %q", response, want)
	}
	if err := <-backendDone; err != nil {
		t.Fatalf("backend: %v", err)
	}
}

func TestBackendClosurePropagatesToClient(t *testing.T) {
	backend := listenTCP(t)
	defer backend.Close()

	go func() {
		connection, err := backend.AcceptTCP()
		if err != nil {
			return
		}
		_, _ = connection.Write([]byte("goodbye"))
		_ = connection.Close()
	}()

	connectionClosed := newLogObserver("connection closed")
	proxyAddress, stopProxy := startProxy(t, backend.Addr().String(), log.New(connectionClosed, "", 0))
	defer stopProxy()

	client := dialTCP(t, proxyAddress)
	defer client.Close()
	setDeadline(t, client)

	response, err := io.ReadAll(client)
	if err != nil {
		t.Fatalf("read until backend closure: %v", err)
	}
	if string(response) != "goodbye" {
		t.Fatalf("response = %q, want %q", response, "goodbye")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	waitFor(t, connectionClosed.seen, "proxy connection handler")
}

func TestBackendDialFailureDoesNotStopListener(t *testing.T) {
	unusedAddress := unusedTCPAddress(t)
	proxyAddress, stopProxy := startProxy(t, unusedAddress)
	defer stopProxy()

	for attempt := 1; attempt <= 2; attempt++ {
		client := dialTCP(t, proxyAddress)
		setDeadline(t, client)
		buffer := make([]byte, 1)
		_, err := client.Read(buffer)
		_ = client.Close()
		if err == nil {
			t.Fatalf("attempt %d: expected client connection to close", attempt)
		}
	}
}

func listenTCP(t *testing.T) *net.TCPListener {
	t.Helper()
	listener, err := net.ListenTCP("tcp", resolveForTest("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return listener
}

func startProxy(t *testing.T, backendAddress string, loggers ...*log.Logger) (string, func()) {
	t.Helper()
	listener := listenTCP(t)
	serveDone := make(chan error, 1)
	logger := log.New(io.Discard, "", 0)
	if len(loggers) > 0 {
		logger = loggers[0]
	}
	go func() {
		serveDone <- Serve(listener, backendAddress, logger)
	}()

	stop := func() {
		_ = listener.Close()
		select {
		case err := <-serveDone:
			if err != nil {
				t.Errorf("proxy serve: %v", err)
			}
		case <-time.After(testTimeout):
			t.Error("proxy did not stop after listener closed")
		}
	}
	return listener.Addr().String(), stop
}

type logObserver struct {
	target []byte
	once   sync.Once
	seen   chan struct{}
}

func newLogObserver(target string) *logObserver {
	return &logObserver{target: []byte(target), seen: make(chan struct{})}
}

func (observer *logObserver) Write(message []byte) (int, error) {
	if bytes.Contains(message, observer.target) {
		observer.once.Do(func() { close(observer.seen) })
	}
	return len(message), nil
}

func dialTCP(t *testing.T, address string) *net.TCPConn {
	t.Helper()
	connection, err := net.DialTCP("tcp", nil, resolveForTest(address))
	if err != nil {
		t.Fatalf("dial %s: %v", address, err)
	}
	return connection
}

func resolveForTest(address string) *net.TCPAddr {
	resolved, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		panic(err)
	}
	return resolved
}

func setDeadline(t *testing.T, connection *net.TCPConn) {
	t.Helper()
	if err := connection.SetDeadline(time.Now().Add(testTimeout)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
}

func unusedTCPAddress(t *testing.T) string {
	t.Helper()
	listener := listenTCP(t)
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close temporary listener: %v", err)
	}
	return address
}

func waitFor(t *testing.T, done <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", description)
	}
}
