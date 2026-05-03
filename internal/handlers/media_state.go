package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MinutelyAI/minutely-api/internal/database"
)

type MediaStateReq struct {
	MeetingID    string `json:"meeting_id"`
	Email        string `json:"email"` // Used to map the user to the participant row
	HasJoined    bool   `json:"has_joined"`
	AudioEnabled bool   `json:"audio_enabled"`
	VideoEnabled bool   `json:"video_enabled"`
}

type MeetingParticipantsResponse struct {
	Participants []map[string]interface{} `json:"participants"`
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

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	token := r.Context().Value(AuthTokenKey).(string)
	authClient := database.CreateAuthenticatedClient(token)

	// Use the new RPC to update state reliably (bypasses RLS issues as it's SECURITY DEFINER)
	err := authClient.DB.Rpc("sync_participant_state", map[string]interface{}{
		"p_meeting_id": req.MeetingID,
		"p_email":      req.Email,
		"p_joined":     req.HasJoined,
		"p_audio":      req.AudioEnabled,
		"p_video":      req.VideoEnabled,
	}).Execute(nil)

	if err != nil {
		fmt.Printf(
			"SUPABASE MEDIA STATE SYNC ERROR meeting_id=%s email=%s joined=%t audio=%t video=%t err=%v\n",
			req.MeetingID,
			req.Email,
			req.HasJoined,
			req.AudioEnabled,
			req.VideoEnabled,
			err,
		)
		http.Error(w, `{"error": "Failed to update media state"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Media state updated successfully"})
}

// GetMeetingParticipants returns joined participants for a meeting.
func GetMeetingParticipants(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	meetingID := r.URL.Query().Get("id")
	if meetingID == "" {
		http.Error(w, `{"error": "Meeting ID is required"}`, http.StatusBadRequest)
		return
	}

	token := r.Context().Value(AuthTokenKey).(string)
	authClient := database.CreateAuthenticatedClient(token)

	var results []map[string]interface{}
	err := authClient.DB.From("meeting_participants").
		Select("email,has_joined,audio_enabled,video_enabled").
		Eq("meeting_id", meetingID).
		Execute(&results)

	if err != nil {
		fmt.Println("🚨 SUPABASE PARTICIPANT FETCH ERROR:", err)
		http.Error(w, `{"error": "Failed to fetch participants"}`, http.StatusInternalServerError)
		return
	}

	joined := make([]map[string]interface{}, 0, len(results))
	for _, participant := range results {
		hasJoined := false
		switch value := participant["has_joined"].(type) {
		case bool:
			hasJoined = value
		case string:
			hasJoined = strings.EqualFold(value, "true") || value == "1" || strings.EqualFold(value, "t")
		case float64:
			hasJoined = value != 0
		}

		if hasJoined {
			joined = append(joined, participant)
		}
	}

	json.NewEncoder(w).Encode(MeetingParticipantsResponse{Participants: joined})
}
