package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

// Compose view styles
var (
	// composeHeaderStyle is the style for the compose view header section
	composeHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Bold(true).
				MarginBottom(1)

	// composeFieldLabelStyle is the style for field labels (To, Subject)
	composeFieldLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Bold(true).
				Width(10)

	// composeFieldValueStyle is the style for field values
	composeFieldValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	// composeOriginalHeaderStyle is the style for original email context
	composeOriginalHeaderStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("241")).
					Italic(true).
					MarginBottom(1)

	// composeFooterStyle is the style for the footer instructions
	composeFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("235")).
				Padding(0, 1).
				Bold(true)

	// composeSendingStyle is the style for the sending indicator overlay
	composeSendingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62")).
				Padding(1, 2).
				Bold(true).
				Align(lipgloss.Center)

	// composeSectionStyle is the style for section dividers
	composeSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				MarginTop(1).
				MarginBottom(1)
)

// renderComposeView renders the full-screen compose view
func (m *Model) renderComposeView() string {
	if m.composeData == nil {
		return "Error: No compose data available"
	}

	var content strings.Builder

	// Calculate available dimensions
	availableHeight := m.height - 1 // Reserve 1 line for footer
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Render header section with original email context
	content.WriteString(m.renderOriginalEmailContext())
	content.WriteString("\n")

	// Render section divider
	divider := strings.Repeat("─", m.width)
	content.WriteString(composeSectionStyle.Render(divider))
	content.WriteString("\n")

	// Render To field (pre-populated, read-only)
	content.WriteString(m.renderComposeField("To:", m.composeData.To))
	content.WriteString("\n")

	// Render Subject field (pre-populated, read-only)
	content.WriteString(m.renderComposeField("Subject:", m.composeData.Subject))
	content.WriteString("\n\n")

	// Render body label
	content.WriteString(composeFieldLabelStyle.Render("Body:"))
	content.WriteString("\n")

	// Render textarea for body composition
	if m.composeInput.Focused() {
		content.WriteString(m.composeInput.View())
	} else {
		// If not focused, show the current value
		content.WriteString(composeFieldValueStyle.Render(m.composeData.Body))
	}

	// Render footer with instructions
	footer := m.renderComposeFooter()
	content.WriteString("\n")
	content.WriteString(footer)

	// Add error message if present
	errorMsg := m.renderErrorMessage()
	if errorMsg != "" {
		content.WriteString("\n")
		content.WriteString(errorMsg)
	}

	// If sending, render overlay
	if m.composeSending {
		return m.renderSendingOverlay(content.String())
	}

	return content.String()
}

// renderOriginalEmailContext renders the original email context header
func (m *Model) renderOriginalEmailContext() string {
	if m.composeData == nil || m.composeData.OriginalEmail == nil {
		return ""
	}

	var content strings.Builder

	// Header title
	content.WriteString(composeHeaderStyle.Render("Reply to:"))
	content.WriteString("\n")

	orig := m.composeData.OriginalEmail

	// Original From
	content.WriteString(composeOriginalHeaderStyle.Render(
		fmt.Sprintf("  From: %s", orig.From),
	))
	content.WriteString("\n")

	// Original To
	if len(orig.To) > 0 {
		toStr := strings.Join(orig.To, ", ")
		content.WriteString(composeOriginalHeaderStyle.Render(
			fmt.Sprintf("  To: %s", toStr),
		))
		content.WriteString("\n")
	}

	// Original Date
	content.WriteString(composeOriginalHeaderStyle.Render(
		fmt.Sprintf("  Date: %s", orig.Date.Format("Mon, Jan 02, 2006 at 3:04 PM")),
	))
	content.WriteString("\n")

	// Original Subject
	content.WriteString(composeOriginalHeaderStyle.Render(
		fmt.Sprintf("  Subject: %s", orig.Subject),
	))

	return content.String()
}

// renderComposeField renders a single compose field (To or Subject)
func (m *Model) renderComposeField(label, value string) string {
	labelStyled := composeFieldLabelStyle.Render(label)
	valueStyled := composeFieldValueStyle.Render(value)
	return labelStyled + " " + valueStyled
}

// renderComposeFooter renders the footer with instructions
func (m *Model) renderComposeFooter() string {
	instructions := "Ctrl+S: Send | Esc: Cancel"
	
	// Pad to full width
	padding := m.width - visualLength(instructions) - 2 // -2 for padding
	if padding < 0 {
		padding = 0
	}
	
	footer := instructions + strings.Repeat(" ", padding)
	return composeFooterStyle.Render(footer)
}

// renderSendingOverlay renders the sending indicator overlay
func (m *Model) renderSendingOverlay(baseContent string) string {
	// Split base content into lines
	lines := strings.Split(baseContent, "\n")

	// Calculate center position for overlay
	overlayMessage := "Sending email..."
	overlayHeight := 3
	overlayWidth := len(overlayMessage) + 4 // +4 for padding

	// Calculate vertical center
	centerY := len(lines) / 2

	// Create overlay box
	overlay := composeSendingStyle.Width(overlayWidth).Render(overlayMessage)
	overlayLines := strings.Split(overlay, "\n")

	// Insert overlay into content
	startY := centerY - overlayHeight/2
	if startY < 0 {
		startY = 0
	}

	for i, overlayLine := range overlayLines {
		lineIndex := startY + i
		if lineIndex >= 0 && lineIndex < len(lines) {
			// Center the overlay horizontally
			leftPadding := (m.width - overlayWidth) / 2
			if leftPadding < 0 {
				leftPadding = 0
			}

			// Replace the line with overlay
			lines[lineIndex] = strings.Repeat(" ", leftPadding) + overlayLine
		}
	}

	return strings.Join(lines, "\n")
}
// initComposeTextarea initializes and configures the textarea component for email body composition
func initComposeTextarea() textarea.Model {
	ta := textarea.New()

	// Configure multi-line support
	ta.SetHeight(10)
	ta.SetWidth(80)

	// Set placeholder text
	ta.Placeholder = "Type your message here..."

	// Set focus to textarea
	ta.Focus()

	// Enable character limit (optional, can be adjusted or removed)
	ta.CharLimit = 10000

	// Show line numbers (optional)
	ta.ShowLineNumbers = false

	return ta
}
// renderErrorMessage renders an error message at the bottom of the screen with red styling
// The error message auto-dismisses after 5 seconds based on errorDisplayTime
func (m *Model) renderErrorMessage() string {
	// Check if error message exists
	if m.errorMessage == "" {
		return ""
	}

	// Check if error should be auto-dismissed (5 seconds = 5000000000 nanoseconds)
	if !m.errorDisplayTime.IsZero() {
		elapsed := time.Since(m.errorDisplayTime)
		if elapsed.Seconds() >= 5.0 {
			// Clear the error message after 5 seconds
			m.errorMessage = ""
			m.errorDisplayTime = time.Time{}
			return ""
		}
	}

	// Render error message with red styling
	errorMessageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")). // Red color
		Background(lipgloss.Color("235")). // Dark background
		Padding(0, 1).
		Bold(true).
		Width(m.width)

	return errorMessageStyle.Render("Error: " + m.errorMessage)
}


// mapSESError translates SES API errors into user-friendly messages
// This function maps common AWS SES error types to actionable error messages
// that users can understand and act upon.
func mapSESError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	errLower := strings.ToLower(errMsg)

	// Map unverified sender error
	if strings.Contains(errLower, "not verified") || 
	   strings.Contains(errLower, "email address is not verified") ||
	   strings.Contains(errLower, "messagerejected") {
		return "Email address not verified in Amazon SES. Please verify your sender email address in the AWS SES console before sending emails."
	}

	// Map quota exceeded error
	if strings.Contains(errLower, "quota") || 
	   strings.Contains(errLower, "daily sending quota") ||
	   strings.Contains(errLower, "maximum send rate") {
		return "SES sending quota exceeded. You have reached your daily sending limit. Please wait 24 hours or request a quota increase in the AWS SES console."
	}

	// Map authentication error
	if strings.Contains(errLower, "authentication") || 
	   strings.Contains(errLower, "credentials") ||
	   strings.Contains(errLower, "access denied") ||
	   strings.Contains(errLower, "unauthorized") ||
	   strings.Contains(errLower, "invalid security token") {
		return "AWS authentication failed. Please check your AWS credentials and ensure they are properly configured (environment variables, ~/.aws/credentials, or IAM role)."
	}

	// Map throttling error
	if strings.Contains(errLower, "throttl") || 
	   strings.Contains(errLower, "rate limit") ||
	   strings.Contains(errLower, "too many requests") {
		return "SES rate limit exceeded. You are sending emails too quickly. Please wait a moment and try again."
	}

	// Map network errors
	if strings.Contains(errLower, "network") || 
	   strings.Contains(errLower, "connection") ||
	   strings.Contains(errLower, "timeout") ||
	   strings.Contains(errLower, "dial") ||
	   strings.Contains(errLower, "no such host") {
		return "Network error: Failed to connect to Amazon SES. Please check your internet connection and try again."
	}

	// Default: return the original error message
	return fmt.Sprintf("Failed to send email: %s", errMsg)
}
