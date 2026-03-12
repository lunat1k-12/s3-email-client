package navigation

import "s3emailclient/internal/parser"

// NavigationHandler processes keyboard input and returns appropriate actions
type NavigationHandler interface {
	// HandleKey processes a keyboard event and returns the appropriate action
	HandleKey(key string, state *State) Action
}

// State represents the current navigation state needed for decision making
type State struct {
	FocusedPane       Pane
	SelectedIndex     int
	EmailCount        int
	ContentScroll     int
	MaxScroll         int
	CurrentEmail      *parser.Email // Current selected email for response actions
	CurrentEmailKey   string        // Key of current email in S3
	ComposeMode       bool          // Whether compose view is active
	DeleteModalActive bool          // Whether delete confirmation modal is shown
}

// Pane represents which pane currently has focus
type Pane int

const (
	ListPane Pane = iota
	ContentPane
)

// DefaultNavigationHandler implements NavigationHandler with vim-like keybindings
type DefaultNavigationHandler struct{}

// NewNavigationHandler creates a new DefaultNavigationHandler
func NewNavigationHandler() NavigationHandler {
	return &DefaultNavigationHandler{}
}

// HandleKey maps keyboard events to actions with boundary validation
func (h *DefaultNavigationHandler) HandleKey(key string, state *State) Action {
	// Handle delete modal keys first (highest priority)
	if state.DeleteModalActive {
		switch key {
		case "y", "Y":
			return &ConfirmDeleteAction{}
		case "n", "N", "esc":
			return &CancelDeleteAction{}
		default:
			// Modal is active, ignore all other keys
			return &NoOpAction{}
		}
	}

	// Disable navigation keys in compose mode
	if state.ComposeMode {
		switch key {
		case "j", "k", "J", "K":
			return &NoOpAction{}
		}
	}

	switch key {
	case "j":
		// Move selection down in email list
		if state.SelectedIndex < state.EmailCount-1 {
			return &MoveSelectionAction{Direction: 1}
		}
		// At bottom, stay at bottom (boundary validation)
		return &NoOpAction{}

	case "k":
		// Move selection up in email list
		if state.SelectedIndex > 0 {
			return &MoveSelectionAction{Direction: -1}
		}
		// At top, stay at top (boundary validation)
		return &NoOpAction{}

	case "J":
		// Scroll email content down
		return &ScrollContentAction{Lines: 1}

	case "K":
		// Scroll email content up
		return &ScrollContentAction{Lines: -1}

	case "d":
		// Initiate email deletion
		if state.CurrentEmail != nil && state.SelectedIndex >= 0 {
			return &DeleteAction{
				Key:     state.CurrentEmailKey,
				Subject: state.CurrentEmail.Subject,
			}
		}
		// No email selected, no action
		return &NoOpAction{}

	case "r":
		// Initiate email response
		if state.CurrentEmail != nil {
			return &ResponseAction{Email: state.CurrentEmail}
		}
		// No email selected, no action
		return &NoOpAction{}

	case "q":
		// Quit application
		return &QuitAction{}

	default:
		// Unknown key, no action
		return &NoOpAction{}
	}
}
