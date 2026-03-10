package navigation

// NavigationHandler processes keyboard input and returns appropriate actions
type NavigationHandler interface {
	// HandleKey processes a keyboard event and returns the appropriate action
	HandleKey(key string, state *State) Action
}

// State represents the current navigation state needed for decision making
type State struct {
	FocusedPane   Pane
	SelectedIndex int
	EmailCount    int
	ContentScroll int
	MaxScroll     int
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
	switch key {
	case "j":
		if state.FocusedPane == ListPane {
			// Move selection down in email list
			if state.SelectedIndex < state.EmailCount-1 {
				return &MoveSelectionAction{Direction: 1}
			}
			// At bottom, stay at bottom (boundary validation)
			return &NoOpAction{}
		}
		// Content pane: scroll down
		return &ScrollContentAction{Lines: 1}

	case "k":
		if state.FocusedPane == ListPane {
			// Move selection up in email list
			if state.SelectedIndex > 0 {
				return &MoveSelectionAction{Direction: -1}
			}
			// At top, stay at top (boundary validation)
			return &NoOpAction{}
		}
		// Content pane: scroll up
		return &ScrollContentAction{Lines: -1}

	case "h":
		// Move focus to list pane
		return &ChangeFocusAction{Pane: ListPane}

	case "l":
		// Move focus to content pane
		return &ChangeFocusAction{Pane: ContentPane}

	case "q":
		// Quit application
		return &QuitAction{}

	default:
		// Unknown key, no action
		return &NoOpAction{}
	}
}
