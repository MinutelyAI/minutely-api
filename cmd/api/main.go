package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	// Make sure this path matches your actual module name from your go.mod file!
	"github.com/MinutelyAI/minutely-api/internal/database" 
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	res := Response{Status: "success", Message: "Minutely Go backend is fully operational!"}
	json.NewEncoder(w).Encode(res)
}

func main() {
	// 1. Initialize Supabase right when the app starts
	err := database.InitSupabase()
	if err != nil {
		// If it fails, the server will crash and print the error
		log.Fatalf("Database Error: %v", err) 
	}

	// 2. Start the server (same as before)
	http.HandleFunc("/api/health", healthCheck)
	port := "8080"
	address := "127.0.0.1:" + port

	fmt.Printf("Starting Minutely backend securely on http://%s\n", address)

	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Server failed to start: %v\n", err)
	}
}