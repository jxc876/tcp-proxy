package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

const listenAddress = ":9001"

func main() {
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("listen on %s: %v", listenAddress, err)
	}
	defer listener.Close()

	log.Printf("demo backend listening=%s", listenAddress)
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Fatalf("accept: %v", err)
		}
		go handleConnection(connection)
	}
}

func handleConnection(connection net.Conn) {
	defer connection.Close()
	log.Printf("client connected remote=%s", connection.RemoteAddr())

	scanner := bufio.NewScanner(connection)
	for scanner.Scan() {
		if _, err := fmt.Fprintf(connection, "demo-backend: %s\n", scanner.Text()); err != nil {
			log.Printf("write remote=%s error=%v", connection.RemoteAddr(), err)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("read remote=%s error=%v", connection.RemoteAddr(), err)
	}
}
