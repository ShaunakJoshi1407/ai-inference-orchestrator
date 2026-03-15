package main

import (
	"log"
)

func main() {

	server := NewServer()

	log.Println("Starting MCP server on :8085")

	err := server.Start(":8085")
	if err != nil {
		log.Fatal(err)
	}

}
