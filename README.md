# MisskeyリアクションCLIツール

これは、Go言語で書かれたMisskeyのノートにリアクションを追加するためのシンプルなコマンドラインインターフェース（CLI）ツールです。

## 機能

- 指定されたMisskeyノートにリアクションを追加します。
- 環境変数を通じて設定可能です。

## 要件

- Go (バージョン 1.16 以降)

## インストール

1.  **リポジトリのクローン (該当する場合):**

    ```bash
    git clone https://github.com/your-username/misskey-reaction-cli.git
    cd misskey-reaction-cli
    ```

2.  **実行可能ファイルのビルド:**

    ```bash
    go build -o misskey-reaction-cli cmd/misskey-reaction-cli/main.go
    ```

    これにより、現在のディレクトリに `misskey-reaction-cli` という名前の実行可能ファイルが作成されます。

## 設定

このツールは、以下の環境変数を設定する必要があります。

-   `MISSKEY_URL`: MisskeyインスタンスのベースURL（例: `https://misskey.example.com`）。
-   `MISSKEY_TOKEN`: あなたのMisskey APIトークン。Misskeyの設定から生成できます。

**例 (Linux/macOS):**

```bash
export MISSKEY_URL="https://misskey.example.com"
export MISSKEY_TOKEN="YOUR_MISSKEY_API_TOKEN"
```

**例 (Windows コマンドプロンプト):**

```cmd
set MISSKEY_URL=https://misskey.example.com
set MISSKEY_TOKEN=YOUR_MISSKEY_API_TOKEN
```

## 使用方法

必要なフラグを指定して実行します。

```bash
./misskey-reaction-cli -note-id <ノートID> -reaction <リアクション>
```

-   `-note-id`: リアクションを追加したいMisskeyノートのID。（必須）
-   `-reaction`: 追加するリアクションの絵文字またはカスタム絵文字名（例: `👍`、`:awesome:`）。指定しない場合、デフォルトは `👍` です。

**例:**

```bash
./misskey-reaction-cli -note-id "9s0d8f7g6h5j4k3l2m1n" -reaction "🎉"
```

## エラーハンドリング

このツールは、環境変数の不足、必須コマンドラインフラグの不足、Misskey APIエラーに対する基本的なエラーハンドリングを提供します。

## 開発

### テストの実行

ユニットテストを実行するには：

```bash
go test ./...
```