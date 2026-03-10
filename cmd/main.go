package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"s3emailclient/internal/app"
	"s3emailclient/internal/config"
	"s3emailclient/internal/tui"
)

func main() {
	// Load configuration using Viper
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Initialize Application with all components
	application, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Set up graceful shutdown handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
		if err := application.Shutdown(); err != nil {
			fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		}
		os.Exit(0)
	}()

	// Load email list from S3
	emailList, err := application.LoadEmailList(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load email list: %v\n", err)
		os.Exit(1)
	}

	// Convert email metadata to TUI format
	tuiEmailList := make([]tui.EmailListItem, len(emailList))
	for i, email := range emailList {
		tuiEmailList[i] = tui.EmailListItem{
			Key:     email.Key,
			Subject: extractSubjectFromKey(email.Key),
			From:    "",
			Date:    email.LastModified,
		}
	}

	// Initialize the TUI model with email list
	model := application.GetModel()
	model.SetEmailList(tuiEmailList)

	// Run the application (blocks until exit)
	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}

	// Graceful shutdown
	if err := application.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		os.Exit(1)
	}
}

// extractSubjectFromKey extracts a display name from the S3 key
// This is a placeholder until we parse the actual email
func extractSubjectFromKey(key string) string {
	// Simple extraction: use the filename without extension
	// In practice, we'd need to parse the email to get the real subject
	return key
}
