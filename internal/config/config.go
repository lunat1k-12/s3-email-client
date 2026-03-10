package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	// S3 Configuration
	BucketName string `mapstructure:"bucket_name"`
	Region     string `mapstructure:"region"`
	AWSProfile string `mapstructure:"aws_profile"`

	// UI Configuration
	ListPaneWidth int `mapstructure:"list_pane_width"` // Percentage (default: 40)
	RefreshRate   int `mapstructure:"refresh_rate"`    // Milliseconds (default: 100)

	// Behavior
	CacheEmails  bool `mapstructure:"cache_emails"`   // Default: true
	MaxCacheSize int  `mapstructure:"max_cache_size"` // Default: 50

	// Email Response Configuration
	SourceEmail string `mapstructure:"source_email"`
	SESRegion   string `mapstructure:"ses_region"`
}

// Load loads configuration from multiple sources with priority:
// CLI flags > Environment variables > Config file > Defaults
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure config file location
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine config directory: %w", err)
	}

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	// Read config file if it exists (not an error if it doesn't)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Bind environment variables
	v.SetEnvPrefix("S3EMAIL")
	v.AutomaticEnv()
	
	// Explicitly bind environment variables to config keys
	v.BindEnv("bucket_name", "S3EMAIL_BUCKET_NAME")
	v.BindEnv("region", "S3EMAIL_REGION")
	v.BindEnv("aws_profile", "S3EMAIL_AWS_PROFILE")
	v.BindEnv("list_pane_width", "S3EMAIL_LIST_PANE_WIDTH")
	v.BindEnv("refresh_rate", "S3EMAIL_REFRESH_RATE")
	v.BindEnv("cache_emails", "S3EMAIL_CACHE_EMAILS")
	v.BindEnv("max_cache_size", "S3EMAIL_MAX_CACHE_SIZE")
	v.BindEnv("source_email", "S3EMAIL_SOURCE_EMAIL")
	v.BindEnv("ses_region", "S3EMAIL_SES_REGION")

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply SES region default fallback
	if cfg.SESRegion == "" {
		cfg.SESRegion = cfg.Region
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	v.SetDefault("list_pane_width", 40)
	v.SetDefault("refresh_rate", 100)
	v.SetDefault("cache_emails", true)
	v.SetDefault("max_cache_size", 50)
	v.SetDefault("region", "us-east-1")
}

// getConfigDir returns the configuration directory path
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "s3emailclient"), nil
}

// Validate checks that required configuration fields are set
func (c *Config) Validate() error {
	if c.BucketName == "" {
		return fmt.Errorf("bucket_name is required (set via config file, S3EMAIL_BUCKET_NAME env var, or CLI flag)")
	}

	if c.Region == "" {
		return fmt.Errorf("region is required (set via config file, S3EMAIL_REGION env var, or CLI flag)")
	}

	// Validate UI configuration ranges
	if c.ListPaneWidth < 10 || c.ListPaneWidth > 90 {
		return fmt.Errorf("list_pane_width must be between 10 and 90 (got %d)", c.ListPaneWidth)
	}

	if c.RefreshRate < 10 {
		return fmt.Errorf("refresh_rate must be at least 10ms (got %d)", c.RefreshRate)
	}

	if c.MaxCacheSize < 1 {
		return fmt.Errorf("max_cache_size must be at least 1 (got %d)", c.MaxCacheSize)
	}

	return nil
}

// ValidateSourceEmail validates that the source email is configured and has a valid format
func (c *Config) ValidateSourceEmail() error {
	if c.SourceEmail == "" {
		return fmt.Errorf("source_email is required for sending responses")
	}

	// Basic email format validation
	if !containsAt(c.SourceEmail) {
		return fmt.Errorf("source_email must be a valid email address")
	}

	return nil
}

// containsAt checks if a string contains the '@' character
func containsAt(s string) bool {
	for _, c := range s {
		if c == '@' {
			return true
		}
	}
	return false
}

