package main

import (
	"fmt"
	"os"

	"CRM/client"
	"CRM/server"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run . [server|client]")
		return
	}
	switch os.Args[1] {
	case "server":
		server.StartServer()
	case "client":
		client.StartClient()
	default:
		fmt.Println("Unknown command")
	}
}
