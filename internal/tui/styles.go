package tui

import "github.com/charmbracelet/lipgloss"

var (
	// normalItemStyle is the style for unselected email list items
	normalItemStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("252"))

	// selectedItemStyle is the style for the selected email list item
	selectedItemStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Bold(true)

	// emptyListStyle is the style for the empty list message
	emptyListStyle = lipgloss.NewStyle().
			Padding(2, 4).
			Foreground(lipgloss.Color("241")).
			Italic(true)

	// listHeaderStyle is the style for the list header showing email count
	listHeaderStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("141")).
			Bold(true)

	// Content pane styles
	headerLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Bold(true)

	headerValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	subjectStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Bold(true).
			MarginBottom(1)

	bodyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			MarginTop(1)

	attachmentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")).
			Italic(true)

	emptyContentStyle = lipgloss.NewStyle().
				Padding(2, 4).
				Foreground(lipgloss.Color("241")).
				Italic(true)

	// Status bar styles
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
			Bold(true)

	// Separator style
	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)
