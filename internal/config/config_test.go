package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				BucketName:    "test-bucket",
				Region:        "us-west-2",
				ListPaneWidth: 40,
				RefreshRate:   100,
				MaxCacheSize:  50,
			},
			wantErr: false,
		},
		{
			name: "missing bucket name",
			config: Config{
				Region:        "us-west-2",
				ListPaneWidth: 40,
				RefreshRate:   100,
				MaxCacheSize:  50,
			},
			wantErr: true,
			errMsg:  "bucket_name is required",
		},
		{
			name: "missing region",
			config: Config{
				BucketName:    "test-bucket",
				ListPaneWidth: 40,
				RefreshRate:   100,
				MaxCacheSize:  50,
			},
			wantErr: true,
			errMsg:  "region is required",
		},
		{
			name: "invalid list pane width - too small",
			config: Config{
				BucketName:    "test-bucket",
				Region:        "us-west-2",
				ListPaneWidth: 5,
				RefreshRate:   100,
				MaxCacheSize:  50,
			},
			wantErr: true,
			errMsg:  "list_pane_width must be between 10 and 90",
		},
		{
			name: "invalid list pane width - too large",
			config: Config{
				BucketName:    "test-bucket",
				Region:        "us-west-2",
				ListPaneWidth: 95,
				RefreshRate:   100,
				MaxCacheSize:  50,
			},
			wantErr: true,
			errMsg:  "list_pane_width must be between 10 and 90",
		},
		{
			name: "invalid refresh rate",
			config: Config{
				BucketName:    "test-bucket",
				Region:        "us-west-2",
				ListPaneWidth: 40,
				RefreshRate:   5,
				MaxCacheSize:  50,
			},
			wantErr: true,
			errMsg:  "refresh_rate must be at least 10ms",
		},
		{
			name: "invalid max cache size",
			config: Config{
				BucketName:    "test-bucket",
				Region:        "us-west-2",
				ListPaneWidth: 40,
				RefreshRate:   100,
				MaxCacheSize:  0,
			},
			wantErr: true,
			errMsg:  "max_cache_size must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && len(err.Error()) < len(tt.errMsg) {
					t.Errorf("Validate() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	// Point HOME to an empty temp dir so the real config file is not read
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", t.TempDir())
	defer os.Setenv("HOME", originalHome)

	// Set environment variables
	os.Setenv("S3EMAIL_BUCKET_NAME", "env-test-bucket")
	os.Setenv("S3EMAIL_REGION", "eu-west-1")
	os.Setenv("S3EMAIL_AWS_PROFILE", "test-profile")
	defer func() {
		os.Unsetenv("S3EMAIL_BUCKET_NAME")
		os.Unsetenv("S3EMAIL_REGION")
		os.Unsetenv("S3EMAIL_AWS_PROFILE")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.BucketName != "env-test-bucket" {
		t.Errorf("BucketName = %v, want env-test-bucket", cfg.BucketName)
	}

	if cfg.Region != "eu-west-1" {
		t.Errorf("Region = %v, want eu-west-1", cfg.Region)
	}

	if cfg.AWSProfile != "test-profile" {
		t.Errorf("AWSProfile = %v, want test-profile", cfg.AWSProfile)
	}

	// Check defaults are applied
	if cfg.ListPaneWidth != 40 {
		t.Errorf("ListPaneWidth = %v, want 40", cfg.ListPaneWidth)
	}

	if cfg.RefreshRate != 100 {
		t.Errorf("RefreshRate = %v, want 100", cfg.RefreshRate)
	}

	if cfg.CacheEmails != true {
		t.Errorf("CacheEmails = %v, want true", cfg.CacheEmails)
	}

	if cfg.MaxCacheSize != 50 {
		t.Errorf("MaxCacheSize = %v, want 50", cfg.MaxCacheSize)
	}
}

func TestLoadWithConfigFile(t *testing.T) {
	// Create a temporary config directory
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "s3emailclient")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create a test config file
	configContent := `bucket_name: file-test-bucket
region: ap-southeast-1
aws_profile: file-profile
list_pane_width: 30
refresh_rate: 200
cache_emails: false
max_cache_size: 100
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Temporarily override home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.BucketName != "file-test-bucket" {
		t.Errorf("BucketName = %v, want file-test-bucket", cfg.BucketName)
	}

	if cfg.Region != "ap-southeast-1" {
		t.Errorf("Region = %v, want ap-southeast-1", cfg.Region)
	}

	if cfg.AWSProfile != "file-profile" {
		t.Errorf("AWSProfile = %v, want file-profile", cfg.AWSProfile)
	}

	if cfg.ListPaneWidth != 30 {
		t.Errorf("ListPaneWidth = %v, want 30", cfg.ListPaneWidth)
	}

	if cfg.RefreshRate != 200 {
		t.Errorf("RefreshRate = %v, want 200", cfg.RefreshRate)
	}

	if cfg.CacheEmails != false {
		t.Errorf("CacheEmails = %v, want false", cfg.CacheEmails)
	}

	if cfg.MaxCacheSize != 100 {
		t.Errorf("MaxCacheSize = %v, want 100", cfg.MaxCacheSize)
	}
}

func TestLoadMissingRequiredConfig(t *testing.T) {
	// Clear any environment variables that might be set
	os.Unsetenv("S3EMAIL_BUCKET_NAME")
	os.Unsetenv("S3EMAIL_REGION")

	// Create a temporary empty home directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when required config is missing")
	}
}

func TestGetConfigDir(t *testing.T) {
	dir, err := getConfigDir()
	if err != nil {
		t.Fatalf("getConfigDir() failed: %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".config", "s3emailclient")

	if dir != expected {
		t.Errorf("getConfigDir() = %v, want %v", dir, expected)
	}
}

func TestSESRegionDefaultFallback(t *testing.T) {
	tests := []struct {
		name              string
		region            string
		sesRegion         string
		expectedSESRegion string
	}{
		{
			name:              "ses_region not specified - defaults to region",
			region:            "us-west-2",
			sesRegion:         "",
			expectedSESRegion: "us-west-2",
		},
		{
			name:              "ses_region explicitly specified",
			region:            "us-west-2",
			sesRegion:         "us-east-1",
			expectedSESRegion: "us-east-1",
		},
		{
			name:              "ses_region same as region",
			region:            "eu-west-1",
			sesRegion:         "eu-west-1",
			expectedSESRegion: "eu-west-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config directory
			tempDir := t.TempDir()
			configDir := filepath.Join(tempDir, ".config", "s3emailclient")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatalf("Failed to create config directory: %v", err)
			}

			// Create a test config file
			configContent := "bucket_name: test-bucket\n"
			configContent += "region: " + tt.region + "\n"
			if tt.sesRegion != "" {
				configContent += "ses_region: " + tt.sesRegion + "\n"
			}

			configPath := filepath.Join(configDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Temporarily override home directory
			originalHome := os.Getenv("HOME")
			os.Setenv("HOME", tempDir)
			defer os.Setenv("HOME", originalHome)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.SESRegion != tt.expectedSESRegion {
				t.Errorf("SESRegion = %v, want %v", cfg.SESRegion, tt.expectedSESRegion)
			}
		})
	}
}

func TestSESRegionFallbackWithEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name              string
		setRegionEnv      bool
		regionEnv         string
		setSESRegionEnv   bool
		sesRegionEnv      string
		expectedSESRegion string
	}{
		{
			name:              "only region env set - ses_region defaults to region",
			setRegionEnv:      true,
			regionEnv:         "us-west-2",
			setSESRegionEnv:   false,
			expectedSESRegion: "us-west-2",
		},
		{
			name:              "both region and ses_region env set",
			setRegionEnv:      true,
			regionEnv:         "us-west-2",
			setSESRegionEnv:   true,
			sesRegionEnv:      "eu-central-1",
			expectedSESRegion: "eu-central-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("S3EMAIL_BUCKET_NAME", "env-test-bucket")
			if tt.setRegionEnv {
				os.Setenv("S3EMAIL_REGION", tt.regionEnv)
			}
			if tt.setSESRegionEnv {
				os.Setenv("S3EMAIL_SES_REGION", tt.sesRegionEnv)
			}

			defer func() {
				os.Unsetenv("S3EMAIL_BUCKET_NAME")
				os.Unsetenv("S3EMAIL_REGION")
				os.Unsetenv("S3EMAIL_SES_REGION")
			}()

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.SESRegion != tt.expectedSESRegion {
				t.Errorf("SESRegion = %v, want %v", cfg.SESRegion, tt.expectedSESRegion)
			}
		})
	}
}
