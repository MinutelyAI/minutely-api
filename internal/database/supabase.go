package database

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/nedpals/supabase-go"
)

// SupaClient is a global variable so other parts of the app can use the database
var SupaClient *supabase.Client
var supabaseURL string
var anonKey string

// InitSupabase loads the .env file and sets up the connection
func InitSupabase() error {
	// 1. Load the .env file
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("could not find or load the .env file")
	}

	// 2. Fetch the keys
	supabaseURL = os.Getenv("SUPABASE_URL")
	anonKey = os.Getenv("SUPABASE_KEY")

	if supabaseURL == "" || anonKey == "" {
		return fmt.Errorf("supabase keys are missing from the .env file")
	}

	// 3. Initialize the client
	SupaClient = supabase.CreateClient(supabaseURL, anonKey)

	fmt.Println("✅ Supabase client successfully initialized and ready!")
	return nil
}

// CreateAuthenticatedClient creates a new Supabase client with user JWT for RLS
func CreateAuthenticatedClient(token string) *supabase.Client {
	// Note: JWT token is managed at the API handler level for RLS.
	// Create the client with the anon key so the required apikey header is present,
	// then attach the user's JWT for auth.uid() evaluation in RLS policies.
	client := supabase.CreateClient(supabaseURL, anonKey)
	client.DB.AddHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	return client
}
