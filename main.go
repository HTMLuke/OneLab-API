package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Response struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

func main() {
	http.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		res := Response{
			Message: "OneAPI running!",
			Status:  "OK",
		}
		json.NewEncoder(w).Encode(res)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		res := Response{
			Message: "Healthy",
			Status:  "UP",
		}
		json.NewEncoder(w).Encode(res)
	})

	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
