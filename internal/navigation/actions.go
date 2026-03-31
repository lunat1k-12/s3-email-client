package navigation

import (
	"s3emailclient/internal/parser"
)

// Action represents a navigation action that can be executed
type Action interface {
	// Execute performs the action
	// Returns true if the action was executed successfully
	Execute() bool
}

// MoveSelectionAction moves the selection in the email list
type MoveSelectionAction struct {
	Direction int // -1 for up, +1 for down
}

// Execute implements Action for MoveSelectionAction
func (a *MoveSelectionAction) Execute() bool {
	// Actual execution will be handled by the application controller
	// This method exists to satisfy the Action interface
	return true
}

// ScrollContentAction scrolls the content pane
type ScrollContentAction struct {
	Lines int // Positive for down, negative for up
}

// Execute implements Action for ScrollContentAction
func (a *ScrollContentAction) Execute() bool {
	return true
}

// QuitAction exits the application
type QuitAction struct{}

// Execute implements Action for QuitAction
func (a *QuitAction) Execute() bool {
	return true
}

// ResponseAction initiates email response workflow
type ResponseAction struct {
	Email *parser.Email
}

// Execute implements Action for ResponseAction
func (a *ResponseAction) Execute() bool {
	return true
}

// NoOpAction represents no action (used for boundary cases)
type NoOpAction struct{}

// Execute implements Action for NoOpAction
func (a *NoOpAction) Execute() bool {
	return false
}
// DeleteAction initiates email deletion workflow
type DeleteAction struct {
	Key     string
	Subject string
}

// Execute implements Action for DeleteAction
func (a *DeleteAction) Execute() bool {
	return true
}

// ConfirmDeleteAction confirms the deletion
type ConfirmDeleteAction struct{}

// Execute implements Action for ConfirmDeleteAction
func (a *ConfirmDeleteAction) Execute() bool {
	return true
}

// CancelDeleteAction cancels the deletion
type CancelDeleteAction struct{}
// RefreshAction refreshes the email list from S3
type RefreshAction struct{}

// Execute implements Action for RefreshAction
func (a *RefreshAction) Execute() bool {
	return true
}

// Execute implements Action for CancelDeleteAction
func (a *CancelDeleteAction) Execute() bool {
	return true
}

// OpenLinkPickerAction opens the link picker overlay for the current email
type OpenLinkPickerAction struct{}

// Execute implements Action for OpenLinkPickerAction
func (a *OpenLinkPickerAction) Execute() bool {
	return true
}

// NewEmailAction initiates composing a new outbound email
type NewEmailAction struct{}

// Execute implements Action for NewEmailAction
func (a *NewEmailAction) Execute() bool {
	return true
}
