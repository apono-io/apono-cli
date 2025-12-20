package assist

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
)

const (
	keyEsc   = "esc"
	keyEnter = "enter"
	keyTab   = "tab"
	keyUp    = "up"
	keyDown  = "down"
)

const hintDisplayTimeout = 1300 * time.Millisecond

type escHintTimeoutMsg struct{}

func escHintTimeoutCmd() tea.Cmd {
	return tea.Tick(hintDisplayTimeout, func(t time.Time) tea.Msg {
		return escHintTimeoutMsg{}
	})
}

type ctrlCHintTimeoutMsg struct{}

func ctrlCHintTimeoutCmd() tea.Cmd {
	return tea.Tick(hintDisplayTimeout, func(t time.Time) tea.Msg {
		return ctrlCHintTimeoutMsg{}
	})
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case assistantResponseMsg:
		return m.handleAssistantResponse(msg)

	case conversationListMsg:
		return m.handleConversationList(msg)

	case conversationHistoryMsg:
		return m.handleConversationHistory(msg)

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case accessRequestResultMsg:
		return m.handleAccessRequestResult(msg)

	case escHintTimeoutMsg:
		m.escPendingClear = false
		return m, nil

	case ctrlCHintTimeoutMsg:
		m.ctrlCPendingExit = false
		return m, nil
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	m.updateSlashMenu()

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keys.Quit) {
		if m.ctrlCPendingExit {
			m.exiting = true
			return m, tea.Quit
		}

		switch m.viewState {
		case ViewSearchCTA:
			m.restoreTextareaForChat()
			m.activeSearchCTA = nil
			m.ctaItems = nil
			m.ctaCursor = 0
			m.viewState = ViewChat
		case ViewRequestCTA:
			m.restoreTextareaForChat()
			m.activeRequestCTA = nil
			m.editedJustification = ""
			m.requestButtonCursor = 0
			m.viewState = ViewChat
		case ViewConversationList:
			m.convSearchInput.Blur()
			m.convSearchInput.SetValue("")
			m.convSearchQuery = ""
			if len(m.messages) == 0 {
				m.viewState = ViewWelcome
			} else {
				m.viewState = ViewChat
			}
		case ViewEditJustification:
			m.textarea.SetValue("")
			m.activeRequestCTA = nil
			m.editedJustification = ""
			m.viewState = ViewChat
		}

		m.textarea.SetValue("")
		m.updateTextareaHeight()
		m.ctrlCPendingExit = true
		return m, ctrlCHintTimeoutCmd()
	}

	switch m.viewState {
	case ViewConversationList:
		return m.handleConversationListKeys(msg)
	case ViewSearchCTA:
		return m.handleSearchCTAKeys(msg)
	case ViewRequestCTA:
		return m.handleRequestCTAKeys(msg)
	case ViewEditJustification:
		return m.handleEditJustificationKeys(msg)
	default:
		return m.handleChatKeys(msg)
	}
}

func (m Model) handleChatKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showSlashMenu {
		switch msg.String() {
		case keyTab, keyEnter:
			if item, ok := m.slashList.SelectedItem().(slashItem); ok {
				return m.executeSlashCommand(item.cmd.Name)
			}
			return m, nil
		case keyEsc:
			m.showSlashMenu = false
			m.textarea.SetValue("")
			return m, nil
		case keyUp, keyDown:
			var cmd tea.Cmd
			m.slashList, cmd = m.slashList.Update(msg)
			return m, cmd
		}
	}

	if msg.String() == keyEsc {
		if strings.TrimSpace(m.textarea.Value()) == "" {
			return m, nil
		}
		if m.escPendingClear {
			m.textarea.SetValue("")
			m.escPendingClear = false
			m.updateTextareaHeight()
			return m, nil
		}
		m.escPendingClear = true
		return m, escHintTimeoutCmd()
	}

	if msg.Type == tea.KeyEnter && !m.loading && !msg.Paste {
		input := strings.TrimSpace(m.textarea.Value())
		if input == "" {
			return m, nil
		}
		if strings.HasPrefix(input, "/") {
			return m.executeSlashCommand(input)
		}
		return m.sendMessage(input)
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	m.updateSlashMenu()
	m.updateTextareaHeight()

	return m, cmd
}

func (m Model) handleConversationListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.convSearchInput.Focused() {
		switch msg.String() {
		case keyEsc:
			m.convSearchInput.Blur()
			m.convSearchInput.SetValue("")
			m.convSearchQuery = ""
			m.filterConversations()
			return m, nil
		case keyEnter:
			m.convSearchInput.Blur()
			conv := getSelectedConversation(m.convList)
			if conv != nil {
				m.conversationID = conv.Id
				m.viewState = ViewChat
				m.loading = true
				m.loadingMessage = LoadingMsgHistory
				m.convSearchQuery = ""
				m.convSearchInput.SetValue("")
				return m, m.loadConversationHistory(conv.Id)
			}
			return m, nil
		case keyUp, keyDown:
			var cmd tea.Cmd
			m.convList, cmd = m.convList.Update(msg)
			return m, cmd
		default:
			var cmd tea.Cmd
			m.convSearchInput, cmd = m.convSearchInput.Update(msg)
			newQuery := m.convSearchInput.Value()
			if newQuery != m.convSearchQuery {
				m.convSearchQuery = newQuery
				m.filterConversations()
			}
			return m, cmd
		}
	}

	switch msg.String() {
	case "/":
		m.convSearchInput.Focus()
		return m, textinput.Blink
	case keyEnter:
		conv := getSelectedConversation(m.convList)
		if conv != nil {
			m.conversationID = conv.Id
			m.viewState = ViewChat
			m.loading = true
			m.loadingMessage = LoadingMsgHistory
			return m, m.loadConversationHistory(conv.Id)
		}
		return m, nil
	case keyEsc:
		m.viewState = ViewChat
		if len(m.messages) == 0 {
			m.viewState = ViewWelcome
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.convList, cmd = m.convList.Update(msg)
		return m, cmd
	}
}

func (m Model) handleSearchCTAKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	totalOptions := len(m.ctaItems) + 1
	otherIndex := len(m.ctaItems)
	isOnOther := m.ctaCursor == otherIndex

	if isOnOther {
		switch msg.String() {
		case keyUp:
			if m.ctaCursor > 0 {
				m.textarea.SetValue("")
				m.restoreTextareaForChat()
				m.ctaCursor--
			}
			return m, nil
		case keyEnter:
			customText := strings.TrimSpace(m.textarea.Value())
			if customText != "" {
				return m.submitCustomCTAResponse(customText)
			}
			return m, nil
		case keyEsc:
			if m.textarea.Value() != "" {
				m.textarea.SetValue("")
				return m, nil
			}
			m.restoreTextareaForChat()
			m.activeSearchCTA = nil
			m.ctaItems = nil
			m.ctaCursor = 0
			m.viewState = ViewChat
			return m, nil
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
	}

	switch msg.String() {
	case keyUp:
		if m.ctaCursor > 0 {
			m.ctaCursor--
		}
		return m, nil
	case keyDown:
		if m.ctaCursor < totalOptions-1 {
			m.ctaCursor++
			if m.ctaCursor == otherIndex {
				m.configureTextareaForOther()
			}
		}
		return m, nil
	case keyEnter:
		if m.ctaCursor < len(m.ctaItems) {
			selected := m.ctaItems[m.ctaCursor]
			return m.selectCTAItem(selected)
		}
		return m, nil
	case keyEsc:
		m.restoreTextareaForChat()
		m.activeSearchCTA = nil
		m.ctaItems = nil
		m.ctaCursor = 0
		m.viewState = ViewChat
		return m, nil
	}
	return m, nil
}

func (m Model) submitCustomCTAResponse(text string) (tea.Model, tea.Cmd) {
	m.restoreTextareaForChat()
	m.activeSearchCTA = nil
	m.ctaItems = nil
	m.ctaCursor = 0
	m.textarea.SetValue("")

	m.AddUserMessage(text)
	userMsg := m.messages[len(m.messages)-1]
	printCmd := tea.Println(FormatMessage(userMsg) + "\n")

	m.viewState = ViewChat
	m.loading = true
	m.loadingMessage = LoadingMsgThinking

	return m, tea.Batch(printCmd, m.sendToAssistant(text))
}

const (
	requestCTASubmit        = 0
	requestCTASubmitAndWait = 1
	requestCTAEditJustify   = 2
	requestCTAOtherIndex    = 3
	requestCTATotalOptions  = 4
)

func (m Model) handleRequestCTAKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	isOnOther := m.requestButtonCursor == requestCTAOtherIndex

	if isOnOther {
		switch msg.String() {
		case keyUp:
			if m.requestButtonCursor > 0 {
				m.textarea.SetValue("")
				m.restoreTextareaForChat()
				m.requestButtonCursor--
			}
			return m, nil
		case keyEnter:
			customText := strings.TrimSpace(m.textarea.Value())
			if customText != "" {
				return m.submitRequestCTACustomResponse(customText)
			}
			return m, nil
		case keyEsc:
			if m.textarea.Value() != "" {
				m.textarea.SetValue("")
				return m, nil
			}
			m.restoreTextareaForChat()
			m.activeRequestCTA = nil
			m.editedJustification = ""
			m.viewState = ViewChat
			return m, nil
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
	}

	switch msg.String() {
	case keyUp:
		if m.requestButtonCursor > 0 {
			m.requestButtonCursor--
		}
		return m, nil
	case keyDown:
		if m.requestButtonCursor < requestCTATotalOptions-1 {
			m.requestButtonCursor++
			if m.requestButtonCursor == requestCTAOtherIndex {
				m.configureTextareaForOther()
			}
		}
		return m, nil
	case "e":
		return m.enterEditJustification()
	case keyEnter:
		switch m.requestButtonCursor {
		case requestCTASubmit:
			return m.submitAccessRequest(false)
		case requestCTASubmitAndWait:
			return m.submitAccessRequest(true)
		case requestCTAEditJustify:
			return m.enterEditJustification()
		}
		return m, nil
	case keyEsc:
		m.activeRequestCTA = nil
		m.editedJustification = ""
		m.viewState = ViewChat
		return m, nil
	}
	return m, nil
}

func (m Model) submitRequestCTACustomResponse(text string) (tea.Model, tea.Cmd) {
	m.restoreTextareaForChat()
	m.activeRequestCTA = nil
	m.editedJustification = ""
	m.requestButtonCursor = 0
	m.textarea.SetValue("")

	m.AddUserMessage(text)
	userMsg := m.messages[len(m.messages)-1]
	printCmd := tea.Println(FormatMessage(userMsg) + "\n")

	m.viewState = ViewChat
	m.loading = true
	m.loadingMessage = LoadingMsgThinking

	return m, tea.Batch(printCmd, m.sendToAssistant(text))
}

func (m Model) handleEditJustificationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEnter:
		m.editedJustification = strings.TrimSpace(m.textarea.Value())
		m.textarea.SetValue("")
		m.viewState = ViewRequestCTA
		return m, nil
	case keyEsc:
		m.textarea.SetValue("")
		m.viewState = ViewRequestCTA
		return m, nil
	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
}

func (m Model) selectCTAItem(item ctaItem) (tea.Model, tea.Cmd) {
	m.restoreTextareaForChat()
	m.activeSearchCTA = nil
	m.ctaItems = nil
	m.ctaCursor = 0

	var selectionMessage string
	if item.Type == CTAItemTypeResource {
		selectionMessage = fmt.Sprintf("Request resource with name %s, path %s and type %s",
			item.Name, item.Path, item.ResourceType)
	} else {
		selectionMessage = "Request bundle with name " + item.Name
	}

	m.AddUserMessage(selectionMessage)
	userMsg := m.messages[len(m.messages)-1]
	printCmd := tea.Println(FormatMessage(userMsg) + "\n")

	m.viewState = ViewChat
	m.loading = true
	m.loadingMessage = LoadingMsgThinking

	return m, tea.Batch(printCmd, m.sendToAssistant(selectionMessage))
}

func (m Model) enterEditJustification() (tea.Model, tea.Cmd) {
	currentJustification := m.editedJustification
	if currentJustification == "" && m.activeRequestCTA != nil {
		if m.activeRequestCTA.HasResourcesRequest() {
			currentJustification = m.activeRequestCTA.GetResourcesRequest().Justification
		} else if m.activeRequestCTA.HasBundlesRequest() {
			currentJustification = m.activeRequestCTA.GetBundlesRequest().Justification
		}
	}
	m.textarea.SetValue(currentJustification)
	m.viewState = ViewEditJustification
	return m, nil
}

func (m Model) submitAccessRequest(waitForStatus bool) (tea.Model, tea.Cmd) {
	if m.activeRequestCTA == nil {
		return m, nil
	}

	req := m.buildAccessRequestFromCTA()
	if req == nil {
		m.err = fmt.Errorf("failed to build access request")
		m.viewState = ViewChat
		return m, tea.Println(errorStyle.Render("Error: Failed to build access request"))
	}

	m.activeRequestCTA = nil
	m.editedJustification = ""
	m.viewState = ViewChat
	m.loading = true
	m.loadingMessage = LoadingMsgSubmitting

	return m, m.doSubmitAccessRequest(req, waitForStatus)
}

func (m Model) buildAccessRequestFromCTA() *clientapi.CreateAccessRequestClientModel {
	if m.activeRequestCTA == nil {
		return nil
	}

	req := services.GetEmptyNewRequestAPIModel()
	var justification string

	if m.activeRequestCTA.HasResourcesRequest() {
		resReq := m.activeRequestCTA.GetResourcesRequest()
		var accessUnitIDs []string
		for _, e := range resReq.GetEntitlements() {
			accessUnitIDs = append(accessUnitIDs, e.Id)
		}
		req.FilterAccessUnitIds = accessUnitIDs
		justification = resReq.Justification
	}

	if m.activeRequestCTA.HasBundlesRequest() {
		bundleReq := m.activeRequestCTA.GetBundlesRequest()
		var bundleIDs []string
		for _, e := range bundleReq.GetEntitlements() {
			bundleIDs = append(bundleIDs, e.Id)
		}
		req.FilterBundleIds = bundleIDs
		if justification == "" {
			justification = bundleReq.Justification
		}
	}

	if m.editedJustification != "" {
		justification = m.editedJustification
	}

	if justification != "" {
		req.SetJustification(justification)
	}

	return req
}

func (m Model) doSubmitAccessRequest(req *clientapi.CreateAccessRequestClientModel, waitForStatus bool) tea.Cmd {
	return func() tea.Msg {
		requestID, err := services.CreateAccessRequest(m.ctx, m.client, req)
		if err != nil {
			return accessRequestResultMsg{Err: err}
		}

		if !waitForStatus {
			return accessRequestResultMsg{
				RequestID:     requestID,
				WaitForStatus: false,
			}
		}

		accessRequest, err := services.PollAccessRequestStatus(
			m.ctx,
			m.client,
			requestID,
			services.AccessRequestPollingTimeout,
			services.AccessRequestPollingInterval,
		)
		if err != nil {
			return accessRequestResultMsg{
				RequestID:     requestID,
				WaitForStatus: true,
				Err:           err,
			}
		}

		return accessRequestResultMsg{
			RequestID:     requestID,
			Request:       accessRequest,
			WaitForStatus: true,
		}
	}
}

func (m *Model) updateSlashMenu() {
	input := m.textarea.Value()
	if strings.HasPrefix(input, "/") && !strings.Contains(input, " ") {
		matches := m.GetMatchingSlashCommands(input)
		m.showSlashMenu = len(matches) > 0
		if m.showSlashMenu {
			items := slashCommandsToItems(matches)
			m.slashList.SetItems(items)
		}
	} else {
		m.showSlashMenu = false
	}
}

func (m *Model) updateTextareaHeight() {
	content := m.textarea.Value()
	if content == "" {
		m.textarea.SetHeight(1)
		return
	}

	textWidth := m.textarea.Width()
	if textWidth < 10 {
		textWidth = 10
	}

	wrapped := wordwrap.String(content, textWidth)
	visualLines := strings.Count(wrapped, "\n") + 1

	if visualLines < 1 {
		visualLines = 1
	}

	m.textarea.SetHeight(visualLines)
}

func (m Model) executeSlashCommand(cmd string) (tea.Model, tea.Cmd) {
	m.showSlashMenu = false
	m.textarea.SetValue("")

	switch cmd {
	case SlashCmdNew:
		m.StartNewConversation()
		return m, tea.ClearScreen
	case SlashCmdResume:
		m.viewState = ViewConversationList
		m.loading = true
		m.loadingMessage = LoadingMsgConversations
		return m, m.loadConversations()
	case SlashCmdQuit:
		m.exiting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) sendMessage(content string) (tea.Model, tea.Cmd) {
	m.textarea.SetValue("")
	m.AddUserMessage(content)
	m.viewState = ViewChat
	m.loading = true
	m.loadingMessage = LoadingMsgThinking
	m.err = nil

	userMsg := m.messages[len(m.messages)-1]
	printCmd := tea.Println(FormatMessage(userMsg) + "\n")

	return m, tea.Batch(printCmd, m.sendToAssistant(content))
}

func (m Model) sendToAssistant(content string) tea.Cmd {
	return func() tea.Msg {
		req := clientapi.NewAssistantMessageRequestModel(MessageTypeUserMessage, content, m.conversationID)

		resp, err := services.SendAssistantMessage(m.ctx, m.client, req)
		if err != nil {
			return assistantResponseMsg{Err: err}
		}

		return assistantResponseMsg{Response: resp}
	}
}

func (m Model) loadConversations() tea.Cmd {
	return func() tea.Msg {
		conversations, err := services.ListAssistantConversations(m.ctx, m.client)
		if err != nil {
			return conversationListMsg{Err: err}
		}

		return conversationListMsg{Conversations: conversations}
	}
}

func (m Model) loadConversationHistory(convID string) tea.Cmd {
	return func() tea.Msg {
		messages, err := services.GetAssistantConversationHistory(m.ctx, m.client, convID)
		if err != nil {
			return conversationHistoryMsg{Err: err}
		}

		return conversationHistoryMsg{
			ConversationID: convID,
			Messages:       messages,
		}
	}
}

func (m Model) handleAssistantResponse(msg assistantResponseMsg) (tea.Model, tea.Cmd) {
	m.loading = false

	if msg.Err != nil {
		m.err = msg.Err
		return m, tea.Println(errorStyle.Render("Error: " + msg.Err.Error()))
	}

	if msg.Response.ConversationId != "" {
		m.conversationID = msg.Response.ConversationId
	}

	m.AddAssistantMessage(msg.Response.Message)
	m.suggestions = msg.Response.Suggestions

	var searchCTA *clientapi.AssistantMessageDataClientModelClientSearchCta
	var requestCTA *clientapi.AssistantMessageDataClientModelClientRequestCta

	for _, d := range msg.Response.Message.Data {
		if d.HasClientSearchCta() {
			cta := d.GetClientSearchCta()
			searchCTA = &cta
		}
		if d.HasClientRequestCta() {
			cta := d.GetClientRequestCta()
			requestCTA = &cta
		}
	}

	assistantMsg := m.messages[len(m.messages)-1]
	output := FormatMessage(assistantMsg)

	if searchCTA == nil && requestCTA == nil {
		if suggestionsStr := FormatSuggestions(m.suggestions); suggestionsStr != "" {
			output += "\n" + suggestionsStr
		}
	}

	printCmd := tea.Println(output + "\n")

	if searchCTA != nil {
		m.activeSearchCTA = searchCTA
		items, meta := buildCTAItems(searchCTA)
		m.ctaItems = items
		m.ctaResourcesTotal = meta.resourcesTotal
		m.ctaResourcesHasMore = meta.resourcesHasMore
		m.ctaBundlesTotal = meta.bundlesTotal
		m.ctaBundlesHasMore = meta.bundlesHasMore
		m.ctaCursor = 0
		m.viewState = ViewSearchCTA
		return m, printCmd
	}

	if requestCTA != nil {
		m.activeRequestCTA = requestCTA
		m.requestButtonCursor = 0
		m.editedJustification = ""
		m.viewState = ViewRequestCTA
		return m, printCmd
	}

	return m, printCmd
}

type ctaMetadata struct {
	resourcesTotal   int32
	resourcesHasMore bool
	bundlesTotal     int32
	bundlesHasMore   bool
}

func buildCTAItems(cta *clientapi.AssistantMessageDataClientModelClientSearchCta) ([]ctaItem, ctaMetadata) {
	var items []ctaItem
	var meta ctaMetadata

	if cta.HasResources() {
		resources := cta.GetResources()
		meta.resourcesTotal = resources.Total
		meta.resourcesHasMore = resources.HasMore

		for _, r := range resources.GetData() {
			items = append(items, ctaItem{
				Type:             CTAItemTypeResource,
				ID:               r.Id,
				Name:             r.Name,
				Path:             r.SourceId,
				ResourceType:     r.Type.Name,
				ResourceTypePath: r.Type.DisplayPath,
				IntegrationName:  r.Integration.Name,
				IntegrationType:  r.Integration.TypeDisplayName,
			})
		}
	}

	if cta.HasBundles() {
		bundles := cta.GetBundles()
		meta.bundlesTotal = bundles.Total
		meta.bundlesHasMore = bundles.HasMore

		for _, b := range bundles.GetData() {
			items = append(items, ctaItem{
				Type:         CTAItemTypeBundle,
				ID:           b.Id,
				Name:         b.Name,
				ResourceType: ResourceTypeBundleDisplayName,
			})
		}
	}

	return items, meta
}

func (m Model) handleConversationList(msg conversationListMsg) (tea.Model, tea.Cmd) {
	m.loading = false

	if msg.Err != nil {
		m.err = msg.Err
		m.viewState = ViewWelcome
		return m, nil
	}

	m.allConversations = msg.Conversations
	items := conversationsToItems(msg.Conversations)
	cmd := m.convList.SetItems(items)
	return m, cmd
}

func (m *Model) filterConversations() {
	query := strings.ToLower(m.convSearchQuery)

	if query == "" {
		items := conversationsToItems(m.allConversations)
		m.convList.SetItems(items)
		return
	}

	var filtered []clientapi.AssistantConversationClientModel
	for _, conv := range m.allConversations {
		if strings.Contains(strings.ToLower(conv.Title), query) {
			filtered = append(filtered, conv)
		}
	}

	items := conversationsToItems(filtered)
	m.convList.SetItems(items)
}

func (m Model) handleConversationHistory(msg conversationHistoryMsg) (tea.Model, tea.Cmd) {
	m.loading = false

	if msg.Err != nil {
		m.err = msg.Err
		return m, tea.Println(errorStyle.Render("Error loading conversation: " + msg.Err.Error()))
	}

	m.messages = nil
	m.conversationID = msg.ConversationID

	messages := msg.Messages
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].MessageOrdinal < messages[j].MessageOrdinal
	})

	for _, apiMsg := range messages {
		m.AddMessageFromHistory(apiMsg)
	}

	m.viewState = ViewChat

	var output strings.Builder
	for i, chatMsg := range m.messages {
		if i > 0 {
			output.WriteString("\n\n")
		}
		output.WriteString(FormatMessage(chatMsg))
	}

	return m, tea.Sequence(tea.ClearScreen, tea.Println(output.String()))
}

func (m Model) handleAccessRequestResult(msg accessRequestResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false

	if msg.Err != nil {
		m.err = msg.Err
		return m, tea.Println(errorStyle.Render("Error: " + msg.Err.Error()))
	}

	var output strings.Builder
	output.WriteString(assistantLabelStyle.Render("Access Request"))
	output.WriteString("\n")

	if msg.Request != nil {
		output.WriteString(fmt.Sprintf("  Request %s is %s\n", msg.RequestID, services.ColoredStatus(*msg.Request)))
	} else {
		output.WriteString(fmt.Sprintf("  Request %s submitted successfully\n", msg.RequestID))
	}

	return m, tea.Println(output.String())
}

func (m *Model) configureTextareaForOther() {
	m.textarea.Placeholder = "Type something else..."
	m.textarea.SetPromptFunc(0, func(lineIdx int) string {
		return ""
	})
}

func (m *Model) restoreTextareaForChat() {
	m.textarea.Placeholder = DefaultPlaceholder
	m.textarea.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return PromptChar
		}
		return PromptContinue
	})
}
