# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o bin/s3emailclient ./cmd/s3emailclient

# Run
go run ./cmd/s3emailclient/main.go

# Test
go test ./...                                                    # all tests
go test -v ./internal/tui -run TestCompose                       # single test
go test -cover ./...                                             # with coverage

# Format & tidy
go fmt ./...
go mod tidy
```

## Architecture

This is a Go TUI application for managing emails stored in S3 (via Amazon SES inbound). Incoming emails are stored in S3 by SES; this app reads, displays, and replies to them.

**Data flow:**
```
Inbound email → SES → S3 bucket → this app (reads/displays)
Reply composed in app → SES outbound → recipient
```

**Package layout under `internal/`:**

| Package | Role |
|---|---|
| `app/` | Orchestrates all components; owns `LoadEmailList`, `LoadEmail`, `DeleteEmail`, `Run`, `Shutdown` |
| `tui/` | Bubble Tea model — list pane + content pane + compose mode + delete modal |
| `navigation/` | Maps keyboard input to actions; holds `State` for context-aware key handling |
| `s3client/` | AWS S3 ops (`ListEmails`, `DownloadEmail`, `DeleteEmail`) with in-memory caching |
| `parser/` | MIME parsing via `enmime`; HTML→text via `html2text`; produces `Email` struct |
| `sesclient/` | Sends replies via SES with RFC 5322 threading headers |
| `response/` | Bridges compose UI and SES; handles reply threading (`InReplyTo`, `References`) |
| `config/` | Loads config from `~/.config/s3emailclient/config.yaml` → env vars (`S3EMAIL_` prefix) → defaults |

All inter-package dependencies are expressed through interfaces, injected at startup in `app.New()`. Async operations use Bubble Tea's command/message pattern.

## Key keyboard shortcuts

`j/k` navigate list, `Enter` open email, `r` reply, `d` delete, `Ctrl+S` send reply, `Esc` back/cancel, `q` quit.

## Configuration

Copy `config.example.yaml` to `~/.config/s3emailclient/config.yaml`. Required fields: `bucket_name`, `region`, `source_email`.
