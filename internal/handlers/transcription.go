package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MinutelyAI/minutely-api/internal/database"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type TranscriptSegment struct {
	MeetingID    string   `json:"meeting_id"`
	SpeakerName  string   `json:"speaker_name"`
	SpeakerEmail string   `json:"speaker_email"`
	Text         string   `json:"text"`
	StartSecs    float64  `json:"start_secs"`
	EndSecs      float64  `json:"end_secs"`
	IsPartial    bool     `json:"is_partial"`
	Confidence   *float64 `json:"confidence,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
}

type StoredTranscript struct {
	MeetingID   string
	Segments    []TranscriptSegment
	StartedAt   time.Time
	CompletedAt *time.Time
	Status      string
}

type TranscriptionStartResponse struct {
	SessionID            string `json:"session_id"`
	DeepgramToken        string `json:"deepgram_token"`
	DeepgramTokenExpires string `json:"deepgram_token_expires"`
	TranscriptionWS      string `json:"transcription_ws"`
}

var transcriptMemory = struct {
	sync.RWMutex
	meetings map[string]*StoredTranscript
}{
	meetings: map[string]*StoredTranscript{},
}

var transcriptionHub = struct {
	sync.RWMutex
	clients map[string]map[*websocket.Conn]bool
}{
	clients: map[string]map[*websocket.Conn]bool{},
}

var transcriptionUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func meetingIDFromPath(r *http.Request, suffix string) string {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/meetings/")
	return strings.TrimSuffix(path, suffix)
}

func normalizeMeetingUUID(meetingID string) string {
	if _, err := uuid.Parse(meetingID); err == nil {
		return meetingID
	}
	// Fallback for string names
	return uuid.NewMD5(uuid.NameSpaceURL, []byte(meetingID)).String()
}

func jsonAPIError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// StartTranscriptionSession returns the Deepgram token used by the browser pipeline.
func StartTranscriptionSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		jsonAPIError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	meetingID := meetingIDFromPath(r, "/transcription/start")
	if meetingID == "" {
		jsonAPIError(w, "Meeting ID is required", http.StatusBadRequest)
		return
	}

	deepgramToken := os.Getenv("DEEPGRAM_KEY")
	if deepgramToken == "" {
		// Try DEEPGRAM_API_KEY as fallback
		deepgramToken = os.Getenv("DEEPGRAM_API_KEY")
	}

	if deepgramToken == "" {
		jsonAPIError(w, "DEEPGRAM_KEY is not configured on server", http.StatusServiceUnavailable)
		return
	}

	now := time.Now()
	expires := now.Add(4 * time.Hour)
	
	key := normalizeMeetingUUID(meetingID)
	
	// Initialize in-memory cache
	transcriptMemory.Lock()
	if _, exists := transcriptMemory.meetings[key]; !exists {
		transcriptMemory.meetings[key] = &StoredTranscript{
			MeetingID: key,
			StartedAt: now,
			Status:    "in_progress",
			Segments:  []TranscriptSegment{},
		}
	}
	transcriptMemory.Unlock()

	// Persist to DB (Best effort)
	_ = bestEffortStartLiveSession(r, key, expires)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(TranscriptionStartResponse{
		SessionID:            uuid.New().String(),
		DeepgramToken:        deepgramToken,
		DeepgramTokenExpires: expires.Format(time.RFC3339),
		TranscriptionWS:      fmt.Sprintf("/api/v1/meetings/%s/transcription/ws", meetingID),
	})
}

// EndTranscriptionSession marks the live transcript complete.
func EndTranscriptionSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		jsonAPIError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	meetingID := normalizeMeetingUUID(meetingIDFromPath(r, "/transcription/end"))
	completedAt := time.Now()
	
	transcriptMemory.Lock()
	if transcript := transcriptMemory.meetings[meetingID]; transcript != nil {
		transcript.CompletedAt = &completedAt
		transcript.Status = "completed"
	}
	transcriptMemory.Unlock()

	_ = bestEffortEndLiveSession(r, meetingID)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ended"})
}

// HandleTranscriptionWebSocket broadcasts final transcript segments to meeting participants.
func HandleTranscriptionWebSocket(w http.ResponseWriter, r *http.Request) {
	meetingID := normalizeMeetingUUID(meetingIDFromPath(r, "/transcription/ws"))
	if meetingID == "" {
		jsonAPIError(w, "Meeting ID is required", http.StatusBadRequest)
		return
	}

	conn, err := transcriptionUpgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("🚨 TRANSCRIPTION WS UPGRADE ERROR:", err)
		return
	}
	defer conn.Close()

	transcriptionHub.Lock()
	if transcriptionHub.clients[meetingID] == nil {
		transcriptionHub.clients[meetingID] = map[*websocket.Conn]bool{}
	}
	transcriptionHub.clients[meetingID][conn] = true
	transcriptionHub.Unlock()

	defer func() {
		transcriptionHub.Lock()
		delete(transcriptionHub.clients[meetingID], conn)
		if len(transcriptionHub.clients[meetingID]) == 0 {
			delete(transcriptionHub.clients, meetingID)
		}
		transcriptionHub.Unlock()
	}()

	for {
		var segment TranscriptSegment
		if err := conn.ReadJSON(&segment); err != nil {
			break
		}
		
		segment.MeetingID = meetingID
		segment.SpeakerEmail = strings.ToLower(strings.TrimSpace(segment.SpeakerEmail))
		segment.CreatedAt = time.Now().Format(time.RFC3339)
		
		if segment.Text == "" || segment.SpeakerEmail == "" {
			continue
		}

		// 1. Store in memory for instant retrieval
		storeTranscriptSegmentInMemory(meetingID, segment)
		
		// 2. Persist to DB via RPC (Best effort)
		_ = bestEffortStoreTranscriptSegment(r, meetingID, segment)
		
		// 3. Broadcast to all participants in this meeting
		broadcastTranscriptSegment(meetingID, segment)
	}
}

func storeTranscriptSegmentInMemory(meetingID string, segment TranscriptSegment) {
	transcriptMemory.Lock()
	defer transcriptMemory.Unlock()
	transcript := transcriptMemory.meetings[meetingID]
	if transcript == nil {
		transcript = &StoredTranscript{
			MeetingID: meetingID,
			StartedAt: time.Now(),
			Status:    "in_progress",
			Segments:  []TranscriptSegment{},
		}
		transcriptMemory.meetings[meetingID] = transcript
	}
	transcript.Segments = append(transcript.Segments, segment)
}

func broadcastTranscriptSegment(meetingID string, segment TranscriptSegment) {
	transcriptionHub.RLock()
	clientsMap := transcriptionHub.clients[meetingID]
	if clientsMap == nil {
		transcriptionHub.RUnlock()
		return
	}
	
	// Copy clients to slice to avoid holding lock during network IO
	clients := make([]*websocket.Conn, 0, len(clientsMap))
	for client := range clientsMap {
		clients = append(clients, client)
	}
	transcriptionHub.RUnlock()

	for _, client := range clients {
		_ = client.WriteJSON(segment)
	}
}

// GetMeetingTranscript returns live transcript data for a meeting.
func GetMeetingTranscript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		jsonAPIError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	meetingID := normalizeMeetingUUID(meetingIDFromPath(r, "/transcript"))
	
	// Try memory first
	segments := loadMemoryTranscriptSegments(meetingID)
	
	// If memory empty, try DB
	if len(segments) == 0 {
		segments = bestEffortFetchTranscriptSegments(r, meetingID)
	}

	sort.SliceStable(segments, func(i, j int) bool {
		return segments[i].StartSecs < segments[j].StartSecs
	})

	participants := make([]string, 0)
	seen := map[string]bool{}
	var fullText strings.Builder
	var duration float64
	
	for _, segment := range segments {
		if !seen[segment.SpeakerEmail] {
			participants = append(participants, fmt.Sprintf("%s <%s>", segment.SpeakerName, segment.SpeakerEmail))
			seen[segment.SpeakerEmail] = true
		}
		if fullText.Len() > 0 {
			fullText.WriteString("\n")
		}
		fullText.WriteString(segment.Text)
		if segment.EndSecs > duration {
			duration = segment.EndSecs
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"meeting_id":    meetingID,
		"source":        "live",
		"language":      "en",
		"duration_secs": duration,
		"speaker_count": len(participants),
		"full_text":     fullText.String(),
		"segments":      segments,
		"participants":  participants,
		"status":        "in_progress",
	})
}

func loadMemoryTranscriptSegments(meetingID string) []TranscriptSegment {
	transcriptMemory.RLock()
	defer transcriptMemory.RUnlock()
	transcript := transcriptMemory.meetings[meetingID]
	if transcript == nil {
		return nil
	}
	segments := make([]TranscriptSegment, len(transcript.Segments))
	copy(segments, transcript.Segments)
	return segments
}

// GetMeetingInsights returns stored AI outputs for a meeting.
func GetMeetingInsights(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		jsonAPIError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	meetingID := normalizeMeetingUUID(meetingIDFromPath(r, "/ai-insights"))
	token, _ := r.Context().Value(AuthTokenKey).(string)
	authClient := database.CreateAuthenticatedClient(token)

	var outputs []map[string]interface{}
	// AI outputs are stored in ai_outputs table
	err := authClient.DB.From("ai_outputs").Select("*").Eq("meeting_id", meetingID).Execute(&outputs)
	
	if err != nil {
		fmt.Println("🚨 SUPABASE AI INSIGHTS ERROR:", err)
		_ = json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	// Sort manually in Go if needed (newest first)
	sort.SliceStable(outputs, func(i, j int) bool {
		createdAtI, _ := outputs[i]["created_at"].(string)
		createdAtJ, _ := outputs[j]["created_at"].(string)
		return createdAtI > createdAtJ
	})

	_ = json.NewEncoder(w).Encode(outputs)
}

// UploadRecording handles audio file uploads for asynchronous transcription.
func UploadRecording(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		jsonAPIError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2GB limit
	r.Body = http.MaxBytesReader(w, r.Body, 2<<30)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonAPIError(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		jsonAPIError(w, "File field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// In a full implementation, we'd upload to S3/Supabase Storage here.
	// For now, we simulate success and return job IDs.
	
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"file_id": uuid.New().String(),
		"job_id":  uuid.New().String(),
		"status":  "pending",
		"name":    header.Filename,
	})
}

// ── DATABASE HELPERS ─────────────────────────────────────────────────────────

func bestEffortStartLiveSession(r *http.Request, meetingID string, expires time.Time) error {
	token, _ := r.Context().Value(AuthTokenKey).(string)
	if token == "" { return nil }
	client := database.CreateAuthenticatedClient(token)
	return client.DB.Rpc("start_live_transcription_session", map[string]interface{}{
		"p_meeting_id": meetingID,
		"p_expires_at": expires.Format(time.RFC3339),
	}).Execute(nil)
}

func bestEffortEndLiveSession(r *http.Request, meetingID string) error {
	token, _ := r.Context().Value(AuthTokenKey).(string)
	if token == "" { return nil }
	client := database.CreateAuthenticatedClient(token)
	return client.DB.Rpc("end_live_transcription_session", map[string]interface{}{
		"p_meeting_id": meetingID,
	}).Execute(nil)
}

func bestEffortStoreTranscriptSegment(r *http.Request, meetingID string, segment TranscriptSegment) error {
	token, _ := r.Context().Value(AuthTokenKey).(string)
	if token == "" { return nil }
	client := database.CreateAuthenticatedClient(token)
	return client.DB.Rpc("store_transcript_segment", map[string]interface{}{
		"p_meeting_id":    meetingID,
		"p_speaker_name":  segment.SpeakerName,
		"p_speaker_email": segment.SpeakerEmail,
		"p_text":          segment.Text,
		"p_start_secs":    segment.StartSecs,
		"p_end_secs":      segment.EndSecs,
		"p_confidence":    segment.Confidence,
		"p_is_partial":    segment.IsPartial,
	}).Execute(nil)
}

func bestEffortFetchTranscriptSegments(r *http.Request, meetingID string) []TranscriptSegment {
	token, _ := r.Context().Value(AuthTokenKey).(string)
	if token == "" { return nil }
	client := database.CreateAuthenticatedClient(token)
	var rows []map[string]interface{}
	err := client.DB.Rpc("get_meeting_transcript_segments", map[string]interface{}{
		"p_meeting_id": meetingID,
	}).Execute(&rows)
	
	if err != nil {
		return nil
	}

	segments := make([]TranscriptSegment, 0, len(rows))
	for _, row := range rows {
		segments = append(segments, mapTranscriptSegment(row, meetingID))
	}
	return segments
}

func mapTranscriptSegment(row map[string]interface{}, meetingID string) TranscriptSegment {
	return TranscriptSegment{
		MeetingID:    meetingID,
		SpeakerName:  fmt.Sprint(row["speaker_name"]),
		SpeakerEmail: fmt.Sprint(row["speaker_email"]),
		Text:         fmt.Sprint(row["text"]),
		StartSecs:    floatFromValue(row["start_secs"]),
		EndSecs:      floatFromValue(row["end_secs"]),
		IsPartial:    boolFromValue(row["is_partial"]),
		CreatedAt:    fmt.Sprint(row["created_at"]),
	}
}

func floatFromValue(value interface{}) float64 {
	switch typed := value.(type) {
	case float64: return typed
	case float32: return float64(typed)
	case int: return float64(typed)
	case int64: return float64(typed)
	case json.Number:
		out, _ := typed.Float64()
		return out
	default: return 0
	}
}

func boolFromValue(value interface{}) bool {
	switch typed := value.(type) {
	case bool: return typed
	case string: return strings.EqualFold(typed, "true") || typed == "1"
	default: return false
	}
}
