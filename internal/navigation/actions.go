package navigation

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

// ChangeFocusAction changes which pane has focus
type ChangeFocusAction struct {
	Pane Pane
}

// Execute implements Action for ChangeFocusAction
func (a *ChangeFocusAction) Execute() bool {
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

// NoOpAction represents no action (used for boundary cases)
type NoOpAction struct{}

// Execute implements Action for NoOpAction
func (a *NoOpAction) Execute() bool {
	return false
}
