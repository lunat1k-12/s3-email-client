package s3client

import (
	"context"
	"testing"
)

func TestNew_ValidatesRequiredFields(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				BucketName: "test-bucket",
				Region:     "us-east-1",
			},
			expectError: false,
		},
		{
			name: "missing bucket name",
			config: Config{
				Region: "us-east-1",
			},
			expectError: true,
			errorMsg:    "bucket name is required",
		},
		{
			name: "missing region",
			config: Config{
				BucketName: "test-bucket",
			},
			expectError: true,
			errorMsg:    "region is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(ctx, tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestClient_ImplementsInterface(t *testing.T) {
	// This test verifies that client implements S3Client interface at compile time
	var _ S3Client = (*client)(nil)
}

func TestDownloadEmail_ValidatesKey(t *testing.T) {
	// Create a client with minimal config (won't actually connect to AWS)
	c := &client{
		bucketName: "test-bucket",
	}

	ctx := context.Background()

	_, err := c.DownloadEmail(ctx, "")
	if err == nil {
		t.Error("expected error for empty key, got nil")
	}
	if err.Error() != "key is required" {
		t.Errorf("expected 'key is required' error, got: %v", err)
	}
}

func TestClose(t *testing.T) {
	c := &client{
		bucketName: "test-bucket",
		cache:      []EmailMetadata{{Key: "test"}},
		cacheValid: true,
	}

	err := c.Close()
	if err != nil {
		t.Errorf("unexpected error from Close: %v", err)
	}

	if c.cacheValid {
		t.Error("expected cacheValid to be false after Close")
	}

	if c.cache != nil {
		t.Error("expected cache to be nil after Close")
	}
}
