package s3client

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client defines the interface for interacting with S3 storage
type S3Client interface {
	// ListEmails retrieves the list of email file keys from the configured bucket
	ListEmails(ctx context.Context) ([]EmailMetadata, error)

	// DownloadEmail retrieves the raw content of an email file
	DownloadEmail(ctx context.Context, key string) ([]byte, error)

	// Close releases any resources held by the client
	Close() error
}

// EmailMetadata contains metadata about an email file in S3
type EmailMetadata struct {
	Key          string
	LastModified time.Time
	Size         int64
}

// client implements the S3Client interface using AWS SDK v2
type client struct {
	s3Client   *s3.Client
	bucketName string
	cache      []EmailMetadata
	cacheValid bool
}

// Config holds configuration for creating an S3 client
type Config struct {
	BucketName string
	Region     string
	AWSProfile string
}

// New creates a new S3Client with the provided configuration
func New(ctx context.Context, cfg Config) (S3Client, error) {
	if cfg.BucketName == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("region is required")
	}

	// Load AWS configuration with retry and credential options
	awsCfg, err := loadAWSConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with connection pooling (handled by SDK)
	s3Client := s3.NewFromConfig(awsCfg)

	return &client{
		s3Client:   s3Client,
		bucketName: cfg.BucketName,
		cacheValid: false,
	}, nil
}

// loadAWSConfig loads AWS configuration with credential loading from multiple sources
func loadAWSConfig(ctx context.Context, cfg Config) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error

	// Set region
	opts = append(opts, config.WithRegion(cfg.Region))

	// Set profile if specified
	if cfg.AWSProfile != "" {
		opts = append(opts, config.WithSharedConfigProfile(cfg.AWSProfile))
	}

	// Load configuration with retry logic (SDK handles this by default)
	// The SDK will automatically try:
	// 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
	// 2. Shared credentials file (~/.aws/credentials)
	// 3. Shared config file (~/.aws/config)
	// 4. IAM roles (if running on EC2/ECS/Lambda)
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("unable to load SDK config: %w", err)
	}

	return awsCfg, nil
}

// ListEmails retrieves the list of email file keys from the configured bucket
func (c *client) ListEmails(ctx context.Context) ([]EmailMetadata, error) {
	// Return cached list if valid
	if c.cacheValid && len(c.cache) > 0 {
		return c.cache, nil
	}

	var emails []EmailMetadata

	// Use paginator to handle large buckets
	paginator := s3.NewListObjectsV2Paginator(c.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucketName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in bucket %s: %w", c.bucketName, err)
		}

		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}

			emails = append(emails, EmailMetadata{
				Key:          *obj.Key,
				LastModified: *obj.LastModified,
				Size:         *obj.Size,
			})
		}
	}

	// Cache the results
	c.cache = emails
	c.cacheValid = true

	return emails, nil
}

// DownloadEmail retrieves the raw content of an email file
func (c *client) DownloadEmail(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	// Get object from S3
	result, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download object %s from bucket %s: %w", key, c.bucketName, err)
	}
	defer result.Body.Close()

	// Read the entire object content
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object content: %w", err)
	}

	return data, nil
}

// Close releases any resources held by the client
func (c *client) Close() error {
	// AWS SDK v2 clients don't require explicit cleanup
	// Connection pooling is managed by the SDK
	c.cacheValid = false
	c.cache = nil
	return nil
}
