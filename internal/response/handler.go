package response

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"s3emailclient/internal/config"
	"s3emailclient/internal/parser"
	"s3emailclient/internal/sesclient"
)

// ResponseHandler manages the email response workflow
type ResponseHandler interface {
	// InitiateResponse prepares a response to the given email
	// Returns compose data or error if email cannot be replied to
	InitiateResponse(ctx context.Context, email *parser.Email) (*ComposeData, error)

	// InitiateNewEmail prepares a blank compose form for a new outbound email
	InitiateNewEmail(ctx context.Context) (*ComposeData, error)

	// SendResponse sends the composed email via SES
	SendResponse(ctx context.Context, compose *ComposeData) error
}

// ComposeData contains all data needed to compose and send an email response
type ComposeData struct {
	To            string                // Reply-To or From address of original email
	Subject       string                // "Re: " + original subject
	Body          string                // User-composed message body
	InReplyTo     string                // Original Message-ID
	References    string                // Original Message-ID
	OriginalEmail *OriginalEmailContext // For display in compose view
}

// OriginalEmailContext contains the original email's context for display
type OriginalEmailContext struct {
	From    string
	To      []string
	Date    time.Time
	Subject string
}
// DefaultResponseHandler is the concrete implementation of ResponseHandler
type DefaultResponseHandler struct {
	config    *config.Config
	sesClient sesclient.SESClient
}

// NewResponseHandler creates a new DefaultResponseHandler with the provided dependencies
func NewResponseHandler(cfg *config.Config, sesClient sesclient.SESClient) ResponseHandler {
	return &DefaultResponseHandler{
		config:    cfg,
		sesClient: sesClient,
	}
}

// InitiateResponse prepares a response to the given email
// It validates configuration, extracts reply metadata, and builds ComposeData
func (h *DefaultResponseHandler) InitiateResponse(ctx context.Context, email *parser.Email) (*ComposeData, error) {
	// Validate source_email is configured
	if err := h.config.ValidateSourceEmail(); err != nil {
		return nil, err
	}

	// Validate original email has a From address
	if email.From == "" {
		return nil, fmt.Errorf("cannot reply: original email has no sender address")
	}

	// Extract Reply-To header or fall back to From address for To field
	toAddress := email.From
	if email.ReplyTo != "" {
		toAddress = email.ReplyTo
	}

	// Construct subject with "Re: " prefix (idempotent - don't add if already present)
	subject := email.Subject
	if !strings.HasPrefix(subject, "Re: ") {
		subject = "Re: " + subject
	}

	// Extract Message-ID for InReplyTo and References
	// If Message-ID is missing, log warning but proceed with empty values
	inReplyTo := email.MessageID
	references := email.MessageID
	
	if email.MessageID == "" {
		log.Println("Warning: original email has no Message-ID, email threading will not be preserved")
	}

	// Build OriginalEmailContext from email headers
	originalContext := &OriginalEmailContext{
		From:    email.From,
		To:      email.To,
		Date:    email.Date,
		Subject: email.Subject,
	}

	// Return ComposeData
	composeData := &ComposeData{
		To:            toAddress,
		Subject:       subject,
		Body:          "", // User will compose this
		InReplyTo:     inReplyTo,
		References:    references,
		OriginalEmail: originalContext,
	}

	return composeData, nil
}

// InitiateNewEmail prepares a blank ComposeData for composing a new outbound email
func (h *DefaultResponseHandler) InitiateNewEmail(ctx context.Context) (*ComposeData, error) {
	if err := h.config.ValidateSourceEmail(); err != nil {
		return nil, err
	}
	return &ComposeData{
		To:      "",
		Subject: "",
		Body:    "",
	}, nil
}

// SendResponse sends the composed email via SES
// It validates the body, builds an EmailMessage, and calls the SES client
func (h *DefaultResponseHandler) SendResponse(ctx context.Context, compose *ComposeData) error {
	// Validate email body is not empty (log warning if empty, but proceed)
	if strings.TrimSpace(compose.Body) == "" {
		// Log warning but proceed with send as per requirements
		log.Println("Warning: sending email with empty body")
	}

	// Build EmailMessage from ComposeData using source_email as From
	emailMsg := &sesclient.EmailMessage{
		From:       h.config.SourceEmail,
		To:         []string{compose.To},
		Subject:    compose.Subject,
		Body:       compose.Body,
		InReplyTo:  compose.InReplyTo,
		References: compose.References,
	}

	// Call SES client SendEmail method
	if err := h.sesClient.SendEmail(ctx, emailMsg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
