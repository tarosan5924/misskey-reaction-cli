package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
