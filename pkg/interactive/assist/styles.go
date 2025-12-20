package assist

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor = lipgloss.Color("#4AA8C7") // Apono cyan/teal brand color
	mutedColor   = lipgloss.Color("244")     // Gray
	errorColor   = lipgloss.Color("196")     // Red
	selectColor  = lipgloss.Color("6")       // Cyan - matches existing interactive mode
	borderColor  = lipgloss.Color("240")     // Border gray

	// Header styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Message styles
	userLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("114")) // Green - contrasts with assistant cyan

	assistantLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor)

	// Status styles
	loadingStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	// Slash command menu styles (matches existing interactive mode)
	slashItemStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	slashItemSelectedStyle = lipgloss.NewStyle().
				Foreground(selectColor)

	slashDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Conversation list styles (matches existing interactive mode)
	convItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	convItemSelectedStyle = lipgloss.NewStyle().
				Foreground(selectColor)

	convMetadataStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	convMetadataSelectedStyle = lipgloss.NewStyle().
					Foreground(selectColor)

	// Help styles
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			PaddingLeft(2)

	// Welcome styles
	welcomeStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Align(lipgloss.Center)

	suggestionChipStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	// CTA list selection styles (like Claude Code questions)
	ctaCursorStyle = lipgloss.NewStyle().
			Foreground(selectColor)

	ctaItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	ctaItemSelectedStyle = lipgloss.NewStyle().
				Foreground(selectColor).
				Bold(true)

	ctaDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	ctaWarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // Orange/yellow for warning

	// Resource card styles (for chat message display)
	resourceNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor)

	resourceLabelStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	resourceValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15"))

	// Request CTA box style (like Claude Code plan display)
	requestBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)
)
