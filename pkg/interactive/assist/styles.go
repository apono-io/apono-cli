package assist

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Adaptive colors - automatically switch between light/dark terminal backgrounds
	primaryColor = lipgloss.AdaptiveColor{Light: "#007599", Dark: "#4AA8C7"} // Apono cyan/teal brand color
	mutedColor   = lipgloss.AdaptiveColor{Light: "241", Dark: "244"}         // Gray
	errorColor   = lipgloss.AdaptiveColor{Light: "160", Dark: "196"}         // Red
	selectColor  = lipgloss.AdaptiveColor{Light: "30", Dark: "6"}            // Cyan - matches existing interactive mode
	borderColor  = lipgloss.AdaptiveColor{Light: "250", Dark: "240"}         // Border gray
	textColor    = lipgloss.AdaptiveColor{Light: "232", Dark: "15"}          // Primary text
	userColor    = lipgloss.AdaptiveColor{Light: "22", Dark: "114"}          // Green for user label
	warningColor = lipgloss.AdaptiveColor{Light: "172", Dark: "214"}         // Orange/yellow for warnings

	// Header styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Message styles
	userLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(userColor) // Green - contrasts with assistant cyan

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
			Foreground(textColor)

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
			Foreground(textColor)

	ctaItemSelectedStyle = lipgloss.NewStyle().
				Foreground(selectColor).
				Bold(true)

	ctaDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	ctaWarningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	// Resource card styles (for chat message display)
	resourceNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor)

	resourceLabelStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	resourceValueStyle = lipgloss.NewStyle().
				Foreground(textColor)

	// Request CTA box style (like Claude Code plan display)
	requestBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)
)
