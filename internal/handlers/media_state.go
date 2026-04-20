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

	var existing []map[string]interface{}
	err := database.SupaClient.DB.From("meeting_participants").
		Select("meeting_id,email").
		Eq("meeting_id", req.MeetingID).
		Eq("email", req.Email).
		Execute(&existing)
	if err != nil {
		fmt.Println("🚨 SUPABASE MEDIA STATE LOOKUP ERROR:", err)
		http.Error(w, `{"error": "Failed to update media state"}`, http.StatusInternalServerError)
		return
	}

	payload := map[string]interface{}{
		"meeting_id":    req.MeetingID,
		"email":         req.Email,
		"has_joined":    req.HasJoined,
		"audio_enabled": req.AudioEnabled,
		"video_enabled": req.VideoEnabled,
	}

	if len(existing) == 0 {
		var inserted []interface{}
		err = database.SupaClient.DB.From("meeting_participants").Insert(payload).Execute(&inserted)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				var updated []interface{}
				updateErr := database.SupaClient.DB.From("meeting_participants").Update(map[string]interface{}{
					"has_joined":    req.HasJoined,
					"audio_enabled": req.AudioEnabled,
					"video_enabled": req.VideoEnabled,
				}).Eq("meeting_id", req.MeetingID).Eq("email", req.Email).Execute(&updated)
				if updateErr == nil {
					json.NewEncoder(w).Encode(map[string]string{"message": "Media state updated successfully"})
					return
				}

				fmt.Println("🚨 SUPABASE MEDIA STATE DUPLICATE RECOVERY ERROR:", updateErr)
				http.Error(w, `{"error": "Failed to update media state"}`, http.StatusInternalServerError)
				return
			}

			fmt.Println("🚨 SUPABASE MEDIA STATE INSERT ERROR:", err)
			http.Error(w, `{"error": "Failed to update media state"}`, http.StatusInternalServerError)
			return
		}
	} else {
		var updated []interface{}
		err = database.SupaClient.DB.From("meeting_participants").Update(map[string]interface{}{
			"has_joined":    req.HasJoined,
			"audio_enabled": req.AudioEnabled,
			"video_enabled": req.VideoEnabled,
		}).Eq("meeting_id", req.MeetingID).Eq("email", req.Email).Execute(&updated)
		if err != nil {
			fmt.Println("🚨 SUPABASE MEDIA STATE UPDATE ERROR:", err)
			http.Error(w, `{"error": "Failed to update media state"}`, http.StatusInternalServerError)
			return
		}
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

	var results []map[string]interface{}
	err := database.SupaClient.DB.From("meeting_participants").
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