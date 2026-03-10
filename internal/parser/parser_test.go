package parser

import (
	"strings"
	"testing"
)

func TestParse_ValidEmail(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Test Email
Date: Mon, 02 Jan 2006 15:04:05 -0700
Content-Type: text/plain; charset=utf-8

This is a test email body.`

	parser := NewParser()
	email, err := parser.Parse([]byte(rawEmail))

	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if email.Subject != "Test Email" {
		t.Errorf("Subject = %q, want %q", email.Subject, "Test Email")
	}

	if email.From != "sender@example.com" {
		t.Errorf("From = %q, want %q", email.From, "sender@example.com")
	}

	if len(email.To) != 1 || email.To[0] != "recipient@example.com" {
		t.Errorf("To = %v, want [recipient@example.com]", email.To)
	}

	if !strings.Contains(email.Body, "This is a test email body") {
		t.Errorf("Body = %q, want to contain 'This is a test email body'", email.Body)
	}
}

func TestParse_EmptyData(t *testing.T) {
	parser := NewParser()
	_, err := parser.Parse([]byte{})

	if err == nil {
		t.Error("Parse() with empty data should return error")
	}

	if !strings.Contains(err.Error(), "empty email data") {
		t.Errorf("error = %q, want to contain 'empty email data'", err.Error())
	}
}

func TestParse_MalformedEmail(t *testing.T) {
	rawEmail := `This is not a valid email format`

	parser := NewParser()
	email, err := parser.Parse([]byte(rawEmail))

	// enmime is quite forgiving, so it may still parse something
	// We just verify it doesn't panic and returns a result
	if err != nil {
		// If it returns an error, that's acceptable for malformed input
		if !strings.Contains(err.Error(), "failed to parse") {
			t.Errorf("error = %q, want to contain 'failed to parse'", err.Error())
		}
	} else if email == nil {
		t.Error("Parse() returned nil email without error")
	}
}

func TestParse_HTMLEmail(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: HTML Email
Date: Mon, 02 Jan 2006 15:04:05 -0700
Content-Type: text/html; charset=utf-8

<html><body><h1>Hello</h1><p>This is HTML content.</p></body></html>`

	parser := NewParser()
	email, err := parser.Parse([]byte(rawEmail))

	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if email.HTMLBody == "" {
		t.Error("HTMLBody should not be empty for HTML email")
	}

	// Body should be populated with converted plain text
	if email.Body == "" {
		t.Error("Body should be populated with converted HTML")
	}

	if !strings.Contains(email.Body, "Hello") {
		t.Errorf("Body = %q, want to contain 'Hello'", email.Body)
	}
}

func TestParse_MultipleRecipients(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient1@example.com, recipient2@example.com
Cc: cc1@example.com, cc2@example.com
Subject: Multiple Recipients
Date: Mon, 02 Jan 2006 15:04:05 -0700

Test body`

	parser := NewParser()
	email, err := parser.Parse([]byte(rawEmail))

	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	// Note: enmime may parse comma-separated addresses as a single string
	// or split them depending on the format
	if len(email.To) == 0 {
		t.Error("To addresses should not be empty")
	}

	if len(email.Cc) == 0 {
		t.Error("Cc addresses should not be empty")
	}
}

func TestParse_WithAttachments(t *testing.T) {
	// Multipart email with attachment
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Email with Attachment
Date: Mon, 02 Jan 2006 15:04:05 -0700
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="boundary123"

--boundary123
Content-Type: text/plain; charset=utf-8

Email body text

--boundary123
Content-Type: application/pdf; name="document.pdf"
Content-Disposition: attachment; filename="document.pdf"

PDF content here
--boundary123--`

	parser := NewParser()
	email, err := parser.Parse([]byte(rawEmail))

	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if len(email.Attachments) == 0 {
		t.Error("Expected at least one attachment")
	}

	if len(email.Attachments) > 0 {
		att := email.Attachments[0]
		if att.Filename != "document.pdf" {
			t.Errorf("Attachment filename = %q, want %q", att.Filename, "document.pdf")
		}
		if att.ContentType != "application/pdf" {
			t.Errorf("Attachment ContentType = %q, want %q", att.ContentType, "application/pdf")
		}
		if att.Size == 0 {
			t.Error("Attachment size should be greater than 0")
		}
	}
}

func TestParse_DateParsing(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Date Test
Date: Mon, 02 Jan 2006 15:04:05 -0700

Body`

	parser := NewParser()
	email, err := parser.Parse([]byte(rawEmail))

	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	// Check that date was parsed (not zero time)
	if email.Date.IsZero() {
		t.Error("Date should be parsed and not zero")
	}

	expectedYear := 2006
	if email.Date.Year() != expectedYear {
		t.Errorf("Date year = %d, want %d", email.Date.Year(), expectedYear)
	}
}

func TestParse_InvalidDate(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Invalid Date Test
Date: Not a valid date

Body`

	parser := NewParser()
	email, err := parser.Parse([]byte(rawEmail))

	// Should not fail completely, just have zero date
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil (should handle invalid date gracefully)", err)
	}

	// Date should be zero time when parsing fails
	if !email.Date.IsZero() {
		t.Error("Date should be zero time when date parsing fails")
	}
}
