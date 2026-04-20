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

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
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

	mux := http.NewServeMux()

	// Register API routes
	mux.HandleFunc("/api/health", healthCheck)

	// Add the new Authentication routes
	mux.HandleFunc("/api/auth/signup", handlers.SignUp)
	mux.HandleFunc("/api/auth/login", handlers.Login)

	// Protected Routes (Wrapped in the RequireAuth middleware)
	mux.HandleFunc("/api/user/profile", handlers.RequireAuth(handlers.GetProfile))
	// Logout route (also protected)
	mux.HandleFunc("/api/auth/logout", handlers.Logout)

	// Dashboard / Meeting Routes (Protected by Auth)
	mux.HandleFunc("/api/meetings/next", handlers.RequireAuth(handlers.GetNextMeeting))
	mux.HandleFunc("/api/meetings/recent", handlers.RequireAuth(handlers.GetRecentMeetings))

	// Theme Routes (Protected by Auth)
	mux.HandleFunc("/api/preferences/theme", handlers.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/api/meetings/schedule", handlers.RequireAuth(handlers.CreateScheduledMeeting))
	mux.HandleFunc("/api/meetings/schedule/update", handlers.RequireAuth(handlers.UpdateScheduledMeeting))
	mux.HandleFunc("/api/meetings/schedule/cancel", handlers.RequireAuth(handlers.CancelScheduledMeeting))

	// Instant Meetings
	mux.HandleFunc("/api/meetings/instant", handlers.RequireAuth(handlers.CreateInstantMeeting))
	mux.HandleFunc("/api/meetings/end", handlers.RequireAuth(handlers.EndInstantMeeting))

	// Join Meeting Validation
	mux.HandleFunc("/api/meetings/validate", handlers.RequireAuth(handlers.ValidateMeeting))

	// Media State Sync
	mux.HandleFunc("/api/meetings/participant/state", handlers.RequireAuth(handlers.UpdateMediaState))

	port := "8080"
	address := "127.0.0.1:" + port

	fmt.Printf("Starting backend securely on http://%s\n", address)

	if err := http.ListenAndServe(address, withCORS(mux)); err != nil {
		log.Fatalf("Server failed to start: %v\n", err)
	}
}
