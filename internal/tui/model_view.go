package tui

// View implements the Bubble Tea View interface
// It conditionally renders either the delete confirmation modal or the normal view
// based on the showDeleteModal flag
func (m *Model) View() string {
	if m.showDeleteModal {
		return m.renderDeleteModal()
	}
	
	return m.renderNormalView()
}
