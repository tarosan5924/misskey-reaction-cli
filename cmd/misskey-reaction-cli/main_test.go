package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestCreateReaction_Success(t *testing.T) {
	// モックMisskey APIサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// リクエストメソッドをチェック
		if r.Method != http.MethodPost {
			t.Errorf("POSTリクエストを期待しましたが、%sが来ました", r.Method)
		}
		// リクエストパスをチェック
		if r.URL.Path != "/api/notes/reactions/create" {
			t.Errorf("パス /api/notes/reactions/create を期待しましたが、%sが来ました", r.URL.Path)
		}
		// ボディはチェックせず、成功を返す
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// テスト対象の関数を呼び出す
	err := createReaction(server.URL, "testNoteId", "👍", "testToken")

	// 結果をチェックする
	if err != nil {
		t.Errorf("エラーが発生しないことを期待しましたが、発生しました: %v", err)
	}
}

func TestCreateReaction_APIError(t *testing.T) {
	// エラーを返すMisskey APIのモックサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest) // 400 Bad Request
		// Misskeyのエラーレスポンスの典型的な形式
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "ノートが見つかりません。",
				"code":    "NOTE_NOT_FOUND",
			},
		})
	}))
	defer server.Close()

	// テスト対象の関数を呼び出す
	err := createReaction(server.URL, "invalidNoteId", "👍", "testToken")

	// 結果をチェックする
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}

	expectedError := "API error: ノートが見つかりません。"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("エラーメッセージに '%s' が含まれることを期待しましたが、実際は: %v", expectedError, err)
	}
}

func TestStreamNotes(t *testing.T) {
	// モックWebSocketサーバーをセットアップ
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			t.Fatalf("WebSocketアップグレードに失敗しました: %v", err)
		}
		defer conn.Close()

		// テスト用のノートイベントを送信
		noteEvent := streamNoteEvent{
			Type: "channel",
			Body: struct {
				ID   string `json:"id"`
				Type string `json:"type"`
				Body struct {
					ID   string `json:"id"`	
					Text string `json:"text"`
				} `json:"body"`
			}{
				ID:   "testChannelId",
				Type: "note",
				Body: struct {
					ID   string `json:"id"`
					Text string `json:"text"`
				}{
					ID:   "testNoteId123",
					Text: "これはテストノートです",
				},
			},
		}
		jsonBytes, _ := json.Marshal(noteEvent)
		conn.WriteMessage(websocket.TextMessage, jsonBytes)

		// クライアントからのメッセージを待つ（接続維持のため）
		conn.ReadMessage()
	}))
	defer server.Close()

	// WebSocket URLをHTTPからWSに変換
	wsURL := "ws" + server.URL[len("http"):]

	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.Ldate|log.Ltime)
	// テスト対象の関数を呼び出す
	streamNotes(wsURL, "testToken", logger, func(noteID, noteText string) {
		// This is a dummy callback for testing compilation
	})
}

func TestStreamNotes_ParseError(t *testing.T) {
	// モックWebSocketサーバーをセットアップ
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			t.Fatalf("WebSocketアップグレードに失敗しました: %v", err)
		}
		defer conn.Close()

		// 不正なJSONを送信
		conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))

		// クライアントからのメッセージを待つ（接続維持のため）
		conn.ReadMessage()
	}))
	defer server.Close()

	// WebSocket URLをHTTPからWSに変換
	wsURL := "ws" + server.URL[len("http"):]

	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.Ldate|log.Ltime)
	// テスト対象の関数を呼び出す
	streamNotes(wsURL, "testToken", logger, func(noteID, noteText string) {
		// コールバックは呼び出されないはず
		t.Error("コールバックが呼び出されましたが、これはエラーケースです")
	})

	// ログにエラーメッセージが含まれていることを確認
	expectedLog := "エラー: WebSocketメッセージのパースに失敗しました"
	if !strings.Contains(logBuffer.String(), expectedLog) {
		t.Errorf("ログに期待するエラー '%s' が含まれていませんでした: %s", expectedLog, logBuffer.String())
	}
}

func TestLoadConfig(t *testing.T) {
	// モックの設定ファイルの内容
	configContent := `
misskey:
  url: "https://test.misskey.example.com"
  token: "test_token_123"
reaction:
  emoji: ":test_emoji:"
`

	// 一時ファイルに設定内容を書き込む
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗しました: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	_, err = tmpfile.WriteString(configContent)
	if err != nil {
		t.Fatalf("一時ファイルへの書き込みに失敗しました: %v", err)
	}

	// テスト対象の関数を呼び出す
	config, err := loadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	// 結果を検証する
	if config.Misskey.URL != "https://test.misskey.example.com" {
		t.Errorf("期待するMisskey URL: %s, 実際: %s", "https://test.misskey.example.com", config.Misskey.URL)
	}
	if config.Misskey.Token != "test_token_123" {
		t.Errorf("期待するMisskey Token: %s, 実際: %s", "test_token_123", config.Misskey.Token)
	}
	if config.Reaction.Emoji != ":test_emoji:" {
		t.Errorf("期待するReaction Emoji: %s, 実際: %s", ":test_emoji:", config.Reaction.Emoji)
	}
}

func TestRunApp_MissingMatchText(t *testing.T) {
	config := &Config{
		Misskey: struct {
			URL   string `yaml:"url"`
			Token string `yaml:"token"`
		}{
			URL:   "https://test.misskey.example.com",
			Token: "test_token_123",
		},
		Reaction: struct {
			Emoji     string `yaml:"emoji"`
			MatchText string `yaml:"match_text"`
			MatchType string `yaml:"match_type"`
		}{
			MatchText: "", // MatchText is missing
		},
	}

	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.Ldate|log.Ltime)
	err := runApp(config, logger)

	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}

	// エラーメッセージを検証する
	expectedError := "エラー: 設定ファイルにリアクション対象の文字列(match_text)が指定されていません"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("期待するエラーメッセージ: '%s', 実際: '%s'", expectedError, err.Error())
	}
}

func TestCheckTextMatch(t *testing.T) {
	tests := []struct {
		name      string
		matchType string
		noteText  string
		matchText string
		expected  bool
	}{
		{"前方一致_一致", "prefix", "hello world", "hello", true},
		{"前方一致_不一致", "prefix", "hello world", "world", false},
		{"後方一致_一致", "suffix", "hello world", "world", true},
		{"後方一致_不一致", "suffix", "hello world", "hello", false},
		{"部分一致_一致", "contains", "hello world", "lo wo", true},
		{"部分一致_不一致", "contains", "hello world", "wollo", false},
		{"デフォルト(部分一致)_一致", "", "hello world", "lo wo", true},
		{"デフォルト(部分一致)_不一致", "", "hello world", "wollo", false},
		{"無効なタイプ", "invalid", "hello world", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Reaction: struct {
					Emoji     string `yaml:"emoji"`
					MatchText string `yaml:"match_text"`
					MatchType string `yaml:"match_type"`
				}{
					MatchText: tt.matchText,
					MatchType: tt.matchType,
				},
			}
			if checkTextMatch(tt.noteText, config) != tt.expected {
				t.Errorf("期待値: %v, 実際: %v", tt.expected, !tt.expected)
			}
		})
	}
}

func TestLoadConfig_InvalidYaml(t *testing.T) {
	// 無効なYAMLコンテンツ
	configContent := `
misskey:
  url: "https://test.misskey.example.com"
  token: "test_token_123"
reaction:
  emoji: ":test_emoji:"
  match_text: "hello
`

	// 一時ファイルに設定内容を書き込む
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗しました: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	_, err = tmpfile.WriteString(configContent)
	if err != nil {
		t.Fatalf("一時ファイルへの書き込みに失敗しました: %v", err)
	}

	// テスト対象の関数を呼び出す
	_, err = loadConfig(tmpfile.Name())
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}

	// エラーメッセージを検証する
	expectedErrorPart := "設定ファイルのパースに失敗しました"
	if !strings.Contains(err.Error(), expectedErrorPart) {
		t.Errorf("期待するエラーメッセージ '%s' が含まれていませんでした: %v", expectedErrorPart, err)
	}
}

func TestRunApp_MissingURL(t *testing.T) {
	config := &Config{
		Misskey: struct {
			URL   string `yaml:"url"`
			Token string `yaml:"token"`
		}{
			URL:   "", // URL is missing
			Token: "test_token_123",
		},
		Reaction: struct {
			Emoji     string `yaml:"emoji"`
			MatchText string `yaml:"match_text"`
			MatchType string `yaml:"match_type"`
		}{
			MatchText: "hello",
		},
	}

	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.Ldate|log.Ltime)
	err := runApp(config, logger)

	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}
	expectedError := "エラー: 設定ファイルにMisskeyのURLが指定されていません"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("期待するエラーメッセージ: '%s', 実際: '%s'", expectedError, err.Error())
	}
}

func TestRunApp_MissingToken(t *testing.T) {
	config := &Config{
		Misskey: struct {
			URL   string `yaml:"url"`
			Token string `yaml:"token"`
		}{
			URL:   "https://test.misskey.example.com",
			Token: "", // Token is missing
		},
		Reaction: struct {
			Emoji     string `yaml:"emoji"`
			MatchText string `yaml:"match_text"`
			MatchType string `yaml:"match_type"`
		}{
			MatchText: "hello",
		},
	}

	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.Ldate|log.Ltime)
	err := runApp(config, logger)

	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}
	expectedError := "エラー: 設定ファイルにMisskeyのAPIトークンが指定されていません"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("期待するエラーメッセージ: '%s', 実際: '%s'", expectedError, err.Error())
	}
}

func TestCreateReaction_RequestCreationError(t *testing.T) {
	// 無効なURLを渡してリクエスト作成を失敗させる
	err := createReaction("http://invalid url", "noteId", "reaction", "token")
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}
	if !strings.Contains(err.Error(), "failed to create request") {
		t.Errorf("期待するエラーメッセージが含まれていませんでした: %v", err)
	}
}

func TestStreamNotes_DialError(t *testing.T) {
	// 存在しないサーバーへの接続を試みる
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.Ldate|log.Ltime)
	err := streamNotes("ws://localhost:9999", "token", logger, func(noteID, noteText string) {
		t.Error("コールバックが呼び出されるべきではありません")
	})
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}
	if !strings.Contains(err.Error(), "WebSocket接続に失敗しました") {
		t.Errorf("期待するエラーメッセージが含まれていませんでした: %v", err)
	}
}

func TestRun_flags(t *testing.T) {
	var stderr bytes.Buffer
	// 不正な引数を渡して、パースエラーを発生させる
	err := run([]string{"cmd", "-invalid-flag"}, nil, &stderr)
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}
	expectedError := "flag provided but not defined: -invalid-flag"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("期待するエラーメッセージ '%s' が含まれていませんでした: %v", expectedError, err)
	}
}

func TestRun_runAppError(t *testing.T) {
	var stderr bytes.Buffer
	// configファイルが存在しない場合のエラーをテスト
	err := run([]string{"cmd", "-config", "non-existent-file.yaml"}, nil, &stderr)
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}
	expectedError := "設定ファイルの読み込みに失敗しました"
	if !strings.Contains(stderr.String(), expectedError) {
		t.Errorf("期待するエラーメッセージ '%s' が含まれていませんでした: %v", expectedError, stderr.String())
	}
}

func TestRun_LogFile_Error(t *testing.T) {
	// 不正なログファイルパスを持つ設定ファイルを作成
	configContent := `
log_path: "/invalid/path/to/logfile.log"
misskey:
  url: "https://test.misskey.example.2com"
  token: "test_token_123"
reaction:
  match_text: "hello"
`
	tmpConfigFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("一時設定ファイルの作成に失敗しました: %v", err)
	}
	defer os.Remove(tmpConfigFile.Name())
	defer tmpConfigFile.Close()

	_, err = tmpConfigFile.WriteString(configContent)
	if err != nil {
		t.Fatalf("一時設定ファイルへの書き込みに失敗しました: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = run([]string{"cmd", "-config", tmpConfigFile.Name()}, &stdout, &stderr)
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}

	// stderrにエラーメッセージが書き込まれていることを確認
	expectedError := "ログファイルを開けませんでした"
	if !strings.Contains(stderr.String(), expectedError) {
		t.Errorf("期待するエラー '%s' が含まれていませんでした: %s", expectedError, stderr.String())
	}
}