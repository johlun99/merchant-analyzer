// Package styles defines the Lipgloss color palette and shared styles for the TUI.
package styles

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary  = lipgloss.Color("#7C3AED") // violet
	colorAccent   = lipgloss.Color("#A78BFA") // light violet
	colorOK       = lipgloss.Color("#10B981") // emerald
	colorWarning  = lipgloss.Color("#F59E0B") // amber
	colorError    = lipgloss.Color("#EF4444") // red
	colorFatal    = lipgloss.Color("#7F1D1D") // dark red
	colorMuted    = lipgloss.Color("#6B7280") // gray
	colorBorder   = lipgloss.Color("#374151") // dark gray
	colorSelected = lipgloss.Color("#1F2937") // near-black bg for active tab

	// StyleTitle renders the app title.
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			PaddingLeft(1)

	// StyleSubtitle renders secondary headings.
	StyleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted).
			PaddingLeft(1)

	// StyleBorder applies a rounded border.
	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	// StyleTabActive renders the active tab label.
	StyleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Background(colorSelected).
			Padding(0, 2)

	// StyleTabInactive renders inactive tab labels.
	StyleTabInactive = lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 2)

	// StyleTabBar styles the tab bar container.
	StyleTabBar = lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			MarginBottom(1)

	// StyleStatusOK renders OK status text.
	StyleStatusOK = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorOK)

	// StyleStatusWarning renders warning status text.
	StyleStatusWarning = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWarning)

	// StyleStatusError renders error status text.
	StyleStatusError = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorError)

	// StyleStatusFatal renders fatal status text.
	StyleStatusFatal = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorFatal)

	// StyleMetric renders metric values.
	StyleMetric = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	// StyleMetricLabel renders metric labels.
	StyleMetricLabel = lipgloss.NewStyle().
				Foreground(colorMuted)

	// StyleCheckPending renders a pending checker name.
	StyleCheckPending = lipgloss.NewStyle().
				Foreground(colorMuted)

	// StyleCheckOK renders a passed checker name.
	StyleCheckOK = lipgloss.NewStyle().
			Foreground(colorOK)

	// StyleCheckError renders a failed checker name.
	StyleCheckError = lipgloss.NewStyle().
			Foreground(colorError)

	// StyleCheckWarning renders a warning checker name.
	StyleCheckWarning = lipgloss.NewStyle().
				Foreground(colorWarning)

	// StyleHelp renders the help/keybinding bar.
	StyleHelp = lipgloss.NewStyle().
			Foreground(colorMuted).
			PaddingTop(1)

	// StyleOverlay renders the export overlay background.
	StyleOverlay = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2).
			Background(lipgloss.Color("#111827"))

	// StyleSuccess renders a success message.
	StyleSuccess = lipgloss.NewStyle().
			Foreground(colorOK).
			Bold(true)
)
