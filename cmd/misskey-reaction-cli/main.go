package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
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
		Emoji     string `yaml:"emoji"`
		MatchText string `yaml:"match_text"`
		MatchType string `yaml:"match_type"`
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

// MisskeyストリーミングAPIのノートイベント構造体
type streamNoteEvent struct {
	Type string `json:"type"`
	Body struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Body struct {
			ID   string `json:"id"`
			Text string `json:"text"`
			// 他のノートのフィールドは必要に応じて追加
		} `json:"body"`
	} `json:"body"`
}

// streamNotes connects to the Misskey streaming API and calls the callback for each note.
func streamNotes(wsURL, token string, noteCallback func(noteID, noteText string)) error {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket接続に失敗しました: %w", err)
	}
	defer conn.Close()

	// チャンネルに接続するためのメッセージを送信
	connectMsg := map[string]interface{}{
		"type": "connect",
		"body": map[string]string{
			"channel": "homeTimeline",
			"id":      "main-channel-id", // 任意のID
		},
	}

	// トークンをメッセージに追加
	connectMsgBody := connectMsg["body"].(map[string]string)
	connectMsgBody["i"] = token

	if err := conn.WriteJSON(connectMsg); err != nil {
		return fmt.Errorf("WebSocketメッセージの送信に失敗しました: %w", err)
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("WebSocketメッセージの読み込みに失敗しました: %w", err)
		}

		var event streamNoteEvent
		if err := json.Unmarshal(message, &event); err != nil {
			// エラーをログに出力するが、処理は続行
			fmt.Fprintf(os.Stderr, "エラー: WebSocketメッセージのパースに失敗しました: %v, メッセージ: %s\n", err, string(message))
			continue
		}

		if event.Type == "channel" && event.Body.Type == "note" {
			noteCallback(event.Body.Body.ID, event.Body.Body.Text)
		}
	}
}

func checkTextMatch(noteText string, config *Config) bool {
	switch config.Reaction.MatchType {
	case "prefix":
		return strings.HasPrefix(noteText, config.Reaction.MatchText)
	case "suffix":
		return strings.HasSuffix(noteText, config.Reaction.MatchText)
	case "contains", "": // デフォルトは部分一致
		return strings.Contains(noteText, config.Reaction.MatchText)
	default:
		return false
	}
}

func runApp(configPath string, stdout, stderr io.Writer) error {

	// 設定ファイルを読み込む
	config, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	// 設定値のバリデーション
	if config.Misskey.URL == "" {
		return fmt.Errorf("エラー: 設定ファイルにMisskeyのURLが指定されていません")
	}
	if config.Misskey.Token == "" {
		return fmt.Errorf("エラー: 設定ファイルにMisskeyのAPIトークンが指定されていません")
	}
	if config.Reaction.MatchText == "" {
		return fmt.Errorf("エラー: 設定ファイルにリアクション対象の文字列(match_text)が指定されていません")
	}

	// リアクションが指定されていない場合はデフォルト値を使用
	if config.Reaction.Emoji == "" {
		config.Reaction.Emoji = "👍"
	}

	// ストリーミングAPIのURLを構築
	wsURL := strings.Replace(config.Misskey.URL, "http", "ws", 1) + "/streaming?i=" + config.Misskey.Token

	fmt.Fprintf(stdout, "MisskeyストリーミングAPIに接続中... %s\n", wsURL)

	// ストリーミングAPIからノートを受信し、リアクションを投稿
	err = streamNotes(wsURL, config.Misskey.Token, func(noteID, noteText string) {
		// 特定文字列に合致するかチェック
		if !checkTextMatch(noteText, config) {
			return // 合致しない場合はスキップ
		}

		// 即時リアクションが来るのは怖いので若干遅延させる
		delay := time.Duration(rand.Intn(4)+5) * time.Second
		time.Sleep(delay)

		fmt.Fprintf(stdout, "ノートID: %s, テキスト: %s にリアクション %s を投稿します\n", noteID, noteText, config.Reaction.Emoji)
		if err := createReaction(config.Misskey.URL, noteID, config.Reaction.Emoji, config.Misskey.Token); err != nil {
			fmt.Fprintf(stderr, "エラー: リアクションの投稿に失敗しました: %v\n", err)
		}
	})

	if err != nil {
		return fmt.Errorf("ストリーミングAPIの処理中にエラーが発生しました: %w", err)
	}

	return nil
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "config.yaml", "設定ファイルのパス")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	return runApp(*configPath, stdout, stderr)
}

func main() {
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

