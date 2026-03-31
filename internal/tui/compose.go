package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// Compose view styles
var (
	composeFieldLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Bold(true).
				Width(10)

	composeFieldValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	composeDividerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("238"))

	composeReplyContextStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("241")).
					Italic(true)

	composeFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("235")).
				Padding(0, 1)

	composeSendingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62")).
				Padding(1, 4).
				Bold(true)
)

// renderComposeView renders the full-screen reply/compose panel.
//
// Layout (all rows inside the rounded-border panel):
//
//	To:      <address>
//	Subject: <subject>
//	────────────────────────────────────
//	<textarea — fills remaining height>
//	────────────────────────────────────
//	↩  original-from  ·  date
func (m *Model) renderComposeView() string {
	if m.composeData == nil {
		return "Error: No compose data available"
	}

	// Panel fills the full screen minus one row reserved for the footer bar.
	panelH := m.height - 1
	panelW := m.width

	// innerW is the usable content width: panel minus border (1 each side = 2) and
	// padding (1 each side = 2).  The panel style must be Width(innerW+2) so that
	// lipgloss word-wraps at (innerW+2)-leftPad-rightPad = innerW, matching the
	// content exactly.  Content height inside the border: panelH - top/bottom border (2).
	innerW := panelW - 4
	if innerW < 20 {
		innerW = 20
	}
	innerH := panelH - 2
	if innerH < 8 {
		innerH = 8
	}

	// Fixed rows inside the panel (reply has an extra context line):
	//   To field       1
	//   Subject field  1
	//   top divider    1
	//   bottom divider 1
	//   context line   1  (reply only)
	fixedRows := 5
	if m.composeIsNew {
		fixedRows = 4
	}
	taH := innerH - fixedRows
	if taH < 3 {
		taH = 3
	}

	// Resize textarea to fill the available space every frame.
	m.composeInput.SetWidth(innerW)
	m.composeInput.SetHeight(taH)

	// Assemble panel content.
	var b strings.Builder

	if m.composeIsNew {
		// Editable textinput fields for new email
		toInputW := innerW - 10 - 1 // label width 10 + space 1
		if toInputW < 10 {
			toInputW = 10
		}
		m.composeToInput.Width = toInputW
		m.composeSubjectInput.Width = toInputW
		b.WriteString(composeFieldLabelStyle.Render("To:") + " " + m.composeToInput.View())
		b.WriteString("\n")
		b.WriteString(composeFieldLabelStyle.Render("Subject:") + " " + m.composeSubjectInput.View())
	} else {
		b.WriteString(renderComposeFieldRow("To", m.composeData.To, innerW))
		b.WriteString("\n")
		b.WriteString(renderComposeFieldRow("Subject", m.composeData.Subject, innerW))
	}
	b.WriteString("\n")
	b.WriteString(composeDividerStyle.Render(strings.Repeat("─", innerW)))
	b.WriteString("\n")
	b.WriteString(m.composeInput.View())
	b.WriteString("\n")
	b.WriteString(composeDividerStyle.Render(strings.Repeat("─", innerW)))
	if replyCtx := m.renderReplyContext(innerW); replyCtx != "" {
		b.WriteString("\n")
		b.WriteString(replyCtx)
	}

	// Wrap in a full-screen rounded-border panel.
	// Width must be innerW+2 because lipgloss word-wraps at (Width - leftPad - rightPad).
	// Padding(0,1) consumes 2 chars, so wrap fires at (innerW+2)-1-1 = innerW, which
	// matches the content width exactly and prevents every line from wrapping.
	panel := lipgloss.NewStyle().
		Width(innerW + 2).
		Height(innerH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(0, 1).
		Render(b.String())

	if m.composeSending {
		return m.renderSendingOverlay()
	}

	if errMsg := m.renderErrorMessage(); errMsg != "" {
		return panel + "\n" + errMsg
	}

	return panel + "\n" + m.renderComposeFooter()
}

// renderComposeFieldRow renders a "Label:   value" row with consistent alignment.
func renderComposeFieldRow(label, value string, width int) string {
	// composeFieldLabelStyle.Width is 10 — the value starts right after.
	const labelW = 10
	maxValW := width - labelW - 1 // -1 for the space between label and value
	if maxValW < 0 {
		maxValW = 0
	}
	if len(value) > maxValW {
		value = value[:maxValW-3] + "..."
	}
	return composeFieldLabelStyle.Render(label+":") + " " + composeFieldValueStyle.Render(value)
}

// renderReplyContext renders a compact one-line summary of the original email.
func (m *Model) renderReplyContext(width int) string {
	if m.composeData == nil || m.composeData.OriginalEmail == nil {
		return ""
	}
	orig := m.composeData.OriginalEmail
	date := orig.Date.Format("Jan 02, 2006 at 3:04 PM")
	text := fmt.Sprintf("↩  %s  ·  %s", orig.From, date)
	if len(text) > width {
		text = text[:width-3] + "..."
	}
	return composeReplyContextStyle.Render(text)
}

// renderComposeFooter renders the key-hint bar at the bottom of the compose view.
func (m *Model) renderComposeFooter() string {
	var hint string
	if m.composeIsNew {
		hint = "  Ctrl+S  send email   ·   Tab  next field   ·   Esc  cancel"
	} else {
		hint = "  Ctrl+S  send reply   ·   Esc  cancel"
	}
	// Fill remaining width so the dark background spans the full terminal.
	padLen := m.width - 2 - len(hint) // -2 for Padding(0,1) in the style
	if padLen > 0 {
		hint += strings.Repeat(" ", padLen)
	}
	return composeFooterStyle.Render(hint)
}

// renderSendingOverlay shows a centred "Sending…" indicator while the send is in progress.
func (m *Model) renderSendingOverlay() string {
	box := composeSendingStyle.Render("  Sending…  ")
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("240")),
	)
}

// initComposeTextInput initialises a single-line textinput used for the To/Subject fields
// in new-email compose mode.
func initComposeTextInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 500
	ti.PromptStyle = lipgloss.NewStyle() // no prompt glyph
	ti.Prompt = ""
	ti.TextStyle = composeFieldValueStyle
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	return ti
}

// initComposeTextarea initialises and returns a configured textarea for the reply body.
func initComposeTextarea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Write your reply…"
	ta.CharLimit = 10000
	ta.ShowLineNumbers = false
	// Remove the default "┃ " thick-border prompt — it bleeds visually onto the next row.
	ta.Prompt = ""
	// Replace the default black cursor-line background (Dark: "0") with a subtle highlight.
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("237"))
	ta.Focus()
	return ta
}

// renderErrorMessage renders a timed error banner (auto-hides after 5 s).
func (m *Model) renderErrorMessage() string {
	if m.errorMessage == "" {
		return ""
	}

	if !m.errorDisplayTime.IsZero() {
		if time.Since(m.errorDisplayTime).Seconds() >= 5.0 {
			m.errorMessage = ""
			m.errorDisplayTime = time.Time{}
			return ""
		}
	}

	errStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Bold(true).
		Width(m.width)

	return errStyle.Render("Error: " + m.errorMessage)
}

// mapSESError translates SES API errors into user-friendly messages.
func mapSESError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	errLower := strings.ToLower(errMsg)

	if strings.Contains(errLower, "not verified") ||
		strings.Contains(errLower, "email address is not verified") ||
		strings.Contains(errLower, "messagerejected") {
		return "Email address not verified in Amazon SES. Please verify your sender email address in the AWS SES console before sending emails."
	}

	if strings.Contains(errLower, "quota") ||
		strings.Contains(errLower, "daily sending quota") ||
		strings.Contains(errLower, "maximum send rate") {
		return "SES sending quota exceeded. You have reached your daily sending limit. Please wait 24 hours or request a quota increase in the AWS SES console."
	}

	if strings.Contains(errLower, "authentication") ||
		strings.Contains(errLower, "credentials") ||
		strings.Contains(errLower, "access denied") ||
		strings.Contains(errLower, "unauthorized") ||
		strings.Contains(errLower, "invalid security token") {
		return "AWS authentication failed. Please check your AWS credentials and ensure they are properly configured (environment variables, ~/.aws/credentials, or IAM role)."
	}

	if strings.Contains(errLower, "throttl") ||
		strings.Contains(errLower, "rate limit") ||
		strings.Contains(errLower, "too many requests") {
		return "SES rate limit exceeded. You are sending emails too quickly. Please wait a moment and try again."
	}

	if strings.Contains(errLower, "network") ||
		strings.Contains(errLower, "connection") ||
		strings.Contains(errLower, "timeout") ||
		strings.Contains(errLower, "dial") ||
		strings.Contains(errLower, "no such host") {
		return "Network error: Failed to connect to Amazon SES. Please check your internet connection and try again."
	}

	return fmt.Sprintf("Failed to send email: %s", errMsg)
}
