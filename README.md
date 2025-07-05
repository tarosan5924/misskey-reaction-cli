# Misskey Reaction CLI Tool

This is a simple command-line interface (CLI) tool written in Go to add reactions to Misskey notes.

## Features

- Add reactions to a specified Misskey note.
- Configurable via environment variables.

## Requirements

- Go (version 1.16 or higher)

## Installation

1.  **Clone the repository (if applicable):**

    ```bash
    git clone https://github.com/your-username/misskey-reaction-cli.git
    cd misskey-reaction-cli
    ```

2.  **Build the executable:**

    ```bash
    go build -o misskey-reaction-cli cmd/misskey-reaction-cli/main.go
    ```

    This will create an executable named `misskey-reaction-cli` in your current directory.

## Configuration

This tool requires the following environment variables to be set:

-   `MISSKEY_URL`: The base URL of your Misskey instance (e.g., `https://misskey.example.com`).
-   `MISSKEY_TOKEN`: Your Misskey API token. You can generate one from your Misskey settings.

**Example (Linux/macOS):**

```bash
export MISSKEY_URL="https://misskey.example.com"
export MISSKEY_TOKEN="YOUR_MISSKEY_API_TOKEN"
```

**Example (Windows Command Prompt):**

```cmd
set MISSKEY_URL=https://misskey.example.com
set MISSKEY_TOKEN=YOUR_MISSKEY_API_TOKEN
```

## Usage

Run the executable with the required flags:

```bash
./misskey-reaction-cli -note-id <NOTE_ID> -reaction <REACTION>
```

-   `-note-id`: The ID of the Misskey note you want to react to. (Required)
-   `-reaction`: The reaction emoji or custom emoji name (e.g., `👍`, `:awesome:`). Defaults to `👍` if not specified.

**Example:**

```bash
./misskey-reaction-cli -note-id "9s0d8f7g6h5j4k3l2m1n" -reaction "🎉"
```

## Error Handling

The tool provides basic error handling for missing environment variables, required command-line flags, and Misskey API errors.

## Development

### Running Tests

To run the unit tests:

```bash
go test ./...
```
