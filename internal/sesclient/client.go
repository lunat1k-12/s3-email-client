package sesclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/smithy-go"
)

// SESClient defines the interface for sending emails via Amazon SES
type SESClient interface {
	// SendEmail sends an email via Amazon SES
	SendEmail(ctx context.Context, msg *EmailMessage) error

	// Close releases any resources held by the client
	Close() error
}

// EmailMessage represents an email to be sent via SES
type EmailMessage struct {
	From       string
	To         []string
	Subject    string
	Body       string
	InReplyTo  string // Optional - for email threading
	References string // Optional - for email threading
}

// Config holds configuration for the SES client
type Config struct {
	Region     string
	AWSProfile string // Optional, uses default credentials if empty
}

// constructRFC5322Message builds an RFC 5322 compliant email message
func constructRFC5322Message(msg *EmailMessage) string {
	var builder strings.Builder

	// From header (required)
	builder.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))

	// To header (required)
	builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))

	// Subject header (required)
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))

	// In-Reply-To header (optional - only if present)
	if msg.InReplyTo != "" {
		builder.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", msg.InReplyTo))
	}

	// References header (optional - only if present)
	if msg.References != "" {
		builder.WriteString(fmt.Sprintf("References: %s\r\n", msg.References))
	}

	// Date header (required) - RFC 5322 format
	// Format: Mon, 02 Jan 2006 15:04:05 -0700
	date := time.Now().Format(time.RFC1123Z)
	builder.WriteString(fmt.Sprintf("Date: %s\r\n", date))

	// MIME-Version header
	builder.WriteString("MIME-Version: 1.0\r\n")

	// Content-Type header
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")

	// Empty line separating headers from body
	builder.WriteString("\r\n")

	// Body with proper line endings
	// Replace any \n with \r\n for RFC 5322 compliance
	body := strings.ReplaceAll(msg.Body, "\r\n", "\n") // Normalize first
	body = strings.ReplaceAll(body, "\n", "\r\n")      // Then convert to CRLF
	builder.WriteString(body)

	return builder.String()
}

// defaultSESClient is the concrete implementation of SESClient using AWS SDK v2
type defaultSESClient struct {
	client *ses.Client
}

// NewSESClient creates a new SES client with the provided configuration
func NewSESClient(ctx context.Context, cfg Config) (SESClient, error) {
	// Build AWS config options
	configOpts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	// Add profile if specified
	if cfg.AWSProfile != "" {
		configOpts = append(configOpts, config.WithSharedConfigProfile(cfg.AWSProfile))
	}

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SES service client
	sesClient := ses.NewFromConfig(awsCfg)

	return &defaultSESClient{
		client: sesClient,
	}, nil
}

// SendEmail sends an email via Amazon SES using the SendRawEmail API
func (c *defaultSESClient) SendEmail(ctx context.Context, msg *EmailMessage) error {
	// Construct RFC 5322 compliant message
	rawMessage := constructRFC5322Message(msg)

	// Prepare SendRawEmail input
	input := &ses.SendRawEmailInput{
		RawMessage: &types.RawMessage{
			Data: []byte(rawMessage),
		},
	}

	// Send the email
	_, err := c.client.SendRawEmail(ctx, input)
	if err != nil {
		return handleSESError(err, msg.From)
	}

	return nil
}

// handleSESError translates SES API errors into user-friendly error messages
func handleSESError(err error, fromAddress string) error {
	// Check for specific SES error types
	var messageRejectedException *types.MessageRejected
	if errors.As(err, &messageRejectedException) {
		// This typically indicates unverified sender or other message rejection
		return fmt.Errorf("email rejected by SES: %s. If the sender address is not verified, please verify %s in the AWS SES console", messageRejectedException.ErrorMessage(), fromAddress)
	}

	var mailFromDomainNotVerifiedException *types.MailFromDomainNotVerifiedException
	if errors.As(err, &mailFromDomainNotVerifiedException) {
		return fmt.Errorf("sender email address not verified in Amazon SES. Please verify %s in the AWS SES console", fromAddress)
	}

	var accountSendingPausedException *types.AccountSendingPausedException
	if errors.As(err, &accountSendingPausedException) {
		return fmt.Errorf("SES account sending is paused. Please check your AWS SES account status")
	}

	var configurationSetDoesNotExistException *types.ConfigurationSetDoesNotExistException
	if errors.As(err, &configurationSetDoesNotExistException) {
		return fmt.Errorf("SES configuration set does not exist: %s", configurationSetDoesNotExistException.ErrorMessage())
	}

	var configurationSetSendingPausedException *types.ConfigurationSetSendingPausedException
	if errors.As(err, &configurationSetSendingPausedException) {
		return fmt.Errorf("SES configuration set sending is paused: %s", configurationSetSendingPausedException.ErrorMessage())
	}

	// Check for throttling errors
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "Throttling", "ThrottlingException":
			return fmt.Errorf("SES rate limit exceeded. Please try again in a moment")
		case "RequestLimitExceeded":
			return fmt.Errorf("SES sending quota exceeded. Please wait or request a quota increase in the AWS SES console")
		case "InvalidParameterValue":
			return fmt.Errorf("invalid email parameter: %s", apiErr.ErrorMessage())
		case "UnauthorizedOperation", "UnrecognizedClientException", "InvalidClientTokenId":
			return fmt.Errorf("AWS authentication failed. Please check your credentials and permissions")
		case "AccessDenied", "AccessDeniedException":
			return fmt.Errorf("access denied: insufficient permissions to send email via SES. Please check your IAM permissions")
		}
	}

	// Check for network/connection errors
	if strings.Contains(err.Error(), "connection") || 
	   strings.Contains(err.Error(), "timeout") || 
	   strings.Contains(err.Error(), "network") ||
	   strings.Contains(err.Error(), "dial") {
		return fmt.Errorf("network error while connecting to SES: %w. Please check your internet connection", err)
	}

	// Generic error fallback
	return fmt.Errorf("failed to send email via SES: %w", err)
}

// Close releases any resources held by the client
func (c *defaultSESClient) Close() error {
	// AWS SDK v2 clients don't require explicit cleanup
	// This method is provided for interface compatibility and future extensibility
	return nil
}
