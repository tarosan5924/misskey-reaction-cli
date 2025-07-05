package main

import (
	"bytes"
	"encoding/json"
	
	"fmt"
	"io"
	
	"net/http"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// Misskey APIへのリクエストボディ
type reactionRequest struct {
	NoteID   string `json:"noteId"`
	Reaction string `json:"reaction"`
}

// Misskey APIのエラーレスポンス構造体
type misskeyErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code,omitempty"`
	}
}

// Config struct to hold application settings
type Config struct {
	Misskey struct {
		URL   string `yaml:"url"`
		Token string `yaml:"token"`
	} `yaml:"misskey"`
	Reaction struct {
		NoteID string `yaml:"note_id"`
		Emoji  string `yaml:"emoji"`
	} `yaml:"reaction"`
}

// loadConfig reads the configuration from the specified YAML file.
func loadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルを開けませんでした: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("設定ファイルのパースに失敗しました: %w", err)
	}

	return &config, nil
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
	// 設定ファイルを読み込む
	config, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 設定値のバリデーション
	if config.Misskey.URL == "" {
		fmt.Fprintln(os.Stderr, "エラー: 設定ファイルにMisskeyのURLが指定されていません")
		os.Exit(1)
	}
	if config.Misskey.Token == "" {
		fmt.Fprintln(os.Stderr, "エラー: 設定ファイルにMisskeyのAPIトークンが指定されていません")
		os.Exit(1)
	}
	if config.Reaction.NoteID == "" {
		fmt.Fprintln(os.Stderr, "エラー: 設定ファイルにリアクション対象のノートIDが指定されていません")
		os.Exit(1)
	}
	// リアクションが指定されていない場合はデフォルト値を使用
	if config.Reaction.Emoji == "" {
		config.Reaction.Emoji = "👍"
	}

	if err := createReaction(config.Misskey.URL, config.Reaction.NoteID, config.Reaction.Emoji, config.Misskey.Token); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("ノート %s に %s でリアクションしました\n", config.Reaction.NoteID, config.Reaction.Emoji)
}

