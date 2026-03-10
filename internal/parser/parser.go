package parser

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jaytaylor/html2text"
	"github.com/jhillyerd/enmime"
)

// EmailParser defines the interface for parsing raw email data
type EmailParser interface {
	// Parse converts raw email bytes into a structured Email object
	Parse(data []byte) (*Email, error)
}

// Email represents a parsed email with headers, body, and attachments
type Email struct {
	Subject     string
	From        string
	To          []string
	Cc          []string
	Date        time.Time
	Body        string
	HTMLBody    string
	Attachments []Attachment
	
	// Email threading headers for reply functionality
	MessageID  string   // Message-ID header
	ReplyTo    string   // Reply-To header (optional)
	References []string // References header for threading
}

// Attachment represents an email attachment with metadata
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
}

// enmimeParser implements EmailParser using the enmime library
type enmimeParser struct{}

// NewParser creates a new EmailParser instance
func NewParser() EmailParser {
	return &enmimeParser{}
}

// Parse parses raw email bytes into a structured Email object
// It extracts headers, body content (both plain text and HTML), and attachment metadata
// Returns an error if the email is malformed or cannot be parsed
func (p *enmimeParser) Parse(data []byte) (*Email, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty email data")
	}

	// Parse the email using enmime
	envelope, err := enmime.ReadEnvelope(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse email: %w", err)
	}

	// Extract date
	date, err := envelope.Date()
	if err != nil {
		// If date parsing fails, use zero time rather than failing completely
		date = time.Time{}
	}

	// Extract To addresses
	toAddrs := extractAddresses(envelope.GetHeaderValues("To"))

	// Extract Cc addresses
	ccAddrs := extractAddresses(envelope.GetHeaderValues("Cc"))

	// Extract body content
	body := envelope.Text
	htmlBody := envelope.HTML

	// If there's no plain text body but there is HTML, convert HTML to text
	if body == "" && htmlBody != "" {
		plainText, err := convertHTMLToText(htmlBody)
		if err == nil {
			body = plainText
		}
	}

	// Extract attachments metadata
	attachments := make([]Attachment, 0, len(envelope.Attachments))
	for _, att := range envelope.Attachments {
		attachments = append(attachments, Attachment{
			Filename:    att.FileName,
			ContentType: att.ContentType,
			Size:        int64(len(att.Content)),
		})
	}

	// Extract email threading headers
	messageID := envelope.GetHeader("Message-ID")
	replyTo := envelope.GetHeader("Reply-To")
	references := extractReferences(envelope.GetHeaderValues("References"))

	email := &Email{
		Subject:     envelope.GetHeader("Subject"),
		From:        envelope.GetHeader("From"),
		To:          toAddrs,
		Cc:          ccAddrs,
		Date:        date,
		Body:        body,
		HTMLBody:    htmlBody,
		Attachments: attachments,
		MessageID:   messageID,
		ReplyTo:     replyTo,
		References:  references,
	}

	return email, nil
}

// extractAddresses parses email address headers into a slice of strings
func extractAddresses(headers []string) []string {
	if len(headers) == 0 {
		return []string{}
	}

	addresses := make([]string, 0, len(headers))
	for _, header := range headers {
		if header != "" {
			addresses = append(addresses, header)
		}
	}

	return addresses
}

// extractReferences parses References header into a slice of message IDs
func extractReferences(headers []string) []string {
	if len(headers) == 0 {
		return []string{}
	}

	references := make([]string, 0, len(headers))
	for _, header := range headers {
		if header != "" {
			references = append(references, header)
		}
	}

	return references
}

// convertHTMLToText converts HTML content to plain text
func convertHTMLToText(html string) (string, error) {
	text, err := html2text.FromString(html, html2text.Options{
		PrettyTables: true,
		OmitLinks:    false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to text: %w", err)
	}
	return text, nil
}
