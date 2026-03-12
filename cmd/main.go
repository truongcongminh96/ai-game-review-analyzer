package main

import (
	"log"
	"net/http"
	"os"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/api"
)

func main() {
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("Server running at http://localhost:%s", port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
