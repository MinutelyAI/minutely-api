package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/MinutelyAI/minutely-api/internal/database"
)

type contextKey string
const UserIDKey contextKey = "userID"

type Meeting struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	ScheduledFor time.Time `json:"scheduled_for,omitempty"`
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func GetNextMeeting(w http.ResponseWriter, r *http.Request) {
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

	var results []Meeting
	
	// Fetch all scheduled meetings for this user without relying on library sorting
	err := database.SupaClient.DB.From("meetings").Select("*").Eq("user_id", userID).Eq("status", "scheduled").Execute(&results)

	if err != nil {
		fmt.Println("🚨 SUPABASE GET MEETING ERROR:", err)
		http.Error(w, `{"error": "Failed to fetch next meeting"}`, http.StatusInternalServerError)
		return
	}

	if len(results) == 0 {
		w.Write([]byte(`{"data": null, "message": "No upcoming meetings scheduled."}`))
		return
	}

	// Pure Go Sorting: Sort ascending (oldest dates first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].ScheduledFor.Before(results[j].ScheduledFor)
	})

	// Return just the first one (simulating Limit(1))
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": results[0],
	})
}

// StartInstantMeeting creates a new active meeting immediately
func StartInstantMeeting(w http.ResponseWriter, r *http.Request) {
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

	defaultTitle := "Instant Meeting - " + time.Now().Format("Jan 02, 3:04 PM")

	var results []Meeting
	err := database.SupaClient.DB.From("meetings").Insert(map[string]interface{}{
		"user_id":       userID,
		"title":         defaultTitle,
		"status":        "in_progress",
		"scheduled_for": time.Now(),
	}).Execute(&results)

	if err != nil || len(results) == 0 {
		fmt.Println("🚨 SUPABASE INSTANT MEETING ERROR:", err)
		http.Error(w, `{"error": "Failed to start meeting"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Meeting started",
		"data":    results[0],
	})
}

func GetRecentMeetings(w http.ResponseWriter, r *http.Request) {
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

	var results []Meeting
	
	// Fetch all completed meetings
	err := database.SupaClient.DB.From("meetings").Select("*").Eq("user_id", userID).Eq("status", "completed").Execute(&results)

	if err != nil {
		fmt.Println("🚨 SUPABASE RECENT MEETINGS ERROR:", err)
		http.Error(w, `{"error": "Failed to fetch recent history"}`, http.StatusInternalServerError)
		return
	}

	if len(results) == 0 {
		w.Write([]byte(`{"data": []}`))
		return
	}

	// Pure Go Sorting: Sort descending (newest dates first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].ScheduledFor.After(results[j].ScheduledFor)
	})

	// Slice the array to return only the first 5 items (simulating Limit(5))
	if len(results) > 5 {
		results = results[:5]
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": results,
	})
}