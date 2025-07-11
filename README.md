# MisskeyリアクションCLIツール

これは、Go言語で書かれたMisskeyのノートにリアクションを追加するためのシンプルなコマンドラインインターフェース（CLI）ツールです。

## 機能

- 指定されたMisskeyノートにリアクションを追加します。
- 設定ファイルを通じて設定可能です。

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

このツールは、`config.yaml` という設定ファイルから設定を読み込みます。デフォルトでは実行可能ファイルと同じディレクトリの `config.yaml` を探しますが、`-config` フラグで別のパスを指定することもできます。

**`config.yaml` の例:**

```yaml
misskey:
  url: "https://misskey.example.com"
  token: "YOUR_MISSKEY_API_TOKEN"
reaction:
  emoji: "👍"
  match_text: "特定の文字列"
  match_type: "contains"
```

-   `misskey.url`: MisskeyインスタンスのベースURL（例: `https://misskey.example.com`）。
-   `misskey.token`: あなたのMisskey APIトークン。Misskeyの設定から生成できます。
-   `reaction.emoji`: 追加するリアクションの絵文字またはカスタム絵文字名（例: `👍`、`:awesome:`）。指定しない場合、デフォルトは `👍` です。
-   `reaction.match_text`: リアクションを行うノートのテキストに含まれるべき特定の文字列。
-   `reaction.match_type`: `match_text`とノートのテキストを比較する方法を指定します。以下のいずれかを指定できます。
    -   `prefix`: 前方一致
    -   `suffix`: 後方一致
    -   `contains`: 部分一致（デフォルト）

## 使用方法

設定ファイル (`config.yaml`) を準備した後、以下のコマンドでツールを実行します。

```bash
./misskey-reaction-cli
```

または、`-config` フラグで設定ファイルのパスを指定します。

```bash
./misskey-reaction-cli -config /path/to/your/custom_config.yaml
```

**例:**

`config.yaml` に以下の内容を記述します。

```yaml
misskey:
  url: "https://misskey.example.com"
  token: "YOUR_MISSKEY_API_TOKEN"
reaction:
  emoji: "🎉"
  match_text: "テスト"
```

その後、ツールを実行します。

```bash
./misskey-reaction-cli
```

## エラーハンドリング

このツールは、設定ファイルの不足、設定値の不足、Misskey APIエラーに対する基本的なエラーハンドリングを提供します。

## 開発

### テストの実行

ユニットテストを実行するには：

```bash
go test ./...
```