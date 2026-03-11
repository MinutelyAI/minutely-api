package database

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/nedpals/supabase-go"
)

// SupaClient is a global variable so other parts of your app can use the database
var SupaClient *supabase.Client

// InitSupabase loads the .env file and sets up the connection
func InitSupabase() error {
	// 1. Load the .env file
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("could not find or load the .env file")
	}

	// 2. Fetch the keys
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")

	if supabaseURL == "" || supabaseKey == "" {
		return fmt.Errorf("supabase keys are missing from the .env file")
	}

	// 3. Initialize the client
	SupaClient = supabase.CreateClient(supabaseURL, supabaseKey)
	
	fmt.Println("✅ Supabase client successfully initialized and ready!")
	return nil
}