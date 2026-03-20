package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/MinutelyAI/minutely-api/internal/database"
)

// RequireAuth is the middleware
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		// Handle preflight OPTIONS requests for CORS (Important for Electron)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 1. Grab the Authorization header sent by Electron
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "Unauthorized: Missing token"}`, http.StatusUnauthorized)
			return
		}

		// 2. Format should be "Bearer <the_actual_token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "Unauthorized: Invalid token format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// 3. Verify the token with Supabase
		ctx := context.Background()
		user, err := database.SupaClient.Auth.User(ctx, token)

		if err != nil || user == nil {
			http.Error(w, `{"error": "Unauthorized: Invalid or expired session"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}
