package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MinutelyAI/minutely-api/internal/database"
)

type EndMeetingReq struct {
	ID string `json:"id"`
}

// Create Instant Meeting & Generate Link
func CreateInstantMeeting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(UserIDKey).(string)

	// Generate a dynamic title
	title := fmt.Sprintf("Instant Meeting - %s", time.Now().Format("Jan 02, 3:04 PM"))

	// 1. Insert the meeting as 'in_progress'
	var insertedMeetings []map[string]interface{}
	err := database.SupaClient.DB.From("meetings").Insert(map[string]interface{}{
		"user_id": userID,
		"title":   title,
		"status":  "in_progress",
	}).Execute(&insertedMeetings)

	if err != nil || len(insertedMeetings) == 0 {
		fmt.Println("🚨 SUPABASE INSTANT MEETING ERROR:", err)
		http.Error(w, `{"error": "Failed to start meeting"}`, http.StatusInternalServerError)
		return
	}

	meeting := insertedMeetings[0]
	meetingID := meeting["id"].(string)

	// 2. Generate shareable Join Link & Code
	// We use the meeting ID as the unique room code for the frontend React Router
	joinLink := fmt.Sprintf("http://localhost:3000/#/join/%s", meetingID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Instant meeting started",
		"meeting":   meeting,
		"join_code": meetingID,
		"join_link": joinLink,
	})
}

//  End Meeting properly
func EndInstantMeeting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(UserIDKey).(string)
	var req EndMeetingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, `{"error": "Meeting ID is required"}`, http.StatusBadRequest)
		return
	}

	// Update status to 'completed'. 
	// Security: Eq("user_id", userID) ensures ONLY the host who created it can end it!
	var results []interface{}
	err := database.SupaClient.DB.From("meetings").Update(map[string]interface{}{
		"status": "completed",
	}).Eq("id", req.ID).Eq("user_id", userID).Execute(&results)

	if err != nil {
		fmt.Println("🚨 SUPABASE END MEETING ERROR:", err)
		http.Error(w, `{"error": "Failed to end meeting or unauthorized"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Meeting ended successfully for all participants"})
}