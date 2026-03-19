package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/MinutelyAI/minutely-api/internal/database"
	"github.com/nedpals/supabase-go"
)


type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignUp handles new user registration
func SignUp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Send credentials to Supabase
	ctx := context.Background()
	user, err := database.SupaClient.Auth.SignUp(ctx, supabase.UserCredentials{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "User created successfully",
		"user_id": user.ID,
	})
}

// Login handles authenticating existing users
func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Verify credentials with Supabase
	ctx := context.Background()
	session, err := database.SupaClient.Auth.SignIn(ctx, supabase.UserCredentials{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}

	// Return the session token to Electron so it can prove the user is logged in
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "success",
		"access_token": session.AccessToken,
		"user_id":      session.User.ID,
	})
}