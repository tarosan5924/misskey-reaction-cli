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
	err := runApp(fs, configPath, &stdout, &stderr)

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
  note_id: "test_note_id_456"
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
