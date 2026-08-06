package main

import (
	"log"
	"net"

	"tcp-load-balancer/internal/proxy"
)

const (
	listenAddress  = ":8080"
	backendAddress = "127.0.0.1:9001"
)

func main() {
	listener, err := net.Listen("tcp", listenAddress)

	if err != nil {
		log.Fatalf("listen on %s: %v", listenAddress, err)
	}
	defer listener.Close()

	log.Printf("load balancer listening=%s backend=%s", listenAddress, backendAddress)

	if err := proxy.Serve(listener, backendAddress, log.Default()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
