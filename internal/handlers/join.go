package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/MinutelyAI/minutely-api/internal/database"
)

// Validate Meeting Link
func ValidateMeeting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Extract the meeting ID from the query string: /api/meetings/validate?id=YOUR_ID
	meetingID := r.URL.Query().Get("id")
	if meetingID == "" {
		http.Error(w, `{"error": "Meeting ID is required"}`, http.StatusBadRequest)
		return
	}

	var results []map[string]interface{}
	err := database.SupaClient.DB.From("meetings").Select("id, title, status, scheduled_for").Eq("id", meetingID).Execute(&results)

	if err != nil || len(results) == 0 {
		http.Error(w, `{"error": "Invalid meeting link or meeting does not exist"}`, http.StatusNotFound)
		return
	}

	meeting := results[0]

	// Reject entry if the meeting was explicitly canceled
	if meeting["status"] == "canceled" {
		http.Error(w, `{"error": "This meeting has been canceled"}`, http.StatusForbidden)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Meeting is valid",
		"meeting": meeting,
	})
}