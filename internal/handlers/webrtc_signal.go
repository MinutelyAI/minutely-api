package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type WebRTCSignalReq struct {
	MeetingID     string `json:"meeting_id"`
	FromEmail     string `json:"from_email"`
	ToEmail       string `json:"to_email"`
	Type          string `json:"type"`
	SDP           string `json:"sdp,omitempty"`
	Candidate     string `json:"candidate,omitempty"`
	SDPMid        string `json:"sdp_mid,omitempty"`
	SDPMLineIndex *int   `json:"sdp_mline_index,omitempty"`
}

type WebRTCSignalMessage struct {
	ID            int64  `json:"id"`
	MeetingID     string `json:"meeting_id"`
	FromEmail     string `json:"from_email"`
	ToEmail       string `json:"to_email"`
	Type          string `json:"type"`
	SDP           string `json:"sdp,omitempty"`
	Candidate     string `json:"candidate,omitempty"`
	SDPMid        string `json:"sdp_mid,omitempty"`
	SDPMLineIndex *int   `json:"sdp_mline_index,omitempty"`
}

type WebRTCPollResponse struct {
	Signals []WebRTCSignalMessage `json:"signals"`
}

var webrtcSignalStore = struct {
	sync.Mutex
	nextID   int64
	messages map[string][]WebRTCSignalMessage
}{
	nextID:   1,
	messages: map[string][]WebRTCSignalMessage{},
}

// SendWebRTCSignal stores a signal so another participant can fetch it.
func SendWebRTCSignal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req WebRTCSignalReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.MeetingID == "" || req.FromEmail == "" || req.ToEmail == "" || req.Type == "" {
		http.Error(w, `{"error": "meeting_id, from_email, to_email, and type are required"}`, http.StatusBadRequest)
		return
	}

	req.FromEmail = strings.ToLower(strings.TrimSpace(req.FromEmail))
	req.ToEmail = strings.ToLower(strings.TrimSpace(req.ToEmail))

	if req.Type != "offer" && req.Type != "answer" && req.Type != "ice-candidate" {
		http.Error(w, `{"error": "Invalid signal type"}`, http.StatusBadRequest)
		return
	}

	webrtcSignalStore.Lock()
	message := WebRTCSignalMessage{
		ID:            webrtcSignalStore.nextID,
		MeetingID:     req.MeetingID,
		FromEmail:     req.FromEmail,
		ToEmail:       req.ToEmail,
		Type:          req.Type,
		SDP:           req.SDP,
		Candidate:     req.Candidate,
		SDPMid:        req.SDPMid,
		SDPMLineIndex: req.SDPMLineIndex,
	}
	webrtcSignalStore.nextID++
	webrtcSignalStore.messages[req.MeetingID] = append(webrtcSignalStore.messages[req.MeetingID], message)

	// Keep memory bounded for long-lived meetings in dev mode.
	if len(webrtcSignalStore.messages[req.MeetingID]) > 1000 {
		webrtcSignalStore.messages[req.MeetingID] = webrtcSignalStore.messages[req.MeetingID][len(webrtcSignalStore.messages[req.MeetingID])-1000:]
	}
	webrtcSignalStore.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"message": "Signal stored"})
}

// PollWebRTCSignals returns pending signals for a participant after a known signal id.
func PollWebRTCSignals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	meetingID := r.URL.Query().Get("meeting_id")
	email := r.URL.Query().Get("email")
	sinceRaw := r.URL.Query().Get("since")
	if meetingID == "" || email == "" {
		http.Error(w, `{"error": "meeting_id and email are required"}`, http.StatusBadRequest)
		return
	}
	email = strings.ToLower(strings.TrimSpace(email))

	var since int64
	if sinceRaw != "" {
		parsed, err := strconv.ParseInt(sinceRaw, 10, 64)
		if err != nil {
			http.Error(w, `{"error": "Invalid since value"}`, http.StatusBadRequest)
			return
		}
		since = parsed
	}

	webrtcSignalStore.Lock()
	messages := webrtcSignalStore.messages[meetingID]
	result := make([]WebRTCSignalMessage, 0)
	for _, msg := range messages {
		if strings.EqualFold(msg.ToEmail, email) && msg.ID > since {
			result = append(result, msg)
		}
	}
	webrtcSignalStore.Unlock()

	json.NewEncoder(w).Encode(WebRTCPollResponse{Signals: result})
}
