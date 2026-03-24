package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MinutelyAI/minutely-api/internal/database"
)

type ThemeRequest struct {
	Theme string `json:"theme"`
}

type PreferenceRow struct {
	Theme string `json:"theme"`
}

// GetTheme fetches the user's saved theme from Supabase
func GetTheme(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var results []PreferenceRow
	err := database.SupaClient.DB.From("preferences").Select("theme").Eq("user_id", userID).Execute(&results)

	if err != nil {
		fmt.Println("🚨 SUPABASE GET THEME ERROR:", err)
		http.Error(w, `{"error": "Failed to fetch preferences"}`, http.StatusInternalServerError)
		return
	}

	// Default to light theme if no record exists yet
	if len(results) == 0 {
		json.NewEncoder(w).Encode(map[string]string{"theme": "light"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"theme": results[0].Theme})
}

// SaveTheme inserts or updates the user's theme in Supabase
func SaveTheme(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	if req.Theme != "light" && req.Theme != "dark" && req.Theme != "system" {
		http.Error(w, `{"error": "Invalid theme. Must be 'light', 'dark', or 'system'"}`, http.StatusBadRequest)
		return
	}

	// Upsert: Update if exists, Insert if new
	var results []interface{}
	err := database.SupaClient.DB.From("preferences").Upsert(map[string]interface{}{
		"user_id": userID,
		"theme":   req.Theme,
	}).Execute(&results)

	if err != nil {
		fmt.Println("🚨 SUPABASE SAVE THEME ERROR:", err)
		http.Error(w, `{"error": "Failed to save theme"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Theme updated to " + req.Theme,
	})
}