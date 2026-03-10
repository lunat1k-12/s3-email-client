package sesclient

import (
	"strings"
	"testing"
)

func TestConstructRFC5322Message(t *testing.T) {
	tests := []struct {
		name    string
		msg     *EmailMessage
		wantHeaders []string
		wantBody string
	}{
		{
			name: "basic message with all required fields",
			msg: &EmailMessage{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				Body:    "Test body content",
			},
			wantHeaders: []string{
				"From: sender@example.com",
				"To: recipient@example.com",
				"Subject: Test Subject",
				"Date: ",
				"MIME-Version: 1.0",
				"Content-Type: text/plain; charset=UTF-8",
			},
			wantBody: "Test body content",
		},
		{
			name: "message with threading headers",
			msg: &EmailMessage{
				From:       "sender@example.com",
				To:         []string{"recipient@example.com"},
				Subject:    "Re: Original Subject",
				Body:       "Reply body",
				InReplyTo:  "<original-message-id@example.com>",
				References: "<original-message-id@example.com>",
			},
			wantHeaders: []string{
				"From: sender@example.com",
				"To: recipient@example.com",
				"Subject: Re: Original Subject",
				"In-Reply-To: <original-message-id@example.com>",
				"References: <original-message-id@example.com>",
				"Date: ",
				"MIME-Version: 1.0",
				"Content-Type: text/plain; charset=UTF-8",
			},
			wantBody: "Reply body",
		},
		{
			name: "message with multiple recipients",
			msg: &EmailMessage{
				From:    "sender@example.com",
				To:      []string{"recipient1@example.com", "recipient2@example.com"},
				Subject: "Multiple Recipients",
				Body:    "Body content",
			},
			wantHeaders: []string{
				"From: sender@example.com",
				"To: recipient1@example.com, recipient2@example.com",
				"Subject: Multiple Recipients",
			},
			wantBody: "Body content",
		},
		{
			name: "message with multi-line body",
			msg: &EmailMessage{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Multi-line Test",
				Body:    "Line 1\nLine 2\nLine 3",
			},
			wantHeaders: []string{
				"From: sender@example.com",
				"To: recipient@example.com",
				"Subject: Multi-line Test",
			},
			wantBody: "Line 1\r\nLine 2\r\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constructRFC5322Message(tt.msg)

			// Check that result uses CRLF line endings
			if !strings.Contains(result, "\r\n") {
				t.Error("Message should use CRLF line endings")
			}

			// Check for required headers
			for _, header := range tt.wantHeaders {
				if !strings.Contains(result, header) {
					t.Errorf("Message missing expected header: %s", header)
				}
			}

			// Check body content
			if !strings.Contains(result, tt.wantBody) {
				t.Errorf("Message body = %q, want to contain %q", result, tt.wantBody)
			}

			// Verify header/body separation (empty line with CRLF)
			if !strings.Contains(result, "\r\n\r\n") {
				t.Error("Message should have empty line separating headers from body")
			}
		})
	}
}

func TestConstructRFC5322Message_NoThreadingHeaders(t *testing.T) {
	msg := &EmailMessage{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "No Threading",
		Body:    "Body",
	}

	result := constructRFC5322Message(msg)

	// Should NOT contain In-Reply-To or References when not provided
	if strings.Contains(result, "In-Reply-To:") {
		t.Error("Message should not contain In-Reply-To header when not provided")
	}
	if strings.Contains(result, "References:") {
		t.Error("Message should not contain References header when not provided")
	}
}
