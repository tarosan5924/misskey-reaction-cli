package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateReaction_Success(t *testing.T) {
	// Mock Misskey API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		// Check request path
		if r.URL.Path != "/api/notes/reactions/create" {
			t.Errorf("Expected path /api/notes/reactions/create, got %s", r.URL.Path)
		}
		// No body to check, just return success
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// Call the function to be tested (it doesn't exist yet)
	err := createReaction(server.URL, "testNoteId", "👍", "testToken")

	// Check the result
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestCreateReaction_APIError(t *testing.T) {
	// Mock Misskey API server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest) // 400 Bad Request
		// A typical Misskey error response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Note not found.",
				"code":    "NOTE_NOT_FOUND",
			},
		})
	}))
	defer server.Close()

	// Call the function to be tested
	err := createReaction(server.URL, "invalidNoteId", "👍", "testToken")

	// Check the result
	if err == nil {
		t.Fatal("Expected an error, but got none")
	}

	expectedError := "API error: Note not found."
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedError, err)
	}
}
