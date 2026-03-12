package app

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"s3emailclient/internal/config"
	"s3emailclient/internal/navigation"
	"s3emailclient/internal/parser"
	"s3emailclient/internal/response"
	"s3emailclient/internal/s3client"
	"s3emailclient/internal/sesclient"
	"s3emailclient/internal/tui"
)

// Application coordinates all components and manages application lifecycle
type Application struct {
	s3Client        s3client.S3Client
	parser          parser.EmailParser
	model           *tui.Model
	navHandler      navigation.NavigationHandler
	responseHandler response.ResponseHandler
	config          *config.Config
	program         *tea.Program
	emailCache      map[string]*parser.Email // Cache of parsed emails by S3 key
}

// New creates a new Application with dependency injection for all components
func New(cfg *config.Config) (*Application, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Initialize S3 client
	ctx := context.Background()
	s3Client, err := s3client.New(ctx, s3client.Config{
		BucketName: cfg.BucketName,
		Region:     cfg.Region,
		AWSProfile: cfg.AWSProfile,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Initialize email parser
	emailParser := parser.NewParser()

	// Initialize navigation handler
	navHandler := navigation.NewNavigationHandler()

	// Initialize SES client for email responses
	// Use SESRegion if configured, otherwise fall back to S3 region
	sesRegion := cfg.SESRegion
	if sesRegion == "" {
		sesRegion = cfg.Region
	}

	sesClient, err := sesclient.NewSESClient(ctx, sesclient.Config{
		Region:     sesRegion,
		AWSProfile: cfg.AWSProfile,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SES client: %w", err)
	}

	// Initialize response handler with config and SES client
	responseHandler := response.NewResponseHandler(cfg, sesClient)

	// Initialize TUI model
	model := &tui.Model{}

	app := &Application{
		s3Client:        s3Client,
		parser:          emailParser,
		model:           model,
		navHandler:      navHandler,
		responseHandler: responseHandler,
		config:          cfg,
		emailCache:      make(map[string]*parser.Email),
	}

	// Wire navigation handler into the model
	model.SetNavigationHandler(navHandler)

	// Wire response handler into the model
	model.SetResponseHandler(responseHandler)

	// Wire email loading callback into the model
	model.SetOnLoadEmail(app.LoadEmailCmd)

	// Wire email delete callback into the model
	model.SetOnDeleteEmail(app.DeleteEmailCmd)

	// Wire email list refresh callback into the model
	model.SetOnRefreshList(app.LoadEmailListCmd)

	return app, nil
}

// LoadEmailList retrieves the list of email files from S3 and returns metadata
// Handles S3 errors and empty bucket scenarios according to requirements 1.2, 1.3, 1.4
func (app *Application) LoadEmailList(ctx context.Context) ([]s3client.EmailMetadata, error) {
	// Invalidate S3 cache to ensure fresh data
	app.s3Client.InvalidateCache()
	
	// Retrieve email list from S3
	emails, err := app.s3Client.ListEmails(ctx)
	if err != nil {
		// Requirement 1.3: Handle S3 inaccessibility
		return nil, fmt.Errorf("failed to retrieve email list from S3: %w", err)
	}

	// Requirement 1.4: Handle empty bucket scenario
	if len(emails) == 0 {
		return []s3client.EmailMetadata{}, nil
	}

	return emails, nil
}
// LoadEmailListCmd returns a Bubble Tea command that loads the email list asynchronously
// This integrates with the TUI's message-based architecture for async operations.
//
// The command performs the following operations:
// 1. Calls LoadEmailList to retrieve email metadata from S3
// 2. Converts EmailMetadata to TUI EmailListItem format
// 3. Returns either EmailListLoadedMsg (success) or EmailLoadErrorMsg (failure)
//
// Note: Subject and From fields are initially empty and will be populated when
// individual emails are loaded. This allows for fast list loading without parsing
// every email upfront.
//
// Requirements: 5.1
func (app *Application) LoadEmailListCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		emails, err := app.LoadEmailList(ctx)
		if err != nil {
			return tui.EmailLoadErrorMsg{Err: err}
		}

		// Convert EmailMetadata to TUI EmailListItem format
		tuiEmails := make([]tui.EmailListItem, len(emails))
		for i, email := range emails {
			tuiEmails[i] = tui.EmailListItem{
				Key:     email.Key,
				Subject: email.Key, // Use S3 key as subject until email is loaded
				From:    "",        // Will be populated when email is loaded
				Date:    email.LastModified,
			}
		}

		return tui.EmailListLoadedMsg{
			Emails: tuiEmails,
		}
	}
}
// DeleteEmail deletes an email from S3 and removes it from cache if caching is enabled
// Returns an error if the deletion fails
// Requirements: 3.1, 3.2
func (app *Application) DeleteEmail(ctx context.Context, key string) error {
	// Delete from S3
	err := app.s3Client.DeleteEmail(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete email from S3: %w", err)
	}

	// Remove from cache if caching is enabled
	if app.config.CacheEmails {
		delete(app.emailCache, key)
	}

	return nil
}

// DeleteEmailCmd returns a Bubble Tea command that deletes an email asynchronously
// This integrates with the TUI's message-based architecture for async operations.
//
// The command performs the following operations:
// 1. Calls DeleteEmail to remove the email from S3
// 2. Invalidates the cache if caching is enabled
// 3. Returns either EmailDeletedMsg (success) or EmailDeleteErrorMsg (failure)
//
// Requirements: 3.1, 3.4, 3.5
func (app *Application) DeleteEmailCmd(key string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := app.DeleteEmail(ctx, key)
		if err != nil {
			return tui.EmailDeleteErrorMsg{
				Key: key,
				Err: err,
			}
		}

		return tui.EmailDeletedMsg{
			Key: key,
		}
	}
}



// LoadEmailCmd returns a Bubble Tea command that loads an email asynchronously
// This integrates with the TUI's message-based architecture for async operations.
//
// The command performs the following operations:
// 1. Downloads the email from S3 using the S3Client
// 2. Parses the raw email data using the EmailParser
// 3. Converts the parsed email to TUI format
// 4. Returns either EmailLoadedMsg (success) or EmailLoadErrorMsg (failure)
//
// Usage example in TUI Update method:
//
//	case SelectEmailMsg:
//	    // Send LoadEmailMsg to update UI state
//	    return m, tea.Batch(
//	        func() tea.Msg { return tui.LoadEmailMsg{Key: msg.Key} },
//	        app.LoadEmailCmd(msg.Key),
//	    )
//
// Requirements: 1.5 (email download), 2.1 (email parsing), 8.1, 8.2 (async coordination)
func (app *Application) LoadEmailCmd(key string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		email, err := app.LoadEmail(ctx, key)
		if err != nil {
			return tui.EmailLoadErrorMsg{Err: err}
		}

		// Convert parser.Email to tui.Email
		tuiEmail := &tui.Email{
			Subject:  email.Subject,
			From:     email.From,
			To:       email.To,
			Cc:       email.Cc,
			Date:     email.Date,
			Body:     email.Body,
			HTMLBody: email.HTMLBody,
		}

		// Convert attachments
		tuiEmail.Attachments = make([]tui.Attachment, len(email.Attachments))
		for i, att := range email.Attachments {
			tuiEmail.Attachments[i] = tui.Attachment{
				Filename:    att.Filename,
				ContentType: att.ContentType,
				Size:        att.Size,
			}
		}

		return tui.EmailLoadedMsg{
			Email:       tuiEmail,
			ParserEmail: email,
		}
	}
}

// LoadEmail retrieves and parses a specific email from S3, using cache if enabled
// Returns the parsed email or an error if the email cannot be retrieved or parsed
func (app *Application) LoadEmail(ctx context.Context, key string) (*parser.Email, error) {
	// Check cache if caching is enabled
	if app.config.CacheEmails {
		if cached, ok := app.emailCache[key]; ok {
			return cached, nil
		}
	}

	// Download email from S3
	data, err := app.s3Client.DownloadEmail(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to download email %s: %w", key, err)
	}

	// Parse email
	email, err := app.parser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email %s: %w", key, err)
	}

	// Cache the email if caching is enabled
	if app.config.CacheEmails {
		app.cacheEmail(key, email)
	}

	return email, nil
}

// cacheEmail adds an email to the cache, evicting oldest entries if cache is full
func (app *Application) cacheEmail(key string, email *parser.Email) {
	// If cache is at max size, remove one entry (simple eviction strategy)
	if len(app.emailCache) >= app.config.MaxCacheSize {
		// Remove the first entry we find (simple FIFO-like eviction)
		for k := range app.emailCache {
			delete(app.emailCache, k)
			break
		}
	}

	app.emailCache[key] = email
}

// Run initializes the Bubble Tea program and starts the application
// This method blocks until the application exits
// Requirements: 5.1 - Load email list on startup for auto-load functionality
func (app *Application) Run() error {
	// Initialize the Bubble Tea program
	app.program = tea.NewProgram(
		app.model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Send initial command to load email list
	// This will trigger the auto-load flow via EmailListLoadedMsg
	go func() {
		app.program.Send(app.LoadEmailListCmd()())
	}()

	// Run the program
	if _, err := app.program.Run(); err != nil {
		return fmt.Errorf("error running TUI program: %w", err)
	}

	return nil
}

// Shutdown performs graceful cleanup of all resources
func (app *Application) Shutdown() error {
	// Close S3 client and release resources
	if app.s3Client != nil {
		if err := app.s3Client.Close(); err != nil {
			return fmt.Errorf("failed to close S3 client: %w", err)
		}
	}

	// Quit the Bubble Tea program if it's running
	if app.program != nil {
		app.program.Quit()
	}

	return nil
}

// GetModel returns the TUI model for initialization
func (app *Application) GetModel() *tui.Model {
	return app.model
}
