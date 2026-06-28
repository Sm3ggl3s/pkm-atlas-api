package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	address := ":" + port

	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API listening on http://localhost:%s", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
