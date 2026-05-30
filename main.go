package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/HTMLuke/OneLab-API/config"
	"github.com/HTMLuke/OneLab-API/fileService"
)

type Response struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

func main() {
	// Initialize Config Service
	cfgService, err := config.NewConfigService("config.json")
	if err != nil {
		log.Printf("Warning: Failed to load config file (falling back to defaults & env): %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		res := Response{
			Message: "OneAPI running!",
			Status:  "OK",
		}
		json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		res := Response{
			Message: "Healthy",
			Status:  "UP",
		}
		json.NewEncoder(w).Encode(res)
	})

	// Initialize App Controller and specify exactly what Services are available.
	fController := fileService.NewFileController()

	// Inject integrations securely configured with their base URLs and credentials from the config service
	fController.AddIntegration("nextcloud", fileService.NewNextcloudService(
		cfgService.GetNextcloudURL(),
		cfgService.GetNextcloudUser(),
		cfgService.GetNextcloudPassword(),
	))
	fController.AddIntegration("paperless", fileService.NewPaperlessService(
		cfgService.GetPaperlessURL(),
		cfgService.GetPaperlessToken(),
	))

	fController.RegisterRoutes(mux)

	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
