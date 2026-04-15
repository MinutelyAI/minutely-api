package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/MinutelyAI/minutely-api/internal/database"
	"github.com/MinutelyAI/minutely-api/internal/handlers"
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
	// Initialize Supabase
	err := database.InitSupabase()
	if err != nil {
		log.Fatalf("Database Error: %v", err)
	}

	// Register API routes
	http.HandleFunc("/api/health", healthCheck)

	// Add the new Authentication routes
	http.HandleFunc("/api/auth/signup", handlers.SignUp)
	http.HandleFunc("/api/auth/login", handlers.Login)

	// Protected Routes (Wrapped in the RequireAuth middleware)
	http.HandleFunc("/api/user/profile", handlers.RequireAuth(handlers.GetProfile))
	// Logout route (also protected)
	http.HandleFunc("/api/auth/logout", handlers.Logout)

	// Dashboard / Meeting Routes (Protected by Auth)
	http.HandleFunc("/api/meetings/next", handlers.RequireAuth(handlers.GetNextMeeting))
	http.HandleFunc("/api/meetings/recent", handlers.RequireAuth(handlers.GetRecentMeetings))

	// Theme Routes (Protected by Auth)
	http.HandleFunc("/api/preferences/theme", handlers.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetTheme(w, r)
		case http.MethodPost:
			handlers.SaveTheme(w, r)
		default:
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}))

	// Schedule Meeting Routes
	http.HandleFunc("/api/meetings/schedule", handlers.RequireAuth(handlers.CreateScheduledMeeting))
	http.HandleFunc("/api/meetings/schedule/update", handlers.RequireAuth(handlers.UpdateScheduledMeeting))
	http.HandleFunc("/api/meetings/schedule/cancel", handlers.RequireAuth(handlers.CancelScheduledMeeting))

	// Instant Meetings
	http.HandleFunc("/api/meetings/instant", handlers.RequireAuth(handlers.CreateInstantMeeting))
	http.HandleFunc("/api/meetings/end", handlers.RequireAuth(handlers.EndInstantMeeting))

	// Join Meeting Validation
	http.HandleFunc("/api/meetings/validate", handlers.RequireAuth(handlers.ValidateMeeting))

	// Media State Sync
	http.HandleFunc("/api/meetings/participant/state", handlers.RequireAuth(handlers.UpdateMediaState))

	port := "8080"
	address := "127.0.0.1:" + port

	fmt.Printf("Starting backend securely on http://%s\n", address)

	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Server failed to start: %v\n", err)
	}
}
