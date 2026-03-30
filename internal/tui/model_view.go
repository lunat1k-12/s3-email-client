package tui

// View implements the Bubble Tea View interface
func (m *Model) View() string {
	if m.showDeleteModal {
		return m.renderDeleteModal()
	}

	if m.linkPickerMode {
		return m.renderLinkPickerModal()
	}

	return m.renderNormalView()
}
