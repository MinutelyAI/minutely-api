package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MinutelyAI/minutely-api/internal/database"
)

type MediaStateReq struct {
	MeetingID    string `json:"meeting_id"`
	Email        string `json:"email"` // Used to map the user to the participant row
	HasJoined    bool   `json:"has_joined"`
	AudioEnabled bool   `json:"audio_enabled"`
	VideoEnabled bool   `json:"video_enabled"`
}

// Update Participant Media State
func UpdateMediaState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req MediaStateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MeetingID == "" || req.Email == "" {
		http.Error(w, `{"error": "Invalid payload. Meeting ID and Email are required."}`, http.StatusBadRequest)
		return
	}

	// Upsert the participant's state. 
	var results []interface{}
	err := database.SupaClient.DB.From("meeting_participants").Upsert(map[string]interface{}{
		"meeting_id":    req.MeetingID,
		"email":         req.Email,
		"has_joined":    req.HasJoined,
		"audio_enabled": req.AudioEnabled,
		"video_enabled": req.VideoEnabled,
	}).Execute(&results)

	if err != nil {
		fmt.Println("🚨 SUPABASE MEDIA STATE ERROR:", err)
		http.Error(w, `{"error": "Failed to update media state"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Media state updated successfully"})
}