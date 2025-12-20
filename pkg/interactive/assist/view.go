package assist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.exiting {
		return ""
	}

	var sections []string

	switch m.viewState {
	case ViewWelcome:
		sections = append(sections, m.renderWelcome())
	case ViewChat:
		if m.loading {
			sections = append(sections, loadingStyle.Render(m.spinner.View()+" "+m.loadingMessage))
			return lipgloss.JoinVertical(lipgloss.Left, sections...)
		}
	case ViewConversationList:
		sections = append(sections, m.renderConversationList())
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	case ViewSearchCTA:
		sections = append(sections, m.renderSearchCTA())
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	case ViewRequestCTA:
		sections = append(sections, m.renderRequestCTA())
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	case ViewEditJustification:
		sections = append(sections, m.renderEditJustification())
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	sections = append(sections, m.renderInput())

	if m.showSlashMenu {
		sections = append(sections, m.renderSlashMenu())
	}

	sections = append(sections, m.renderHelp())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderWelcome() string {
	var content strings.Builder

	content.WriteString(headerStyle.Render("Apono Assist"))
	content.WriteString("\n")
	if m.width > 0 {
		content.WriteString(strings.Repeat("─", m.width))
	}
	content.WriteString("\n\n")

	welcome := welcomeStyle.Width(m.width).Render("How can I help you today?")
	content.WriteString(welcome)
	content.WriteString("\n\n")

	suggestions := []string{
		"Hey, I need access",
		"What can I request access to?",
	}

	var chips []string
	for _, s := range suggestions {
		chips = append(chips, suggestionChipStyle.Render(s))
	}
	suggestionsLine := lipgloss.JoinHorizontal(lipgloss.Center, chips...)
	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(suggestionsLine)
	content.WriteString(centered)
	content.WriteString("\n")

	return content.String()
}

func (m Model) renderConversationList() string {
	var content strings.Builder

	if m.convSearchInput.Focused() || m.convSearchQuery != "" {
		content.WriteString(m.convSearchInput.View())
		content.WriteString("\n\n")
	}

	if m.loading {
		content.WriteString(loadingStyle.Render(m.spinner.View() + " " + m.loadingMessage))
		content.WriteString("\n")
		return content.String()
	}

	if len(m.convList.Items()) == 0 {
		content.WriteString(titleStyle.Render("Resume"))
		content.WriteString("\n\n")
		if m.convSearchQuery != "" {
			content.WriteString(subtitleStyle.Render("No conversations matching \"" + m.convSearchQuery + "\""))
		} else {
			content.WriteString(subtitleStyle.Render("No previous conversations found."))
		}
		content.WriteString("\n\n")
		content.WriteString(helpStyle.Render("Esc: back"))
		return content.String()
	}

	content.WriteString(m.convList.View())

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("↑/↓: navigate · /: search · Enter: select · Esc: cancel"))

	return content.String()
}

func (m Model) renderSlashMenu() string {
	if len(m.slashList.Items()) == 0 {
		return ""
	}
	return m.slashList.View()
}

func (m Model) renderInput() string {
	borderLine := lipgloss.NewStyle().
		Foreground(borderColor).
		Render(strings.Repeat("─", m.width))

	return "\n" + borderLine + "\n" + m.textarea.View() + "\n" + borderLine
}

func (m Model) renderHelp() string {
	if m.showSlashMenu {
		return helpStyle.Render("↑/↓: navigate · Tab/Enter: select · Esc: cancel")
	}

	var leftHint, rightHint string

	if m.ctrlCPendingExit {
		leftHint = "Press Ctrl-C again to exit"
	}

	if m.escPendingClear {
		rightHint = "Esc to clear again"
	}

	if leftHint == "" && rightHint == "" {
		return ""
	}

	leftStyled := lipgloss.NewStyle().
		Foreground(mutedColor).
		PaddingLeft(2).
		Render(leftHint)

	rightStyled := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(rightHint)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyled,
		lipgloss.NewStyle().Width(m.width-lipgloss.Width(leftStyled)-lipgloss.Width(rightStyled)-4).Render(""),
		rightStyled,
	)
}

func (m Model) renderSearchCTA() string {
	var content strings.Builder

	if len(m.ctaItems) == 0 {
		content.WriteString(subtitleStyle.Render("No options available"))
		content.WriteString("\n\n")
		content.WriteString(helpStyle.Render("Esc: back"))
		return content.String()
	}

	content.WriteString(subtitleStyle.Render("Select an option:"))
	content.WriteString("\n\n")

	for i, item := range m.ctaItems {
		isSelected := i == m.ctaCursor
		content.WriteString(m.renderSimpleCTAOption(i+1, item, isSelected))
	}

	otherIndex := len(m.ctaItems)
	isOtherSelected := m.ctaCursor == otherIndex
	content.WriteString(m.renderOtherOption(len(m.ctaItems)+1, isOtherSelected))

	content.WriteString(helpStyle.Render("↑/↓: navigate · Enter: select · Esc: cancel"))

	return content.String()
}

func (m Model) renderSimpleCTAOption(index int, item ctaItem, isSelected bool) string {
	label := item.Name
	if item.IntegrationName != "" {
		label = fmt.Sprintf("%s (%s)", item.Name, item.IntegrationName)
	}

	if isSelected {
		return ctaCursorStyle.Render(PromptChar) +
			ctaItemSelectedStyle.Render(fmt.Sprintf("%d. %s", index, label)) + "\n"
	}
	return fmt.Sprintf("  %d. ", index) + ctaItemStyle.Render(label) + "\n"
}

func (m Model) renderOtherOption(index int, isSelected bool) string {
	var content strings.Builder

	if isSelected {
		content.WriteString(ctaCursorStyle.Render(PromptChar))
		content.WriteString(ctaItemSelectedStyle.Render(fmt.Sprintf("%d. ", index)))
		content.WriteString(m.textarea.View())
	} else {
		content.WriteString(fmt.Sprintf("  %d. ", index))
		content.WriteString(ctaDescStyle.Render("Other..."))
	}
	content.WriteString("\n\n")

	return content.String()
}

func (m Model) renderRequestCTA() string {
	var content strings.Builder

	if m.activeRequestCTA == nil {
		content.WriteString(subtitleStyle.Render("No request available"))
		content.WriteString("\n\n")
		content.WriteString(helpStyle.Render("Esc: back"))
		return content.String()
	}

	content.WriteString(subtitleStyle.Render("Would you like to submit this request?"))
	content.WriteString("\n\n")

	options := []string{
		"Submit",
		"Submit & Wait",
		"Edit justification",
	}

	for i, label := range options {
		isSelected := i == m.requestButtonCursor
		if isSelected {
			content.WriteString(ctaCursorStyle.Render(PromptChar))
			content.WriteString(ctaItemSelectedStyle.Render(fmt.Sprintf("%d. %s", i+1, label)))
		} else {
			content.WriteString(fmt.Sprintf("  %d. ", i+1))
			content.WriteString(ctaItemStyle.Render(label))
		}
		content.WriteString("\n")
	}

	otherIndex := len(options)
	isOtherSelected := m.requestButtonCursor == otherIndex
	content.WriteString(m.renderOtherOption(otherIndex+1, isOtherSelected))

	content.WriteString(helpStyle.Render("↑/↓: navigate · Enter: select · e: edit justification · Esc: cancel"))

	return content.String()
}

func (m Model) renderEditJustification() string {
	var content strings.Builder

	content.WriteString(titleStyle.Render("Edit Justification"))
	content.WriteString("\n\n")
	content.WriteString(m.textarea.View())
	content.WriteString("\n\n")
	content.WriteString(helpStyle.Render("Enter: save · Esc: cancel"))

	return content.String()
}
