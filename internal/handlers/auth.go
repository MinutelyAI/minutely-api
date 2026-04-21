package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/MinutelyAI/minutely-api/internal/database"
	"github.com/nedpals/supabase-go"
	"fmt"
	"strings"
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

	fmt.Println("➡️ Login attempt received for:", req.Email)

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
			// 1. Print it to the terminal so YOU can see it
			fmt.Println("🚨 SUPABASE LOGIN ERROR:", err)

			// 2. Safely format it as JSON for Postman
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

	// Return the session token to Electron so it can prove the user is logged in
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "success",
		"access_token": session.AccessToken,
		"token":        session.AccessToken,
		"user_id":      session.User.ID,
	})
}

// GetProfile is a protected route that only logged-in users can access
func GetProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// If the code reaches this point, we know they are 100% authenticated.
	w.Write([]byte(`{"status": "success", "message": "Welcome to your private dashboard!"}`))
}

// Logout invalidates the user's session in Supabase
func Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// 1. Grab the token from the header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "Unauthorized: Missing token"}`, http.StatusUnauthorized)
		return
	}

	// 2. Isolate the actual token string
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		http.Error(w, `{"error": "Unauthorized: Invalid token format"}`, http.StatusUnauthorized)
		return
	}
	token := parts[1]

	// 3. Tell Supabase to invalidate this specific session
	ctx := context.Background()
	err := database.SupaClient.Auth.SignOut(ctx, token)
	
	if err != nil {
		fmt.Println("⚠️ SUPABASE LOGOUT ERROR (ignoring):", err)
		// We ignore the error because if the token is already expired or invalid,
		// the user is effectively logged out anyway. Returning 500 breaks the frontend.
	}

	// 4. Send success response
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Successfully logged out",
	})
}