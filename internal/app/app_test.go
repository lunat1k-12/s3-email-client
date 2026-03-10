package app

import (
	"context"
	"fmt"
	"testing"

	"s3emailclient/internal/config"
	"s3emailclient/internal/parser"
	"s3emailclient/internal/s3client"
	"s3emailclient/internal/tui"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name:    "nil config returns error",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config creates application",
			config: &config.Config{
				BucketName:    "test-bucket",
				Region:        "us-east-1",
				AWSProfile:    "",
				ListPaneWidth: 40,
				RefreshRate:   100,
				CacheEmails:   true,
				MaxCacheSize:  50,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := New(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}

			if app == nil {
				t.Error("New() returned nil application")
				return
			}

			// Verify all components are initialized
			if app.s3Client == nil {
				t.Error("s3Client not initialized")
			}
			if app.parser == nil {
				t.Error("parser not initialized")
			}
			if app.model == nil {
				t.Error("model not initialized")
			}
			if app.navHandler == nil {
				t.Error("navHandler not initialized")
			}
			if app.config == nil {
				t.Error("config not initialized")
			}
			if app.emailCache == nil {
				t.Error("emailCache not initialized")
			}

			// Clean up
			if err := app.Shutdown(); err != nil {
				t.Errorf("Shutdown() error: %v", err)
			}
		})
	}
}

func TestShutdown(t *testing.T) {
	cfg := &config.Config{
		BucketName:    "test-bucket",
		Region:        "us-east-1",
		ListPaneWidth: 40,
		RefreshRate:   100,
		CacheEmails:   true,
		MaxCacheSize:  50,
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Test shutdown
	err = app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() error: %v", err)
	}

	// Test multiple shutdowns (should be safe)
	err = app.Shutdown()
	if err != nil {
		t.Errorf("Second Shutdown() error: %v", err)
	}
}

func TestLoadEmailList(t *testing.T) {
	tests := []struct {
		name      string
		mockEmails []s3client.EmailMetadata
		mockError error
		wantErr   bool
		wantEmpty bool
	}{
		{
			name: "successful load with emails",
			mockEmails: []s3client.EmailMetadata{
				{Key: "email1.eml", Size: 1024},
				{Key: "email2.eml", Size: 2048},
			},
			mockError: nil,
			wantErr:   false,
			wantEmpty: false,
		},
		{
			name:       "empty bucket returns empty list",
			mockEmails: []s3client.EmailMetadata{},
			mockError:  nil,
			wantErr:    false,
			wantEmpty:  true,
		},
		{
			name:       "S3 error returns error",
			mockEmails: nil,
			mockError:  fmt.Errorf("S3 connection failed"),
			wantErr:    true,
			wantEmpty:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock S3 client
			mockClient := &mockS3Client{
				emails: tt.mockEmails,
				err:    tt.mockError,
			}

			app := &Application{
				s3Client: mockClient,
				config: &config.Config{
					CacheEmails:  true,
					MaxCacheSize: 50,
				},
				emailCache: make(map[string]*parser.Email),
			}

			ctx := context.Background()
			emails, err := app.LoadEmailList(ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadEmailList() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("LoadEmailList() unexpected error: %v", err)
				return
			}

			if tt.wantEmpty && len(emails) != 0 {
				t.Errorf("LoadEmailList() expected empty list, got %d emails", len(emails))
			}

			if !tt.wantEmpty && len(emails) != len(tt.mockEmails) {
				t.Errorf("LoadEmailList() expected %d emails, got %d", len(tt.mockEmails), len(emails))
			}
		})
	}
}

func TestLoadEmail(t *testing.T) {
	testEmailData := []byte("From: test@example.com\r\nSubject: Test\r\n\r\nTest body")

	tests := []struct {
		name         string
		key          string
		cacheEnabled bool
		mockData     []byte
		mockError    error
		wantErr      bool
		testCache    bool
	}{
		{
			name:         "successful load and parse",
			key:          "email1.eml",
			cacheEnabled: true,
			mockData:     testEmailData,
			mockError:    nil,
			wantErr:      false,
			testCache:    false,
		},
		{
			name:         "cache hit returns cached email",
			key:          "email1.eml",
			cacheEnabled: true,
			mockData:     testEmailData,
			mockError:    nil,
			wantErr:      false,
			testCache:    true,
		},
		{
			name:         "S3 download error returns error",
			key:          "email2.eml",
			cacheEnabled: true,
			mockData:     nil,
			mockError:    fmt.Errorf("download failed"),
			wantErr:      true,
			testCache:    false,
		},
		{
			name:         "caching disabled does not cache",
			key:          "email3.eml",
			cacheEnabled: false,
			mockData:     testEmailData,
			mockError:    nil,
			wantErr:      false,
			testCache:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockS3Client{
				data: tt.mockData,
				err:  tt.mockError,
			}

			app := &Application{
				s3Client: mockClient,
				parser:   parser.NewParser(),
				config: &config.Config{
					CacheEmails:  tt.cacheEnabled,
					MaxCacheSize: 50,
				},
				emailCache: make(map[string]*parser.Email),
			}

			// Pre-populate cache if testing cache hit
			if tt.testCache {
				app.emailCache[tt.key] = &parser.Email{
					Subject: "Cached Email",
					From:    "cached@example.com",
				}
			}

			ctx := context.Background()
			email, err := app.LoadEmail(ctx, tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadEmail() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("LoadEmail() unexpected error: %v", err)
				return
			}

			if email == nil {
				t.Error("LoadEmail() returned nil email")
				return
			}

			// Verify cache behavior
			if tt.cacheEnabled && !tt.testCache {
				if _, ok := app.emailCache[tt.key]; !ok {
					t.Error("LoadEmail() did not cache email when caching enabled")
				}
			}

			if !tt.cacheEnabled {
				if len(app.emailCache) > 0 {
					t.Error("LoadEmail() cached email when caching disabled")
				}
			}

			if tt.testCache {
				if email.Subject != "Cached Email" {
					t.Error("LoadEmail() did not return cached email")
				}
			}
		})
	}
}

func TestCacheEviction(t *testing.T) {
	testEmailData := []byte("From: test@example.com\r\nSubject: Test\r\n\r\nTest body")

	mockClient := &mockS3Client{
		data: testEmailData,
		err:  nil,
	}

	app := &Application{
		s3Client: mockClient,
		parser:   parser.NewParser(),
		config: &config.Config{
			CacheEmails:  true,
			MaxCacheSize: 2, // Small cache for testing eviction
		},
		emailCache: make(map[string]*parser.Email),
	}

	ctx := context.Background()

	// Load 3 emails to trigger eviction
	keys := []string{"email1.eml", "email2.eml", "email3.eml"}
	for _, key := range keys {
		_, err := app.LoadEmail(ctx, key)
		if err != nil {
			t.Fatalf("LoadEmail(%s) error: %v", key, err)
		}
	}

	// Cache should only contain 2 emails (max size)
	if len(app.emailCache) != 2 {
		t.Errorf("Cache size = %d, want 2 (max cache size)", len(app.emailCache))
	}
}

// mockS3Client is a mock implementation of S3Client for testing
type mockS3Client struct {
	emails []s3client.EmailMetadata
	data   []byte
	err    error
}

func (m *mockS3Client) ListEmails(ctx context.Context) ([]s3client.EmailMetadata, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.emails, nil
}

func (m *mockS3Client) DownloadEmail(ctx context.Context, key string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func (m *mockS3Client) Close() error {
	return nil
}

func TestLoadEmailCmd(t *testing.T) {
	testEmailData := []byte("From: test@example.com\r\nSubject: Test\r\n\r\nTest body")

	tests := []struct {
		name      string
		key       string
		mockData  []byte
		mockError error
		wantError bool
	}{
		{
			name:      "successful load returns EmailLoadedMsg",
			key:       "email1.eml",
			mockData:  testEmailData,
			mockError: nil,
			wantError: false,
		},
		{
			name:      "download error returns EmailLoadErrorMsg",
			key:       "email2.eml",
			mockData:  nil,
			mockError: fmt.Errorf("download failed"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockS3Client{
				data: tt.mockData,
				err:  tt.mockError,
			}

			app := &Application{
				s3Client: mockClient,
				parser:   parser.NewParser(),
				config: &config.Config{
					CacheEmails:  true,
					MaxCacheSize: 50,
				},
				emailCache: make(map[string]*parser.Email),
			}

			// Execute the command
			cmd := app.LoadEmailCmd(tt.key)
			msg := cmd()

			if tt.wantError {
				// Should return EmailLoadErrorMsg
				errMsg, ok := msg.(tui.EmailLoadErrorMsg)
				if !ok {
					t.Errorf("LoadEmailCmd() expected EmailLoadErrorMsg, got %T", msg)
					return
				}
				if errMsg.Err == nil {
					t.Error("LoadEmailCmd() error message should contain error")
				}
			} else {
				// Should return EmailLoadedMsg with email data
				loadedMsg, ok := msg.(tui.EmailLoadedMsg)
				if !ok {
					t.Errorf("LoadEmailCmd() expected EmailLoadedMsg, got %T", msg)
					return
				}
				if loadedMsg.Email == nil {
					t.Error("LoadEmailCmd() returned nil email")
				}
			}
		})
	}
}
