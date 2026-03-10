package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
)

func TestRenderEmptyList(t *testing.T) {
	m := Model{
		emailList: []EmailListItem{},
		width:     80,
		height:    24,
	}

	result := m.renderEmptyList()
	if !strings.Contains(result, "No emails found") {
		t.Errorf("Expected empty list message, got: %s", result)
	}
}

func TestRenderEmailListItem(t *testing.T) {
	email := EmailListItem{
		Key:     "test-key",
		Subject: "Test Subject",
		From:    "sender@example.com",
		Date:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	m := Model{}

	// Test unselected item
	result := m.renderEmailListItem(email, false)
	if !strings.Contains(result, "Test Subject") {
		t.Errorf("Expected subject in output, got: %s", result)
	}
	if !strings.Contains(result, "sender@example.com") {
		t.Errorf("Expected sender in output, got: %s", result)
	}
	if !strings.Contains(result, "Jan 15, 2024") {
		t.Errorf("Expected date in output, got: %s", result)
	}

	// Test selected item
	selectedResult := m.renderEmailListItem(email, true)
	if !strings.Contains(selectedResult, "Test Subject") {
		t.Errorf("Expected subject in selected output, got: %s", selectedResult)
	}
}

func TestRenderEmailListItemTruncation(t *testing.T) {
	longSubject := "This is a very long subject line that should be truncated to fit within the display"
	longSender := "verylongemailaddress@example.com"

	email := EmailListItem{
		Key:     "test-key",
		Subject: longSubject,
		From:    longSender,
		Date:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	m := Model{}
	result := m.renderEmailListItem(email, false)

	// Check that truncation occurred (should contain "...")
	if !strings.Contains(result, "...") {
		t.Errorf("Expected truncation indicator in output, got: %s", result)
	}
}

func TestRenderEmailListPane(t *testing.T) {
	emails := []EmailListItem{
		{
			Key:     "email1",
			Subject: "First Email",
			From:    "sender1@example.com",
			Date:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			Key:     "email2",
			Subject: "Second Email",
			From:    "sender2@example.com",
			Date:    time.Date(2024, 1, 16, 11, 45, 0, 0, time.UTC),
		},
	}

	m := Model{
		emailList:     emails,
		selectedIndex: 0,
		width:         80,
		height:        24,
		listViewport:  viewport.New(30, 20),
	}

	result := m.renderEmailListPane()

	// The viewport should contain the email content
	// We can't easily test the exact output due to styling,
	// but we can verify the method doesn't panic
	if result == "" {
		t.Error("Expected non-empty output from renderEmailListPane")
	}
}

func TestUpdateViewportSizes(t *testing.T) {
	m := Model{
		width:  100,
		height: 30,
	}

	m.updateViewportSizes()

	// List pane should be 40% of width (no border adjustment)
	expectedWidth := 100 * 40 / 100 // 40
	if m.listViewport.Width != expectedWidth {
		t.Errorf("Expected list viewport width %d, got %d", expectedWidth, m.listViewport.Width)
	}

	// Height should be terminal height minus 1 (for status bar)
	expectedHeight := 30 - 1 // 29
	if m.listViewport.Height != expectedHeight {
		t.Errorf("Expected list viewport height %d, got %d", expectedHeight, m.listViewport.Height)
	}
}

func TestSelectionHighlighting(t *testing.T) {
	emails := []EmailListItem{
		{
			Key:     "email1",
			Subject: "First Email",
			From:    "sender1@example.com",
			Date:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			Key:     "email2",
			Subject: "Second Email",
			From:    "sender2@example.com",
			Date:    time.Date(2024, 1, 16, 11, 45, 0, 0, time.UTC),
		},
	}

	m := Model{
		emailList:     emails,
		selectedIndex: 1, // Select second email
		width:         80,
		height:        24,
		listViewport:  viewport.New(30, 20),
	}

	// Render and verify it doesn't panic
	result := m.renderEmailListPane()
	if result == "" {
		t.Error("Expected non-empty output with selection")
	}
}

func TestRenderEmptyContent(t *testing.T) {
	m := Model{
		currentEmail: nil,
		width:        80,
		height:       24,
	}

	result := m.renderEmptyContent()
	if !strings.Contains(result, "Select an email") {
		t.Errorf("Expected empty content message, got: %s", result)
	}
}

func TestRenderEmailContentPane(t *testing.T) {
	email := &Email{
		Subject: "Test Email Subject",
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Cc:      []string{"cc@example.com"},
		Date:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Body:    "This is the email body content.",
		Attachments: []Attachment{
			{
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				Size:        1024000,
			},
		},
	}

	m := Model{
		currentEmail:    email,
		width:           100,
		height:          30,
		contentViewport: viewport.New(58, 28),
	}

	result := m.renderEmailContentPane()

	// Verify the viewport was populated (we can't easily test styled output)
	if result == "" {
		t.Error("Expected non-empty output from renderEmailContentPane")
	}
}

func TestRenderHeader(t *testing.T) {
	m := Model{}

	result := m.renderHeader("From:", "sender@example.com")
	if !strings.Contains(result, "From:") {
		t.Errorf("Expected header label in output, got: %s", result)
	}
	if !strings.Contains(result, "sender@example.com") {
		t.Errorf("Expected header value in output, got: %s", result)
	}
}

func TestFormatRecipients(t *testing.T) {
	m := Model{}

	tests := []struct {
		name       string
		recipients []string
		expected   string
	}{
		{
			name:       "empty list",
			recipients: []string{},
			expected:   "",
		},
		{
			name:       "single recipient",
			recipients: []string{"user@example.com"},
			expected:   "user@example.com",
		},
		{
			name:       "multiple recipients",
			recipients: []string{"user1@example.com", "user2@example.com", "user3@example.com"},
			expected:   "user1@example.com, user2@example.com, user3@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.formatRecipients(tt.recipients)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRenderAttachments(t *testing.T) {
	m := Model{}

	attachments := []Attachment{
		{
			Filename:    "document.pdf",
			ContentType: "application/pdf",
			Size:        1024000,
		},
		{
			Filename:    "image.jpg",
			ContentType: "image/jpeg",
			Size:        512000,
		},
	}

	result := m.renderAttachments(attachments)

	if !strings.Contains(result, "Attachments:") {
		t.Errorf("Expected attachments header, got: %s", result)
	}
	if !strings.Contains(result, "document.pdf") {
		t.Errorf("Expected first attachment filename, got: %s", result)
	}
	if !strings.Contains(result, "image.jpg") {
		t.Errorf("Expected second attachment filename, got: %s", result)
	}
	if !strings.Contains(result, "application/pdf") {
		t.Errorf("Expected first attachment content type, got: %s", result)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "megabytes",
			bytes:    1024 * 1024,
			expected: "1.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1024 * 1024 * 1024,
			expected: "1.0 GB",
		},
		{
			name:     "partial megabytes",
			bytes:    1536 * 1024, // 1.5 MB
			expected: "1.5 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHtmlToPlainText(t *testing.T) {
	m := Model{}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple text",
			html:     "Hello World",
			expected: "Hello World",
		},
		{
			name:     "with paragraph tags",
			html:     "<p>First paragraph</p><p>Second paragraph</p>",
			expected: "First paragraph\n\nSecond paragraph",
		},
		{
			name:     "with br tags",
			html:     "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "with entities",
			html:     "Hello&nbsp;World &amp; Friends",
			expected: "Hello World & Friends",
		},
		{
			name:     "with div tags",
			html:     "<div>Content 1</div><div>Content 2</div>",
			expected: "Content 1\nContent 2",
		},
		{
			name:     "complex html",
			html:     "<html><body><h1>Title</h1><p>Paragraph with <strong>bold</strong> text.</p></body></html>",
			expected: "TitleParagraph with bold text.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.htmlToPlainText(tt.html)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRenderEmailContentPaneWithHTMLBody(t *testing.T) {
	email := &Email{
		Subject:  "HTML Email",
		From:     "sender@example.com",
		To:       []string{"recipient@example.com"},
		Date:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Body:     "", // No plain text body
		HTMLBody: "<p>This is <strong>HTML</strong> content.</p>",
	}

	m := Model{
		currentEmail:    email,
		width:           100,
		height:          30,
		contentViewport: viewport.New(58, 28),
	}

	result := m.renderEmailContentPane()

	// Verify the viewport was populated
	if result == "" {
		t.Error("Expected non-empty output for HTML email")
	}
}

func TestUpdateViewportSizesWithContentPane(t *testing.T) {
	m := Model{
		width:  100,
		height: 30,
	}

	m.updateViewportSizes()

	// List pane should be 40% of width (no border adjustment)
	expectedListWidth := 100 * 40 / 100 // 40
	if m.listViewport.Width != expectedListWidth {
		t.Errorf("Expected list viewport width %d, got %d", expectedListWidth, m.listViewport.Width)
	}

	// Content pane should be 60% of width minus separator (1 char)
	expectedContentWidth := 100 - (100 * 40 / 100) - 1 // 59
	if m.contentViewport.Width != expectedContentWidth {
		t.Errorf("Expected content viewport width %d, got %d", expectedContentWidth, m.contentViewport.Width)
	}

	// Both should have same height (terminal height minus 1 for status bar)
	expectedHeight := 30 - 1 // 29
	if m.listViewport.Height != expectedHeight {
		t.Errorf("Expected list viewport height %d, got %d", expectedHeight, m.listViewport.Height)
	}
	if m.contentViewport.Height != expectedHeight {
		t.Errorf("Expected content viewport height %d, got %d", expectedHeight, m.contentViewport.Height)
	}
}

func TestRenderStatusBar(t *testing.T) {
	tests := []struct {
		name           string
		model          Model
		expectedText   string
		shouldContain  []string
		shouldNotError bool
	}{
		{
			name: "default status with list pane focused",
			model: Model{
				focusedPane: ListPane,
			},
			shouldContain: []string{"Focus: List", "j/k: navigate", "h/l: switch pane", "q: quit"},
		},
		{
			name: "default status with content pane focused",
			model: Model{
				focusedPane: ContentPane,
			},
			shouldContain: []string{"Focus: Content", "j/k: navigate", "h/l: switch pane", "q: quit"},
		},
		{
			name: "loading indicator",
			model: Model{
				loading: true,
			},
			shouldContain: []string{"Loading..."},
		},
		{
			name: "error message",
			model: Model{
				err: &testError{msg: "connection failed"},
			},
			shouldContain: []string{"Error:", "connection failed"},
		},
		{
			name: "custom status message",
			model: Model{
				statusMessage: "Downloading email from S3...",
			},
			shouldContain: []string{"Downloading email from S3..."},
		},
		{
			name: "status message takes precedence over error",
			model: Model{
				statusMessage: "Custom message",
				err:           &testError{msg: "some error"},
			},
			shouldContain: []string{"Custom message"},
		},
		{
			name: "status message takes precedence over loading",
			model: Model{
				statusMessage: "Custom message",
				loading:       true,
			},
			shouldContain: []string{"Custom message"},
		},
		{
			name: "error takes precedence over loading",
			model: Model{
				err:     &testError{msg: "network error"},
				loading: true,
			},
			shouldContain: []string{"Error:", "network error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.model.renderStatusBar()

			if result == "" {
				t.Error("Expected non-empty status bar output")
			}

			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected status bar to contain %q, got: %s", expected, result)
				}
			}
		})
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestStatusBarInView(t *testing.T) {
	m := Model{
		emailList:     []EmailListItem{},
		width:         80,
		height:        24,
		focusedPane:   ListPane,
		listViewport:  viewport.New(32, 23),
		contentViewport: viewport.New(48, 23),
	}

	view := m.View()

	// Status bar should be present in the view
	if !strings.Contains(view, "Focus: List") {
		t.Error("Expected status bar to be present in view")
	}
}

func TestStatusBarWithLoadingState(t *testing.T) {
	m := Model{
		emailList:     []EmailListItem{},
		width:         80,
		height:        24,
		loading:       true,
		listViewport:  viewport.New(32, 23),
		contentViewport: viewport.New(48, 23),
	}

	view := m.View()

	// Loading indicator should be present in the view
	if !strings.Contains(view, "Loading...") {
		t.Error("Expected loading indicator in view")
	}
}

func TestStatusBarWithErrorState(t *testing.T) {
	m := Model{
		emailList:     []EmailListItem{},
		width:         80,
		height:        24,
		err:           &testError{msg: "S3 bucket not found"},
		listViewport:  viewport.New(32, 23),
		contentViewport: viewport.New(48, 23),
	}

	view := m.View()

	// Error message should be present in the view
	if !strings.Contains(view, "Error:") {
		t.Error("Expected error message in view")
	}
	if !strings.Contains(view, "S3 bucket not found") {
		t.Error("Expected specific error message in view")
	}
}

func TestStatusBarWithCustomMessage(t *testing.T) {
	m := Model{
		emailList:     []EmailListItem{},
		width:         80,
		height:        24,
		statusMessage: "Parsing email content...",
		listViewport:  viewport.New(32, 23),
		contentViewport: viewport.New(48, 23),
	}

	view := m.View()

	// Custom status message should be present in the view
	if !strings.Contains(view, "Parsing email content...") {
		t.Error("Expected custom status message in view")
	}
}

func TestStatusBarPriorityOrder(t *testing.T) {
	// Test that statusMessage has highest priority
	m := Model{
		statusMessage: "Priority message",
		err:           &testError{msg: "error"},
		loading:       true,
		focusedPane:   ListPane,
	}

	result := m.renderStatusBar()
	if !strings.Contains(result, "Priority message") {
		t.Error("Expected statusMessage to have highest priority")
	}
	if strings.Contains(result, "error") || strings.Contains(result, "Loading") {
		t.Error("Expected statusMessage to override error and loading states")
	}

	// Test that error has priority over loading
	m2 := Model{
		err:         &testError{msg: "error message"},
		loading:     true,
		focusedPane: ListPane,
	}

	result2 := m2.renderStatusBar()
	if !strings.Contains(result2, "error message") {
		t.Error("Expected error to have priority over loading")
	}
	if strings.Contains(result2, "Loading") {
		t.Error("Expected error to override loading state")
	}

	// Test that loading has priority over default
	m3 := Model{
		loading:     true,
		focusedPane: ListPane,
	}

	result3 := m3.renderStatusBar()
	if !strings.Contains(result3, "Loading") {
		t.Error("Expected loading indicator")
	}
	if strings.Contains(result3, "Focus:") {
		t.Error("Expected loading to override default status")
	}
}

func TestStatusBarFocusIndicator(t *testing.T) {
	// Test List pane focus
	m1 := Model{
		focusedPane: ListPane,
	}
	result1 := m1.renderStatusBar()
	if !strings.Contains(result1, "Focus: List") {
		t.Error("Expected 'Focus: List' for ListPane")
	}

	// Test Content pane focus
	m2 := Model{
		focusedPane: ContentPane,
	}
	result2 := m2.renderStatusBar()
	if !strings.Contains(result2, "Focus: Content") {
		t.Error("Expected 'Focus: Content' for ContentPane")
	}
}

func TestStatusBarKeybindingHelp(t *testing.T) {
	m := Model{
		focusedPane: ListPane,
	}

	result := m.renderStatusBar()

	// Verify all keybinding hints are present
	expectedHints := []string{
		"j/k: navigate",
		"h/l: switch pane",
		"q: quit",
	}

	for _, hint := range expectedHints {
		if !strings.Contains(result, hint) {
			t.Errorf("Expected keybinding hint %q in status bar, got: %s", hint, result)
		}
	}
}

func TestModel_EmailLoadingMessages(t *testing.T) {
	t.Run("LoadEmailMsg sets loading state", func(t *testing.T) {
		model := &Model{}
		
		msg := LoadEmailMsg{Key: "test.eml"}
		updatedModel, _ := model.Update(msg)
		
		m := updatedModel.(*Model)
		if !m.loading {
			t.Error("LoadEmailMsg should set loading to true")
		}
		if m.statusMessage != "Loading email..." {
			t.Errorf("LoadEmailMsg should set status message, got %q", m.statusMessage)
		}
		if m.err != nil {
			t.Error("LoadEmailMsg should clear error")
		}
	})

	t.Run("EmailLoadedMsg updates current email", func(t *testing.T) {
		model := &Model{
			loading: true,
			statusMessage: "Loading...",
		}
		
		testEmail := &Email{
			Subject: "Test Subject",
			From:    "test@example.com",
			Body:    "Test body",
		}
		
		msg := EmailLoadedMsg{Email: testEmail}
		updatedModel, _ := model.Update(msg)
		
		m := updatedModel.(*Model)
		if m.loading {
			t.Error("EmailLoadedMsg should set loading to false")
		}
		if m.statusMessage != "" {
			t.Error("EmailLoadedMsg should clear status message")
		}
		if m.err != nil {
			t.Error("EmailLoadedMsg should clear error")
		}
		if m.currentEmail == nil {
			t.Fatal("EmailLoadedMsg should set currentEmail")
		}
		if m.currentEmail.Subject != "Test Subject" {
			t.Errorf("currentEmail.Subject = %q, want %q", m.currentEmail.Subject, "Test Subject")
		}
	})

	t.Run("EmailLoadErrorMsg sets error state", func(t *testing.T) {
		model := &Model{
			loading: true,
			currentEmail: &Email{Subject: "Old Email"},
		}
		
		testErr := fmt.Errorf("failed to load email")
		msg := EmailLoadErrorMsg{Err: testErr}
		updatedModel, _ := model.Update(msg)
		
		m := updatedModel.(*Model)
		if m.loading {
			t.Error("EmailLoadErrorMsg should set loading to false")
		}
		if m.statusMessage != "" {
			t.Error("EmailLoadErrorMsg should clear status message")
		}
		if m.err == nil {
			t.Fatal("EmailLoadErrorMsg should set error")
		}
		if m.err.Error() != "failed to load email" {
			t.Errorf("err = %v, want %v", m.err, testErr)
		}
		if m.currentEmail != nil {
			t.Error("EmailLoadErrorMsg should clear currentEmail")
		}
	})
}

func TestModel_SetMethods(t *testing.T) {
	t.Run("SetEmailList updates email list", func(t *testing.T) {
		model := &Model{
			listViewport: viewport.Model{},
		}
		
		emails := []EmailListItem{
			{Key: "email1.eml", Subject: "Test 1"},
			{Key: "email2.eml", Subject: "Test 2"},
		}
		
		model.SetEmailList(emails)
		
		if len(model.emailList) != 2 {
			t.Errorf("SetEmailList() emailList length = %d, want 2", len(model.emailList))
		}
	})

	t.Run("SetCurrentEmail updates current email", func(t *testing.T) {
		model := &Model{
			contentViewport: viewport.Model{},
		}
		
		email := &Email{
			Subject: "Test Email",
			From:    "test@example.com",
		}
		
		model.SetCurrentEmail(email)
		
		if model.currentEmail == nil {
			t.Fatal("SetCurrentEmail() should set currentEmail")
		}
		if model.currentEmail.Subject != "Test Email" {
			t.Errorf("currentEmail.Subject = %q, want %q", model.currentEmail.Subject, "Test Email")
		}
	})

	t.Run("SetLoading updates loading state", func(t *testing.T) {
		model := &Model{}
		
		model.SetLoading(true)
		if !model.loading {
			t.Error("SetLoading(true) should set loading to true")
		}
		
		model.SetLoading(false)
		if model.loading {
			t.Error("SetLoading(false) should set loading to false")
		}
	})

	t.Run("SetError updates error state", func(t *testing.T) {
		model := &Model{}
		
		testErr := fmt.Errorf("test error")
		model.SetError(testErr)
		
		if model.err == nil {
			t.Fatal("SetError() should set error")
		}
		if model.err.Error() != "test error" {
			t.Errorf("err = %v, want %v", model.err, testErr)
		}
	})

	t.Run("SetStatusMessage updates status message", func(t *testing.T) {
		model := &Model{}
		
		model.SetStatusMessage("Test status")
		
		if model.statusMessage != "Test status" {
			t.Errorf("statusMessage = %q, want %q", model.statusMessage, "Test status")
		}
	})
}
