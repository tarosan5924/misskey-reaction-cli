package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
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

func TestRunApp_ConfigPathFlag(t *testing.T) {
	// テスト用のFlagSetを作成
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var stdout, stderr bytes.Buffer
	fs.SetOutput(&stderr) // エラー出力をキャプチャ

	configPath := "testdata/custom_config.yaml"
	// コマンドライン引数を設定
	fs.String("config", configPath, "設定ファイルのパス")

	// runApp を呼び出す
	err := runApp(configPath, &stdout, &stderr)

	// エラーが返されることを期待する（まだ実装されていないため）
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}

	// エラーメッセージに設定ファイルパスが含まれていることを確認
	expectedErrorPart := fmt.Sprintf("設定ファイルを開けませんでした: open %s: no such file or directory", configPath)
	if !strings.Contains(err.Error(), expectedErrorPart) {
		t.Errorf("期待するエラーメッセージの一部 '%s' が含まれていませんでした: %v", expectedErrorPart, err)
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

	// テスト対象の関数を呼び出す
	streamNotes(wsURL, "testToken", func(noteID, noteText string) {
		// コールバックが呼び出されたことを確認するためのロジックをここに追加
		// 例: チャネルに通知を送信し、テスト側で受信を待つ
		// 現状は、コンパイルエラーになることを期待する
	})
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

	// テスト対象の関数を呼び出す (まだ存在しない)
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

func TestRunApp_MissingMatchTextError(t *testing.T) {
	// モックの設定ファイルの内容 (match_textを含まない)
	configContent := `
misskey:
  url: "https://test.misskey.example.com"
  token: "test_token_123"
reaction:
  emoji: "👍"
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

	// テスト用のFlagSetを作成
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var stdout, stderr bytes.Buffer
	fs.SetOutput(&stderr) // エラー出力をキャプチャ

	// runApp を呼び出す
	err = runApp(tmpfile.Name(), &stdout, &stderr)

	// エラーが返されることを期待する
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}

	// エラーメッセージを検証する
	expectedError := "エラー: 設定ファイルにリアクション対象の文字列(match_text)が指定されていません"
	if err.Error() != expectedError {
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
	configContent := `
misskey:
  token: "test_token_123"
reaction:
  match_text: "hello"
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗しました: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()
	tmpfile.WriteString(configContent)

	var stdout, stderr bytes.Buffer
	err = runApp(tmpfile.Name(), &stdout, &stderr)

	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}
	expectedError := "エラー: 設定ファイルにMisskeyのURLが指定されていません"
	if err.Error() != expectedError {
		t.Errorf("期待するエラーメッセージ: '%s', 実際: '%s'", expectedError, err.Error())
	}
}

func TestRunApp_MissingToken(t *testing.T) {
	configContent := `
misskey:
  url: "https://test.misskey.example.com"
reaction:
  match_text: "hello"
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗しました: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()
	tmpfile.WriteString(configContent)

	var stdout, stderr bytes.Buffer
	err = runApp(tmpfile.Name(), &stdout, &stderr)

	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}
	expectedError := "エラー: 設定ファイルにMisskeyのAPIトークンが指定されていません"
	if err.Error() != expectedError {
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
	err := streamNotes("ws://localhost:9999", "token", func(noteID, noteText string) {
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
	expectedError := "設定ファイルを開けませんでした"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("期待するエラーメッセージ '%s' が含まれていませんでした: %v", expectedError, err)
	}
}

func TestRunApp_LogFile(t *testing.T) {
	// 一時ログファイルを作成
	tmpLogFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatalf("一時ログファイルの作成に失敗しました: %v", err)
	}
	logPath := tmpLogFile.Name()
	tmpLogFile.Close() // すぐに閉じる
	defer os.Remove(logPath)

	// モックの設定ファイルの内容
	configContent := fmt.Sprintf(`
log_path: %s
misskey:
  url: "http://localhost:8080"
  token: "test_token_123"
reaction:
  match_text: "hello"
`, logPath)

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

	// runAppを呼び出すが、すぐに終了させるためにストリーミング部分はモック化しない
	// そのため、ここではエラーが発生することを期待する
	var stdout, stderr bytes.Buffer
	err = runApp(tmpConfigFile.Name(), &stdout, &stderr)
	if err == nil {
		t.Fatal("エラーが発生することを期待しましたが、発生しませんでした")
	}

	// ログファイルに書き込まれていることを確認
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ログファイルの読み込みに失敗しました: %v", err)
	}

	expectedLogPart := "MisskeyストリーミングAPIに接続中..."
	if !strings.Contains(string(logContent), expectedLogPart) {
		t.Errorf("ログファイルに期待する文字列 '%s' が含まれていませんでした: %s", expectedLogPart, string(logContent))
	}
}
