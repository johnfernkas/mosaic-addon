package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/johnfernkas/mosaic-addon/internal/server"
)

func main() {
	port := os.Getenv("MOSAIC_PORT")
	if port == "" {
		port = "8176"
	}

	dataDir := os.Getenv("MOSAIC_DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}

	log.Println("🎨 Mosaic - LED Matrix Display Server")
	log.Printf("Port: %s, Data: %s", port, dataDir)

	srv, err := server.New(dataDir)
	if err != nil {
		log.Printf("Warning: failed to initialize full server: %v", err)
		log.Println("Starting with minimal server...")
		srv = server.NewSimple()
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Server listening on %s", addr)
	
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
