package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Request body for the Misskey API
type reactionRequest struct {
	NoteID   string `json:"noteId"`
	Reaction string `json:"reaction"`
}

// Misskey API error response structure
type misskeyErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code,omitempty"`
	}
}

func createReaction(misskeyURL, noteID, reaction, token string) error {
	apiURL := misskeyURL + "/api/notes/reactions/create"

	reactionBody := reactionRequest{
		NoteID:   noteID,
		Reaction: reaction,
	}

	jsonBody, err := json.Marshal(reactionBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		// Read the response body for error details
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("unexpected status code: %d, failed to read response body: %w", resp.StatusCode, readErr)
		}

		var errorResponse misskeyErrorResponse
		if unmarshalErr := json.Unmarshal(bodyBytes, &errorResponse); unmarshalErr != nil {
			return fmt.Errorf("unexpected status code: %d, failed to unmarshal error response: %w, body: %s", resp.StatusCode, unmarshalErr, string(bodyBytes))
		}

		// Adjusted error message format
		errMsg := fmt.Sprintf("API error: %s", errorResponse.Error.Message)
		if errorResponse.Error.Code != "" {
			errMsg += fmt.Sprintf(" (Code: %s)", errorResponse.Error.Code)
		}
		errMsg += fmt.Sprintf(" (Status: %d)", resp.StatusCode)
		return fmt.Errorf(errMsg)
	}

	return nil
}

func main() {
	// Environment variables
	misskeyURL := os.Getenv("MISSKEY_URL")
	if misskeyURL == "" {
		fmt.Fprintln(os.Stderr, "Error: MISSKEY_URL environment variable not set")
		os.Exit(1)
	}

	misskeyToken := os.Getenv("MISSKEY_TOKEN")
	if misskeyToken == "" {
		fmt.Fprintln(os.Stderr, "Error: MISSKEY_TOKEN environment variable not set")
		os.Exit(1)
	}

	// Command-line flags
	noteID := flag.String("note-id", "", "The ID of the note to react to")
	reaction := flag.String("reaction", "👍", "The reaction to add to the note")
	flag.Parse()

	if *noteID == "" {
		fmt.Fprintln(os.Stderr, "Error: -note-id flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if err := createReaction(misskeyURL, *noteID, *reaction, misskeyToken); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully reacted to note %s with %s\n", *noteID, *reaction)
}

