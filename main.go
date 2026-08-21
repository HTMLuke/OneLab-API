package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/HTMLuke/OneLab-API/auth"
	"github.com/HTMLuke/OneLab-API/config"
	"github.com/HTMLuke/OneLab-API/fileService"
	"github.com/HTMLuke/OneLab-API/secretProvider"
	"github.com/joho/godotenv"
)

type Response struct {
	Message      string            `json:"message"`
	Status       string            `json:"status"`
	Integrations map[string]string `json:"integrations,omitempty"`
}

func main() {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			log.Printf("Warning: failed to load .env: %v", err)
		}
	}

	// Initialize Config Service
	cfgService, err := config.NewConfigService()
	if err != nil {
		log.Printf("Warning: Failed to load config file (falling back to defaults & env): %v", err)
	}
	secretService := secretProvider.NewSecretService()
	mux := http.NewServeMux()

	// Build controllers and wire in every integration enabled in config
	authController := auth.NewAuthController()
	fController := fileService.NewFileController()
	registerIntegrations(cfgService, secretService, fController, authController)
	authController.RegisterRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		res := Response{
			Message: "Healthy",
			Status:  "UP",
		}
		json.NewEncoder(w).Encode(res)
	})

	fController.RegisterRoutes(mux, authController.Middleware)

	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
