package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MinutelyAI/minutely-api/internal/database"
)

// Request Models
type ScheduleMeetingReq struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	ScheduledFor string   `json:"scheduled_for"` // e.g., "2026-04-15T10:00:00Z"
	Participants []string `json:"participants"`
}

type UpdateMeetingReq struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ScheduledFor string `json:"scheduled_for"`
}

type CancelMeetingReq struct {
	ID string `json:"id"`
}

// Helper function to remove duplicate emails
func uniqueEmails(emails []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range emails {
		cleanEmail := strings.ToLower(strings.TrimSpace(entry))
		if cleanEmail != "" && !keys[cleanEmail] {
			keys[cleanEmail] = true
			list = append(list, cleanEmail)
		}
	}
	return list
}

// Create Scheduled Meeting & Invite Participants
func CreateScheduledMeeting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(UserIDKey).(string)

	var req ScheduleMeetingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.ScheduledFor == "" {
		http.Error(w, `{"error": "Invalid payload. Title and scheduled_for are required."}`, http.StatusBadRequest)
		return
	}

	// 1. Insert the meeting
	var insertedMeetings []map[string]interface{}
	err := database.SupaClient.DB.From("meetings").Insert(map[string]interface{}{
		"user_id":       userID,
		"title":         req.Title,
		"description":   req.Description,
		"status":        "scheduled",
		"scheduled_for": req.ScheduledFor,
	}).Execute(&insertedMeetings)

	if err != nil || len(insertedMeetings) == 0 {
		fmt.Println("🚨 SUPABASE CREATE MEETING ERROR:", err)
		http.Error(w, `{"error": "Failed to schedule meeting"}`, http.StatusInternalServerError)
		return
	}

	meetingID := insertedMeetings[0]["id"].(string)

	// 2. Insert participants (if any)
	cleanParticipants := uniqueEmails(req.Participants)
	if len(cleanParticipants) > 0 {
		var participantsData []interface{}
		for _, email := range cleanParticipants {
			participantsData = append(participantsData, map[string]interface{}{
				"meeting_id": meetingID,
				"email":      email,
			})
		}
		
		// Bulk insert participants
		err = database.SupaClient.DB.From("meeting_participants").Insert(participantsData).Execute(nil)
		if err != nil {
			fmt.Println("🚨 SUPABASE PARTICIPANT INSERT ERROR:", err)
			// Non-fatal error: The meeting was created, but invites failed.
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Meeting scheduled successfully",
		"meeting": insertedMeetings[0],
		"invited": len(cleanParticipants),
	})
}

// --- US-05.3: Edit Meeting ---
func UpdateScheduledMeeting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPut {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(UserIDKey).(string)
	var req UpdateMeetingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, `{"error": "Meeting ID is required"}`, http.StatusBadRequest)
		return
	}

	// Update the meeting (Supabase RLS is bypassed by Service Role, so we MUST enforce Eq("user_id", userID))
	var results []interface{}
	err := database.SupaClient.DB.From("meetings").Update(map[string]interface{}{
		"title":         req.Title,
		"description":   req.Description,
		"scheduled_for": req.ScheduledFor,
	}).Eq("id", req.ID).Eq("user_id", userID).Execute(&results)

	if err != nil {
		http.Error(w, `{"error": "Failed to update meeting or unauthorized"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Meeting updated successfully"})
}

// Cancel Meeting
func CancelScheduledMeeting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost { // Using POST for an action endpoint
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(UserIDKey).(string)
	var req CancelMeetingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, `{"error": "Meeting ID is required"}`, http.StatusBadRequest)
		return
	}

	// Mark status as 'canceled'
	var results []interface{}
	err := database.SupaClient.DB.From("meetings").Update(map[string]interface{}{
		"status": "canceled",
	}).Eq("id", req.ID).Eq("user_id", userID).Execute(&results)

	if err != nil {
		http.Error(w, `{"error": "Failed to cancel meeting or unauthorized"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Meeting canceled successfully"})
}