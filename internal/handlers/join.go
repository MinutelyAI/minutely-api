package handlers

import (
	"encoding/json"
	"net/http"
	"fmt"

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

	token := r.Context().Value(AuthTokenKey).(string)
	authClient := database.CreateAuthenticatedClient(token)

	var results []map[string]interface{}
	// Use RPC to bypass RLS for validation so guests with the link can see basic info
	err := authClient.DB.Rpc("get_meeting_by_id", map[string]interface{}{"m_id": meetingID}).Execute(&results)

	if err != nil {
		fmt.Println("🚨 SUPABASE VALIDATE MEETING ERROR:", err)
		http.Error(w, `{"error": "Failed to validate meeting"}`, http.StatusInternalServerError)
		return
	}

	if len(results) == 0 {
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