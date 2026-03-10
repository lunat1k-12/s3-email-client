package navigation

import "testing"

func TestHandleKey_MoveDown(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{
		FocusedPane:   ListPane,
		SelectedIndex: 0,
		EmailCount:    5,
	}

	action := handler.HandleKey("j", state)
	if _, ok := action.(*MoveSelectionAction); !ok {
		t.Errorf("Expected MoveSelectionAction, got %T", action)
	}

	moveAction := action.(*MoveSelectionAction)
	if moveAction.Direction != 1 {
		t.Errorf("Expected Direction 1, got %d", moveAction.Direction)
	}
}

func TestHandleKey_MoveUp(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{
		FocusedPane:   ListPane,
		SelectedIndex: 2,
		EmailCount:    5,
	}

	action := handler.HandleKey("k", state)
	if _, ok := action.(*MoveSelectionAction); !ok {
		t.Errorf("Expected MoveSelectionAction, got %T", action)
	}

	moveAction := action.(*MoveSelectionAction)
	if moveAction.Direction != -1 {
		t.Errorf("Expected Direction -1, got %d", moveAction.Direction)
	}
}

func TestHandleKey_BoundaryTop(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{
		FocusedPane:   ListPane,
		SelectedIndex: 0,
		EmailCount:    5,
	}

	action := handler.HandleKey("k", state)
	if _, ok := action.(*NoOpAction); !ok {
		t.Errorf("Expected NoOpAction at top boundary, got %T", action)
	}
}

func TestHandleKey_BoundaryBottom(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{
		FocusedPane:   ListPane,
		SelectedIndex: 4,
		EmailCount:    5,
	}

	action := handler.HandleKey("j", state)
	if _, ok := action.(*NoOpAction); !ok {
		t.Errorf("Expected NoOpAction at bottom boundary, got %T", action)
	}
}

func TestHandleKey_ChangeFocusLeft(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{
		FocusedPane: ContentPane,
	}

	action := handler.HandleKey("h", state)
	if _, ok := action.(*ChangeFocusAction); !ok {
		t.Errorf("Expected ChangeFocusAction, got %T", action)
	}

	focusAction := action.(*ChangeFocusAction)
	if focusAction.Pane != ListPane {
		t.Errorf("Expected ListPane, got %v", focusAction.Pane)
	}
}

func TestHandleKey_ChangeFocusRight(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{
		FocusedPane: ListPane,
	}

	action := handler.HandleKey("l", state)
	if _, ok := action.(*ChangeFocusAction); !ok {
		t.Errorf("Expected ChangeFocusAction, got %T", action)
	}

	focusAction := action.(*ChangeFocusAction)
	if focusAction.Pane != ContentPane {
		t.Errorf("Expected ContentPane, got %v", focusAction.Pane)
	}
}

func TestHandleKey_ScrollContentDown(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{
		FocusedPane: ContentPane,
	}

	action := handler.HandleKey("j", state)
	if _, ok := action.(*ScrollContentAction); !ok {
		t.Errorf("Expected ScrollContentAction, got %T", action)
	}

	scrollAction := action.(*ScrollContentAction)
	if scrollAction.Lines != 1 {
		t.Errorf("Expected Lines 1, got %d", scrollAction.Lines)
	}
}

func TestHandleKey_ScrollContentUp(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{
		FocusedPane: ContentPane,
	}

	action := handler.HandleKey("k", state)
	if _, ok := action.(*ScrollContentAction); !ok {
		t.Errorf("Expected ScrollContentAction, got %T", action)
	}

	scrollAction := action.(*ScrollContentAction)
	if scrollAction.Lines != -1 {
		t.Errorf("Expected Lines -1, got %d", scrollAction.Lines)
	}
}

func TestHandleKey_Quit(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{}

	action := handler.HandleKey("q", state)
	if _, ok := action.(*QuitAction); !ok {
		t.Errorf("Expected QuitAction, got %T", action)
	}
}

func TestHandleKey_UnknownKey(t *testing.T) {
	handler := NewNavigationHandler()
	state := &State{}

	action := handler.HandleKey("x", state)
	if _, ok := action.(*NoOpAction); !ok {
		t.Errorf("Expected NoOpAction for unknown key, got %T", action)
	}
}
