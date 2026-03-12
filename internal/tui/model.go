package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"s3emailclient/internal/navigation"
	"s3emailclient/internal/parser"
	"s3emailclient/internal/response"
)

// LoadEmailMsg is sent when an email should be loaded
type LoadEmailMsg struct {
	Key string
}

// EmailLoadedMsg is sent when an email has been successfully loaded and parsed
type EmailLoadedMsg struct {
	Email       *Email
	ParserEmail *parser.Email // Original parser.Email for response actions
}

// EmailLoadErrorMsg is sent when email loading fails
type EmailLoadErrorMsg struct {
	Err error
}

// Pane represents which pane is currently focused
type Pane int

const (
	ListPane Pane = iota
	ContentPane
)

// EmailListItem represents a single email in the list view
type EmailListItem struct {
	Key     string
	Subject string
	From    string
	Date    time.Time
}

// Email represents a parsed email with full content
type Email struct {
	Subject     string
	From        string
	To          []string
	Cc          []string
	Date        time.Time
	Body        string
	HTMLBody    string
	Attachments []Attachment
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
}

// Model represents the Bubble Tea model for the TUI application
type Model struct {
	// Email data
	emailList          []EmailListItem
	selectedIndex      int
	currentEmail       *Email
	currentEmailKey    string        // Key of currently displayed email
	loadingEmailKey    string        // Key of email currently being loaded
	currentParserEmail *parser.Email // Original parser.Email for response actions

	// UI state
	focusedPane     Pane
	listViewport    viewport.Model
	contentViewport viewport.Model

	// Compose view state
	composeMode    bool
	composeData    *response.ComposeData
	composeInput   textarea.Model
	composeSending bool

	// Status
	loading       bool
	statusMessage string
	err           error

	// Error message display
	errorMessage     string
	errorDisplayTime time.Time

	// Terminal dimensions
	width  int
	height int

	// Navigation handler for processing keyboard input
	navHandler NavigationHandler

	// Response handler for email response workflow
	responseHandler response.ResponseHandler

	// Callback for loading emails when selection changes
	onLoadEmail func(key string) tea.Cmd
}

// NavigationHandler is an interface for processing keyboard input
// This is defined here to avoid circular dependencies with the navigation package
type NavigationHandler interface {
	HandleKey(key string, state *navigation.State) navigation.Action
}

// Init initializes the Bubble Tea model
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportSizes()
		return m, nil

	case tea.KeyMsg:
		// Handle textarea input in compose mode
		if m.composeMode {
			// Handle Esc cancel action
			if msg.String() == "esc" {
				// Set composeMode to false, clear composeData and composeInput
				m.composeMode = false
				m.composeData = nil
				m.composeInput.Reset()
				m.composeSending = false
				m.err = nil
				m.statusMessage = ""

				// Update navigation state to set ComposeMode to false
				// (This happens automatically when navHandler.HandleKey is called next)

				// Return to list view without confirmation
				return m, nil
			}

			// Handle Ctrl+S send action
			if msg.String() == "ctrl+s" {
				// Set composeSending to true
				m.composeSending = true

				// Call response handler SendResponse with composeData
				if m.responseHandler != nil && m.composeData != nil {
					ctx := context.Background()
					err := m.responseHandler.SendResponse(ctx, m.composeData)

					if err != nil {
						// If error, set composeSending to false, display error message, remain in compose view
						m.composeSending = false
						m.err = err
						m.statusMessage = ""
						return m, nil
					}

					// If success, set composeMode to false, display success message, return to list view
					m.composeMode = false
					m.composeSending = false
					m.composeData = nil
					m.statusMessage = "Email sent successfully"
					m.err = nil

					// Clear the compose input
					m.composeInput.Reset()

					return m, nil
				}

				// If no response handler, set error and remain in compose view
				m.composeSending = false
				m.err = fmt.Errorf("response handler not configured")
				return m, nil
			}

			// Handle regular textarea input
			var cmd tea.Cmd
			m.composeInput, cmd = m.composeInput.Update(msg)

			// Store updated textarea value in composeData.Body
			if m.composeData != nil {
				m.composeData.Body = m.composeInput.Value()
			}

			return m, cmd
		}

		// Wire keyboard events to NavigationHandler
		if m.navHandler != nil {
			// Build navigation state
			state := &navigation.State{
				FocusedPane:   navigation.Pane(m.focusedPane),
				SelectedIndex: m.selectedIndex,
				EmailCount:    len(m.emailList),
				ContentScroll: m.contentViewport.YOffset,
				MaxScroll:     m.contentViewport.TotalLineCount(),
				CurrentEmail:  m.currentParserEmail,
				ComposeMode:   m.composeMode,
			}

			// Get action from navigation handler
			action := m.navHandler.HandleKey(msg.String(), state)

			// Execute action and update model state
			return m.executeAction(action)
		}

	case LoadEmailMsg:
		// Set loading state when email loading starts
		m.loading = true
		m.err = nil
		m.statusMessage = ""        // Don't set statusMessage, let loading flag handle it
		m.loadingEmailKey = msg.Key // Track which email we're loading
		return m, nil

	case EmailLoadedMsg:
		// Update model with loaded email
		m.currentEmail = msg.Email
		m.currentParserEmail = msg.ParserEmail
		m.currentEmailKey = m.loadingEmailKey // Update the displayed email key
		m.loadingEmailKey = ""                // Clear loading key
		m.loading = false
		m.statusMessage = ""
		m.err = nil
		m.refreshContentViewport()
		return m, nil

	case EmailLoadErrorMsg:
		// Handle email loading error
		m.loading = false
		m.statusMessage = ""
		m.err = msg.Err
		m.currentEmail = nil
		m.currentEmailKey = "" // Clear the displayed key on error
		m.loadingEmailKey = "" // Clear loading key
		return m, nil
	}

	// Update viewports based on focused pane
	var cmd tea.Cmd
	if m.focusedPane == ListPane {
		m.listViewport, cmd = m.listViewport.Update(msg)
	} else {
		m.contentViewport, cmd = m.contentViewport.Update(msg)
	}

	return m, cmd
}

// executeAction executes a navigation action and updates model state accordingly
// executeAction executes a navigation action and updates model state accordingly
func (m *Model) executeAction(action navigation.Action) (tea.Model, tea.Cmd) {
	switch a := action.(type) {
	case *navigation.MoveSelectionAction:
		// Update selection index
		oldIndex := m.selectedIndex
		m.selectedIndex += a.Direction

		// Ensure selection stays within bounds (defensive check)
		if m.selectedIndex < 0 {
			m.selectedIndex = 0
		}
		if m.selectedIndex >= len(m.emailList) {
			m.selectedIndex = len(m.emailList) - 1
		}

		// Refresh list viewport to update selection highlighting
		m.refreshListViewport()

		// Trigger email loading if selection changed
		if oldIndex != m.selectedIndex && m.selectedIndex < len(m.emailList) {
			emailKey := m.emailList[m.selectedIndex].Key

			// Skip loading if we're already viewing this email
			if m.currentEmailKey == emailKey && m.currentEmail != nil {
				// Already viewing this email, no need to reload
				// Ensure loading state is cleared
				m.loading = false
				m.loadingEmailKey = ""
				return m, nil
			}

			// If we're currently loading this same email, don't send another load request
			if m.loadingEmailKey == emailKey && m.loading {
				// Already loading this email, don't duplicate the request
				return m, nil
			}

			if m.onLoadEmail != nil {
				// Return both LoadEmailMsg (for UI state) and the async load command
				return m, tea.Batch(
					func() tea.Msg { return LoadEmailMsg{Key: emailKey} },
					m.onLoadEmail(emailKey),
				)
			}
		}

		return m, nil

	case *navigation.ScrollContentAction:
		// Scroll content viewport
		if a.Lines > 0 {
			// Scroll down
			m.contentViewport.LineDown(a.Lines)
		} else if a.Lines < 0 {
			// Scroll up
			m.contentViewport.LineUp(-a.Lines)
		}
		return m, nil

	case *navigation.ResponseAction:
		// Handle email response workflow
		if a.Email == nil {
			m.err = fmt.Errorf("no email selected for response")
			return m, nil
		}

		// Call response handler InitiateResponse with email from action
		// Note: Using context.Background() for now; could be enhanced with proper context management
		composeData, err := m.responseHandler.InitiateResponse(context.Background(), a.Email)
		if err != nil {
			// Display error message and remain in list view
			m.err = err
			m.statusMessage = ""
			return m, nil
		}

		// Success: transition to compose mode
		m.composeMode = true
		m.composeData = composeData
		m.composeInput = initComposeTextarea()
		m.err = nil
		m.statusMessage = ""

		// Navigation state ComposeMode will be updated on next HandleKey call
		// (The navigation handler checks state.ComposeMode to disable navigation keys)

		return m, nil

	case *navigation.QuitAction:
		// Quit the application
		return m, tea.Quit

	case *navigation.NoOpAction:
		// No action needed
		return m, nil

	default:
		// Unknown action type, do nothing
		return m, nil
	}
}

// View renders the TUI
func (m *Model) View() string {
	// If in compose mode, render compose view instead of split-pane layout
	if m.composeMode {
		return m.renderComposeView()
	}

	// Render both panes side by side
	listPane := m.renderEmailListPane()
	contentPane := m.renderEmailContentPane()

	// Split the panes into lines for side-by-side rendering
	listLines := strings.Split(listPane, "\n")
	contentLines := strings.Split(contentPane, "\n")

	// Calculate pane widths (40/60 split with 1 char separator)
	// Ensure minimum widths to prevent negative values
	minWidth := 10
	if m.width < minWidth*2+1 {
		// Terminal too small, just show list pane
		return m.renderEmailListPane() + "\n" + m.renderStatusBar()
	}

	listWidth := m.width * 40 / 100
	separatorWidth := 1
	contentWidth := m.width - listWidth - separatorWidth

	// Ensure minimum widths
	if listWidth < minWidth {
		listWidth = minWidth
		contentWidth = m.width - listWidth - separatorWidth
	}
	if contentWidth < minWidth {
		contentWidth = minWidth
		listWidth = m.width - contentWidth - separatorWidth
	}

	// Ensure we have enough lines for both panes
	maxLines := len(listLines)
	if len(contentLines) > maxLines {
		maxLines = len(contentLines)
	}

	// Build the split-pane view
	var result strings.Builder
	for i := 0; i < maxLines; i++ {
		// Get list pane line (or empty if out of bounds)
		listLine := ""
		if i < len(listLines) {
			listLine = listLines[i]
		}

		// Pad or truncate list line to fit width
		listLine = padOrTruncate(listLine, listWidth)

		// Get content pane line (or empty if out of bounds)
		contentLine := ""
		if i < len(contentLines) {
			contentLine = contentLines[i]
		}

		// Pad or truncate content line to fit width
		contentLine = padOrTruncate(contentLine, contentWidth)

		// Combine the lines with separator
		result.WriteString(listLine)
		result.WriteString(separatorStyle.Render("│"))
		result.WriteString(contentLine)

		if i < maxLines-1 {
			result.WriteString("\n")
		}
	}

	// Add status bar at the bottom
	statusBar := m.renderStatusBar()
	if statusBar != "" {
		result.WriteString("\n")
		result.WriteString(statusBar)
	}

	// Add error message if present
	errorMsg := m.renderErrorMessage()
	if errorMsg != "" {
		result.WriteString("\n")
		result.WriteString(errorMsg)
	}

	return result.String()
}

// padOrTruncate pads or truncates a string to the specified width
func padOrTruncate(s string, width int) string {
	// Handle invalid widths
	if width <= 0 {
		return ""
	}

	// Remove ANSI escape codes for length calculation
	visibleLen := visualLength(s)

	if visibleLen > width {
		// Truncate (accounting for ANSI codes is complex, so we'll use a simple approach)
		return s[:width]
	}

	// Pad with spaces
	padding := width - visibleLen
	return s + strings.Repeat(" ", padding)
}

// visualLength calculates the visible length of a string (ignoring ANSI codes)
func visualLength(s string) int {
	// Simple implementation that counts runes, ignoring ANSI escape sequences
	length := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		length++
	}

	return length
}

// wrapText wraps text to fit within the specified width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		if len(line) <= width {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		// Wrap long lines
		words := strings.Fields(line)
		if len(words) == 0 {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			// Check if adding the next word would exceed width
			if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				// Start a new line
				wrappedLines = append(wrappedLines, currentLine)
				currentLine = word
			}
		}
		wrappedLines = append(wrappedLines, currentLine)
	}

	return strings.Join(wrappedLines, "\n")
}

// renderStatusBar renders the status bar at the bottom
func (m *Model) renderStatusBar() string {
	if m.statusMessage != "" {
		return statusBarStyle.Render(m.statusMessage)
	}

	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}

	if m.loading {
		return loadingStyle.Render("Loading email...")
	}

	// Default status showing keybindings
	status := "j/k: list | J/K: scroll | q: quit | r: reply"
	return statusBarStyle.Render(status)
}

// updateViewportSizes updates viewport dimensions based on terminal size
func (m *Model) updateViewportSizes() {
	// Reserve space for status bar (1 line)
	availableHeight := m.height - 1
	if availableHeight < 1 {
		availableHeight = 1
	}

	// List pane takes 40% of width, separator takes 1 char, content pane takes the rest
	listWidth := m.width * 40 / 100
	separatorWidth := 1
	contentWidth := m.width - listWidth - separatorWidth

	// Ensure minimum widths
	if listWidth < 10 {
		listWidth = 10
	}
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Update list viewport dimensions (reserve 1 line for header)
	m.listViewport.Width = listWidth
	listViewportHeight := availableHeight - 1
	if listViewportHeight < 1 {
		listViewportHeight = 1
	}
	m.listViewport.Height = listViewportHeight

	// Update content viewport dimensions
	m.contentViewport.Width = contentWidth
	m.contentViewport.Height = availableHeight

	// Refresh viewport content to apply new dimensions
	if len(m.emailList) > 0 {
		m.refreshListViewport()
	}

	if m.currentEmail != nil {
		m.refreshContentViewport()
	}
}

// refreshListViewport refreshes the list viewport content
func (m *Model) refreshListViewport() {
	var content string
	for i, email := range m.emailList {
		content += m.renderEmailListItem(email, i == m.selectedIndex)
		if i < len(m.emailList)-1 {
			content += "\n"
		}
	}
	m.listViewport.SetContent(content)
}

// refreshContentViewport refreshes the content viewport content
func (m *Model) refreshContentViewport() {
	if m.currentEmail == nil {
		return
	}

	var content string

	// Render subject
	content += subjectStyle.Render(m.currentEmail.Subject) + "\n\n"

	// Render headers
	content += m.renderHeader("From:", m.currentEmail.From)
	content += m.renderHeader("To:", m.formatRecipients(m.currentEmail.To))

	if len(m.currentEmail.Cc) > 0 {
		content += m.renderHeader("Cc:", m.formatRecipients(m.currentEmail.Cc))
	}

	content += m.renderHeader("Date:", m.currentEmail.Date.Format("Mon, Jan 02, 2006 at 3:04 PM"))

	// Render body content
	body := m.currentEmail.Body
	if body == "" && m.currentEmail.HTMLBody != "" {
		body = m.htmlToPlainText(m.currentEmail.HTMLBody)
	}

	if body != "" {
		content += bodyStyle.Render(body)
	}

	// Render attachments
	if len(m.currentEmail.Attachments) > 0 {
		content += "\n\n" + m.renderAttachments(m.currentEmail.Attachments)
	}

	m.contentViewport.SetContent(content)
}

// renderEmailListPane renders the email list pane with styling
func (m *Model) renderEmailListPane() string {
	if len(m.emailList) == 0 {
		return m.renderEmptyList()
	}

	// Create header with email count (rendered outside viewport)
	emailCount := len(m.emailList)
	countStr := fmt.Sprintf("%d emails", emailCount)
	if emailCount == 1 {
		countStr = "1 email"
	}
	header := listHeaderStyle.Render(countStr)

	var content string
	for i, email := range m.emailList {
		content += m.renderEmailListItem(email, i == m.selectedIndex)
		if i < len(m.emailList)-1 {
			content += "\n"
		}
	}

	// Set viewport content WITHOUT header
	m.listViewport.SetContent(content)

	// Calculate line offset for selected item (each item is 2 lines)
	selectedLine := m.selectedIndex * 2

	// Scroll viewport to show selected item
	if selectedLine < m.listViewport.YOffset {
		m.listViewport.YOffset = selectedLine
	} else if selectedLine >= m.listViewport.YOffset+m.listViewport.Height {
		m.listViewport.YOffset = selectedLine - m.listViewport.Height + 2
	}

	// Return header above viewport
	return header + "\n" + m.listViewport.View()
}

// renderEmailListItem renders a single email list item with styling
func (m *Model) renderEmailListItem(email EmailListItem, selected bool) string {
	// Format the date
	dateStr := email.Date.Format("Jan 02, 2006")

	// Calculate available width (viewport width minus padding from style)
	// normalItemStyle and selectedItemStyle both have Padding(0, 1) = 2 chars total
	availableWidth := m.listViewport.Width - 2

	// If viewport not initialized (e.g., in tests), use a reasonable default
	if availableWidth <= 0 {
		availableWidth = 50
	}

	// Subject line gets full width
	maxSubjectLen := availableWidth
	subject := email.Subject
	if len(subject) > maxSubjectLen {
		subject = subject[:maxSubjectLen-3] + "..."
	}

	// Second line: sender + " • " + date
	// Date is fixed at 12 chars, separator is 3 chars (" • ")
	dateLen := len(dateStr) // Should be 12 for "Jan 02, 2006"
	separatorLen := 3       // " • "
	maxSenderLen := availableWidth - dateLen - separatorLen
	if maxSenderLen < 5 {
		maxSenderLen = 5 // Minimum sender length
	}

	sender := email.From
	if len(sender) > maxSenderLen {
		sender = sender[:maxSenderLen-3] + "..."
	}

	// Build the item text
	item := subject + "\n" + sender + " • " + dateStr

	// Apply styling based on selection
	if selected {
		return selectedItemStyle.Render(item)
	}
	return normalItemStyle.Render(item)
}

// renderEmptyList renders the empty list message
func (m *Model) renderEmptyList() string {
	message := "No emails found"
	return emptyListStyle.Render(message)
}

// renderEmailContentPane renders the email content pane with full email details
func (m *Model) renderEmailContentPane() string {
	if m.currentEmail == nil {
		return m.renderEmptyContent()
	}

	var content string

	// Render subject
	content += subjectStyle.Render(m.currentEmail.Subject) + "\n\n"

	// Render headers
	content += m.renderHeader("From:", m.currentEmail.From)
	content += m.renderHeader("To:", m.formatRecipients(m.currentEmail.To))

	if len(m.currentEmail.Cc) > 0 {
		content += m.renderHeader("Cc:", m.formatRecipients(m.currentEmail.Cc))
	}

	content += m.renderHeader("Date:", m.currentEmail.Date.Format("Mon, Jan 02, 2006 at 3:04 PM"))

	// Render body content
	body := m.currentEmail.Body
	if body == "" && m.currentEmail.HTMLBody != "" {
		// If no plain text body, use HTML body (converted to plain text)
		body = m.htmlToPlainText(m.currentEmail.HTMLBody)
	}

	if body != "" {
		// Wrap body text to fit viewport width (accounting for padding)
		wrappedBody := wrapText(body, m.contentViewport.Width-2)
		content += bodyStyle.Render(wrappedBody)
	}

	// Render attachments
	if len(m.currentEmail.Attachments) > 0 {
		content += "\n\n" + m.renderAttachments(m.currentEmail.Attachments)
	}

	// Set viewport content
	m.contentViewport.SetContent(content)

	return m.contentViewport.View()
}

// renderHeader renders a single email header field
func (m *Model) renderHeader(label, value string) string {
	return headerLabelStyle.Render(label) + " " + headerValueStyle.Render(value) + "\n"
}

// formatRecipients formats a list of email addresses
func (m *Model) formatRecipients(recipients []string) string {
	if len(recipients) == 0 {
		return ""
	}

	result := recipients[0]
	for i := 1; i < len(recipients); i++ {
		result += ", " + recipients[i]
	}
	return result
}

// renderAttachments renders the attachment metadata
func (m *Model) renderAttachments(attachments []Attachment) string {
	var content string
	content += headerLabelStyle.Render("Attachments:") + "\n"

	for _, att := range attachments {
		// Format size in human-readable format
		sizeStr := formatSize(att.Size)
		attInfo := att.Filename + " (" + att.ContentType + ", " + sizeStr + ")"
		content += "  " + attachmentStyle.Render(attInfo) + "\n"
	}

	return content
}

// formatSize formats a byte size into human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// htmlToPlainText converts HTML content to plain text
// This is a simple implementation that strips HTML tags
func (m *Model) htmlToPlainText(html string) string {
	// Simple HTML tag stripping - in production, consider using a proper HTML parser
	// For now, this basic implementation removes tags and decodes common entities

	text := html

	// Replace common HTML entities
	replacements := map[string]string{
		"&nbsp;": " ",
		"&lt;":   "<",
		"&gt;":   ">",
		"&amp;":  "&",
		"&quot;": "\"",
		"&#39;":  "'",
		"<br>":   "\n",
		"<br/>":  "\n",
		"<br />": "\n",
		"</p>":   "\n\n",
		"</div>": "\n",
	}

	for entity, replacement := range replacements {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	// Remove HTML tags using a simple regex-like approach
	// This is basic but sufficient for the TUI viewer
	var result strings.Builder
	inTag := false

	for _, char := range text {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(char)
		}
	}

	return strings.TrimSpace(result.String())
}

// renderEmptyContent renders the empty content message
func (m *Model) renderEmptyContent() string {
	message := "Select an email to view its content"
	return emptyContentStyle.Render(message)
}

// SetEmailList updates the model with a list of emails
func (m *Model) SetEmailList(emails []EmailListItem) {
	// Sort emails by date (descending order, newest first)
	sort.SliceStable(emails, func(i, j int) bool {
		// Handle zero-value dates (place at end)
		if emails[i].Date.IsZero() && !emails[j].Date.IsZero() {
			return false
		}
		if !emails[i].Date.IsZero() && emails[j].Date.IsZero() {
			return true
		}
		// Both zero or both non-zero: compare normally (descending)
		return emails[i].Date.After(emails[j].Date)
	})

	m.emailList = emails
	if len(emails) > 0 && m.selectedIndex >= len(emails) {
		m.selectedIndex = 0
	}
	m.refreshListViewport()
}

// SetCurrentEmail updates the model with the currently displayed email
func (m *Model) SetCurrentEmail(email *Email) {
	m.currentEmail = email
	m.refreshContentViewport()
}

// SetLoading sets the loading state
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// SetError sets the error state
func (m *Model) SetError(err error) {
	m.err = err
}

// SetStatusMessage sets a status message
func (m *Model) SetStatusMessage(msg string) {
	m.statusMessage = msg
}

// SetNavigationHandler sets the navigation handler for processing keyboard input
func (m *Model) SetNavigationHandler(handler NavigationHandler) {
	m.navHandler = handler
}

// SetResponseHandler sets the response handler for email response workflow
func (m *Model) SetResponseHandler(handler response.ResponseHandler) {
	m.responseHandler = handler
}

// SetOnLoadEmail sets the callback function for loading emails
func (m *Model) SetOnLoadEmail(callback func(key string) tea.Cmd) {
	m.onLoadEmail = callback
}
