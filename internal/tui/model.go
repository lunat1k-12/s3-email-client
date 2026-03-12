package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"s3emailclient/internal/navigation"
	"s3emailclient/internal/parser"
	"s3emailclient/internal/response"
)

// Pane represents which pane is currently focused
type Pane int

const (
	ListPane Pane = iota
	ContentPane
)

// Model represents the TUI application state
type Model struct {
	// Email data
	emailList          []EmailListItem
	selectedIndex      int
	currentEmail       *Email
	currentEmailKey    string
	loadingEmailKey    string
	currentParserEmail *parser.Email

	// UI state
	focusedPane     Pane
	listViewport    viewport.Model
	contentViewport viewport.Model

	// Compose view state
	composeMode    bool
	composeData    *response.ComposeData
	composeInput   textarea.Model
	composeSending bool

	// Delete confirmation state
	showDeleteModal     bool
	deleteTargetKey     string
	deleteTargetSubject string

	// Auto-load state
	autoLoadPending bool
	emailListLoaded bool

	// Status
	loading          bool
	statusMessage    string
	err              error
	errorMessage     string
	errorDisplayTime time.Time
	lastRefreshTime  time.Time

	// Terminal dimensions
	width  int
	height int

	// Handlers and callbacks
	navHandler      navigation.NavigationHandler
	responseHandler response.ResponseHandler
	onLoadEmail     func(key string) tea.Cmd
	onDeleteEmail   func(key string) tea.Cmd
	onRefreshList   func() tea.Cmd
}

// EmailListItem represents an email in the list view
type EmailListItem struct {
	Key     string
	Subject string
	From    string
	Date    time.Time
}

// Email represents a parsed email for display
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

// Message types for Bubble Tea message passing

// LoadEmailMsg is sent when an email should be loaded
type LoadEmailMsg struct {
	Key string
}

// EmailLoadedMsg is sent when an email has been successfully loaded
type EmailLoadedMsg struct {
	Email       *Email
	ParserEmail *parser.Email
}

// EmailLoadErrorMsg is sent when email loading fails
type EmailLoadErrorMsg struct {
	Err error
}

// DeleteEmailMsg is sent when user confirms deletion
type DeleteEmailMsg struct {
	Key string
}

// EmailDeletedMsg is sent when email deletion succeeds
type EmailDeletedMsg struct {
	Key string
}

// EmailDeleteErrorMsg is sent when email deletion fails
type EmailDeleteErrorMsg struct {
	Key string
	Err error
}

// EmailListLoadedMsg is sent when email list is loaded
type EmailListLoadedMsg struct {
	Emails []EmailListItem
}

// Setter methods

// SetNavigationHandler sets the navigation handler
func (m *Model) SetNavigationHandler(handler navigation.NavigationHandler) {
	m.navHandler = handler
}

// SetResponseHandler sets the response handler
func (m *Model) SetResponseHandler(handler response.ResponseHandler) {
	m.responseHandler = handler
}

// SetOnLoadEmail sets the callback for loading emails
func (m *Model) SetOnLoadEmail(callback func(key string) tea.Cmd) {
	m.onLoadEmail = callback
}

// SetOnDeleteEmail sets the callback for deleting emails
func (m *Model) SetOnDeleteEmail(callback func(key string) tea.Cmd) {
	m.onDeleteEmail = callback
}
// SetOnRefreshList sets the callback for refreshing the email list
func (m *Model) SetOnRefreshList(callback func() tea.Cmd) {
	m.onRefreshList = callback
}

// SetEmailList sets the email list, sorts by date, and refreshes the list viewport
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

// SetCurrentEmail sets the current email and refreshes the content viewport
func (m *Model) SetCurrentEmail(email *Email) {
	m.currentEmail = email
	m.refreshContentViewport()
}

// SetStatusMessage sets a status message
func (m *Model) SetStatusMessage(message string) {
	m.statusMessage = message
}

// SetError sets an error
func (m *Model) SetError(err error) {
	m.err = err
}

// removeEmailFromList removes an email from the list by key
// It adjusts the selectedIndex if needed and clears currentEmail if the deleted email was current
func (m *Model) removeEmailFromList(key string) {
	for i, email := range m.emailList {
		if email.Key == key {
			// Remove the email from the list
			m.emailList = append(m.emailList[:i], m.emailList[i+1:]...)

			// Adjust selected index if it's out of bounds after removal
			if m.selectedIndex >= len(m.emailList) && len(m.emailList) > 0 {
				m.selectedIndex = len(m.emailList) - 1
			}

			// Clear current email if it was deleted
			if m.currentEmailKey == key {
				m.currentEmail = nil
				m.currentEmailKey = ""
			}

			break
		}
	}

	// Refresh list viewport to update display
	m.refreshListViewport()
}
// selectNextEmailAfterDelete selects and loads the next email after deletion
// Returns nil if email list is empty, otherwise returns LoadEmailCmd for email at current selectedIndex
func (m *Model) selectNextEmailAfterDelete() tea.Cmd {
	// Return nil if email list is empty
	if len(m.emailList) == 0 {
		return nil
	}

	// Load the email at the current selected index
	// selectedIndex is already adjusted by removeEmailFromList
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.emailList) {
		nextKey := m.emailList[m.selectedIndex].Key
		if m.onLoadEmail != nil {
			return m.onLoadEmail(nextKey)
		}
	}

	return nil
}

// refreshListViewport updates the list viewport content
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

// refreshContentViewport updates the content viewport with current email
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

// renderNormalView renders the normal view (non-modal)
func (m *Model) renderNormalView() string {
	// If in compose mode, render compose view instead of split-pane layout
	if m.composeMode {
		return m.renderComposeView()
	}

	// Render both panes with borders
	listPane := m.renderEmailListPane()
	contentPane := m.renderEmailContentPane()

	// Calculate pane widths (40/60 split with 1 char separator)
	// Account for border width (2 chars per side = 4 total per pane)
	minWidth := 14 // Minimum to accommodate borders
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

	// Apply borders to panes
	// The Width() sets the content width, so we subtract 4 (2 chars per side for border + padding)
	listPaneWithBorder := listPaneBorderStyle.
		Width(listWidth - 4).
		Height(m.height - 2). // Reserve space for status bar
		Render(listPane)
	
	contentPaneWithBorder := contentPaneBorderStyle.
		Width(contentWidth - 4).
		Height(m.height - 2). // Reserve space for status bar
		Render(contentPane)

	// Combine panes side by side using lipgloss JoinHorizontal
	combined := lipgloss.JoinHorizontal(
		lipgloss.Top,
		listPaneWithBorder,
		separatorStyle.Render("│"),
		contentPaneWithBorder,
	)

	// Add status bar at the bottom
	statusBar := m.renderStatusBar()
	if statusBar != "" {
		combined += "\n" + statusBar
	}

	// Add error message if present
	errorMsg := m.renderErrorMessage()
	if errorMsg != "" {
		combined += "\n" + errorMsg
	}

	return combined
}

// renderDeleteModal renders the delete confirmation modal
func (m *Model) renderDeleteModal() string {
	// Create modal overlay
	modalWidth := 60
	modalHeight := 8

	subject := m.deleteTargetSubject
	if subject == "" {
		subject = m.deleteTargetKey
	}

	// Truncate subject if too long
	maxSubjectLen := modalWidth - 4
	if len(subject) > maxSubjectLen {
		subject = subject[:maxSubjectLen-3] + "..."
	}

	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")). // Red border
		Padding(1, 2)

	modalContent := fmt.Sprintf("Delete Email?\n\n%s\n\n[Y]es  [N]o", subject)
	modal := modalStyle.Render(modalContent)

	// Overlay modal on base view using lipgloss Place
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("240")),
	)
}

// renderEmptyList renders the empty list message
func (m *Model) renderEmptyList() string {
	return emptyListStyle.Render("No emails found")
}

// renderEmptyContent renders the empty content message
func (m *Model) renderEmptyContent() string {
	return emptyContentStyle.Render("Select an email to view its content")
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

// renderHeader renders an email header field
func (m *Model) renderHeader(label, value string) string {
	labelStyled := headerLabelStyle.Render(label)
	valueStyled := headerValueStyle.Render(value)
	return labelStyled + " " + valueStyled + "\n"
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

// Init implements the Bubble Tea Init interface
func (m *Model) Init() tea.Cmd {
	// Initialize viewports if not already initialized
	if m.listViewport.Width == 0 {
		m.listViewport = viewport.New(30, 20)
	}
	if m.contentViewport.Width == 0 {
		m.contentViewport = viewport.New(50, 20)
	}
	
	m.autoLoadPending = true
	return nil
}

// Update implements the Bubble Tea Update interface
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
				m.composeMode = false
				m.composeData = nil
				m.composeInput.Reset()
				m.composeSending = false
				m.err = nil
				m.statusMessage = ""
				return m, nil
			}

			// Handle Shift+R refresh action even in compose mode
			if msg.String() == "R" {
				m.statusMessage = "Refreshing email list..."
				m.err = nil
				if m.onRefreshList != nil {
					return m, m.onRefreshList()
				}
				return m, nil
			}

			// Handle Ctrl+S send action
			if msg.String() == "ctrl+s" {
				m.composeSending = true

				if m.responseHandler != nil && m.composeData != nil {
					ctx := context.Background()
					err := m.responseHandler.SendResponse(ctx, m.composeData)

					if err != nil {
						m.composeSending = false
						m.err = err
						m.statusMessage = ""
						return m, nil
					}

					m.composeMode = false
					m.composeSending = false
					m.composeData = nil
					m.statusMessage = "Email sent successfully"
					m.err = nil
					m.composeInput.Reset()
					return m, nil
				}

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
				FocusedPane:       navigation.Pane(m.focusedPane),
				SelectedIndex:     m.selectedIndex,
				EmailCount:        len(m.emailList),
				ContentScroll:     m.contentViewport.YOffset,
				MaxScroll:         m.contentViewport.TotalLineCount(),
				CurrentEmail:      m.currentParserEmail,
				CurrentEmailKey:   m.currentEmailKey,
				ComposeMode:       m.composeMode,
				DeleteModalActive: m.showDeleteModal,
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
		m.statusMessage = ""
		m.loadingEmailKey = msg.Key
		return m, nil

	case EmailLoadedMsg:
		// Update model with loaded email
		m.currentEmail = msg.Email
		m.currentParserEmail = msg.ParserEmail
		m.currentEmailKey = m.loadingEmailKey
		m.loadingEmailKey = ""
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
		m.currentEmailKey = ""
		m.loadingEmailKey = ""
		return m, nil

	case EmailListLoadedMsg:
		// Set email list
		m.SetEmailList(msg.Emails)
		
		// Clear status message and errors
		m.statusMessage = ""
		m.err = nil
		
		// Update last refresh time
		m.lastRefreshTime = time.Now()
		
		// Mark email list as loaded
		m.emailListLoaded = true
		
		// Auto-load first email if pending and list not empty
		if m.autoLoadPending && len(msg.Emails) > 0 {
			m.selectedIndex = 0
			m.autoLoadPending = false
			if m.onLoadEmail != nil {
				return m, m.onLoadEmail(msg.Emails[0].Key)
			}
		}
		
		// Clear auto-load pending flag even if list is empty
		m.autoLoadPending = false
		
		return m, nil

	case DeleteEmailMsg:
		// Call onDeleteEmail callback with key
		if m.onDeleteEmail != nil {
			return m, m.onDeleteEmail(msg.Key)
		}
		return m, nil

	case EmailDeletedMsg:
		// Remove email from list
		m.removeEmailFromList(msg.Key)

		// Select next email
		nextCmd := m.selectNextEmailAfterDelete()

		// Show success message
		m.SetStatusMessage("Email deleted successfully")

		return m, nextCmd

	case EmailDeleteErrorMsg:
		// Set error message with failure details
		m.SetError(msg.Err)

		// Hide delete modal
		m.showDeleteModal = false

		// Return model without changes to email list
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

// visualLength calculates the visual length of a string (for layout purposes)
func visualLength(s string) int {
	return len(s)
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
	status := "j/k: list | J/K: scroll | q: quit | r: reply | d: delete | R: refresh"
	return statusBarStyle.Render(status)
}

// renderEmailListPane renders the email list pane with styling
func (m *Model) renderEmailListPane() string {
	if len(m.emailList) == 0 {
		return m.renderEmptyList()
	}

	// Create header with email count and last refresh time (rendered outside viewport)
	emailCount := len(m.emailList)
	countStr := fmt.Sprintf("%d emails", emailCount)
	if emailCount == 1 {
		countStr = "1 email"
	}
	
	// Add last refresh time if available
	var headerText string
	if !m.lastRefreshTime.IsZero() {
		refreshStr := m.lastRefreshTime.Format("15:04:05")
		headerText = fmt.Sprintf("%s | Last refreshed: %s", countStr, refreshStr)
	} else {
		headerText = countStr
	}
	
	header := listHeaderStyle.Render(headerText)

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

// executeAction executes a navigation action and updates model state accordingly
func (m *Model) executeAction(action navigation.Action) (tea.Model, tea.Cmd) {
	switch a := action.(type) {
	case *navigation.MoveSelectionAction:
		// Update selection index
		oldIndex := m.selectedIndex
		m.selectedIndex += a.Direction

		// Ensure selection stays within bounds
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
				m.loading = false
				m.loadingEmailKey = ""
				return m, nil
			}

			// If we're currently loading this same email, don't send another load request
			if m.loadingEmailKey == emailKey && m.loading {
				return m, nil
			}

			if m.onLoadEmail != nil {
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
			m.contentViewport.LineDown(a.Lines)
		} else if a.Lines < 0 {
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
		composeData, err := m.responseHandler.InitiateResponse(context.Background(), a.Email)
		if err != nil {
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

		return m, nil

	case *navigation.DeleteAction:
		// Show confirmation modal
		m.showDeleteModal = true
		m.deleteTargetKey = a.Key
		m.deleteTargetSubject = a.Subject
		return m, nil

	case *navigation.ConfirmDeleteAction:
		// Hide modal and execute delete
		m.showDeleteModal = false
		key := m.deleteTargetKey
		m.deleteTargetKey = ""
		m.deleteTargetSubject = ""
		return m, func() tea.Msg {
			return DeleteEmailMsg{Key: key}
		}

	case *navigation.CancelDeleteAction:
		// Hide modal without deleting
		m.showDeleteModal = false
		m.deleteTargetKey = ""
		m.deleteTargetSubject = ""
		return m, nil

	case *navigation.RefreshAction:
		// Refresh email list from S3
		m.statusMessage = "Refreshing email list..."
		m.err = nil
		if m.onRefreshList != nil {
			return m, m.onRefreshList()
		}
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

// updateViewportSizes updates viewport dimensions based on terminal size
func (m *Model) updateViewportSizes() {
	// Reserve space for status bar (1 line)
	availableHeight := m.height - 1
	if availableHeight < 1 {
		availableHeight = 1
	}

	// List pane takes 40% of width, separator takes 1 char, content pane takes the rest
	// Account for borders: 4 chars per pane (2 on each side)
	listWidth := m.width * 40 / 100
	separatorWidth := 1
	contentWidth := m.width - listWidth - separatorWidth

	// Ensure minimum widths
	if listWidth < 14 {
		listWidth = 14
	}
	if contentWidth < 14 {
		contentWidth = 14
	}

	// Calculate inner viewport widths (subtract border and padding: 4 chars total)
	listViewportWidth := listWidth - 4
	contentViewportWidth := contentWidth - 4

	// Ensure positive widths
	if listViewportWidth < 1 {
		listViewportWidth = 1
	}
	if contentViewportWidth < 1 {
		contentViewportWidth = 1
	}

	// Update list viewport dimensions (reserve 1 line for header, 2 for borders)
	m.listViewport.Width = listViewportWidth
	listViewportHeight := availableHeight - 3
	if listViewportHeight < 1 {
		listViewportHeight = 1
	}
	m.listViewport.Height = listViewportHeight

	// Update content viewport dimensions (account for borders: 2 lines)
	m.contentViewport.Width = contentViewportWidth
	contentViewportHeight := availableHeight - 2
	if contentViewportHeight < 1 {
		contentViewportHeight = 1
	}
	m.contentViewport.Height = contentViewportHeight

	// Refresh viewport content to apply new dimensions
	if len(m.emailList) > 0 {
		m.refreshListViewport()
	}

	if m.currentEmail != nil {
		m.refreshContentViewport()
	}
}

