package main

import (
	"log"
	"net/http"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/api"
)

func main() {
	mux := http.NewServeMux()

	api.RegisterRoutes(mux)

	log.Println("Server is running at http://localhost:8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
