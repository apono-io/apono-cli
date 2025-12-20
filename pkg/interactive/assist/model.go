package assist

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type ViewState int

const (
	ViewWelcome ViewState = iota
	ViewChat
	ViewConversationList
	ViewSearchCTA
	ViewRequestCTA
	ViewEditJustification
)

const (
	RoleUser               = "user"
	RoleUserMessage        = "user_message"
	RoleAssistantMessage   = "assistant_message"
	MessageTypeUserMessage = "user_message"
)

const (
	SlashCmdNew    = "/new"
	SlashCmdResume = "/resume"
	SlashCmdQuit   = "/quit"
)

const (
	CTAItemTypeResource = "resource"
	CTAItemTypeBundle   = "bundle"
)

const (
	LoadingMsgThinking      = "Thinking..."
	LoadingMsgConversations = "Loading conversations..."
	LoadingMsgHistory       = "Loading conversation..."
	LoadingMsgSubmitting    = "Submitting request..."
)

const (
	DefaultPlaceholder            = "Type a message or use / for commands..."
	ResourceTypeBundleDisplayName = "Bundle"
)

const (
	DefaultTextareaWidth  = 80
	DefaultTextareaHeight = 1
	MaxTextareaHeight     = 20
	DefaultListHeight     = 10
)

const (
	PromptChar     = "> "
	PromptContinue = "  "
)

type ChatMessage struct {
	ID          string
	Role        string
	Content     string
	RawData     []clientapi.AssistantMessageDataClientModel
	CreatedDate time.Time
}

type Model struct {
	ctx    context.Context
	client *aponoapi.AponoClient

	width     int
	height    int
	viewState ViewState

	conversationID string
	messages       []ChatMessage
	suggestions    []clientapi.AssistantConversationSuggestionClientModel

	textarea textarea.Model
	spinner  spinner.Model

	loading        bool
	loadingMessage string
	err            error

	convList         list.Model
	convSearchInput  textinput.Model
	convSearchQuery  string
	allConversations []clientapi.AssistantConversationClientModel

	showSlashMenu bool
	slashList     list.Model
	slashCommands []SlashCommand

	escPendingClear  bool
	ctrlCPendingExit bool
	exiting          bool

	activeSearchCTA  *clientapi.AssistantMessageDataClientModelClientSearchCta
	activeRequestCTA *clientapi.AssistantMessageDataClientModelClientRequestCta
	ctaItems         []ctaItem
	ctaCursor        int

	ctaResourcesTotal   int32
	ctaResourcesHasMore bool
	ctaBundlesTotal     int32
	ctaBundlesHasMore   bool

	requestButtonCursor int
	editedJustification string
}

type SlashCommand struct {
	Name        string
	Description string
}

type ctaItem struct {
	Type             string
	ID               string
	Name             string
	Path             string
	ResourceType     string
	ResourceTypePath string
	IntegrationName  string
	IntegrationType  string
}

func NewModel(ctx context.Context, client *aponoapi.AponoClient) Model {
	ta := textarea.New()
	ta.Placeholder = DefaultPlaceholder
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(DefaultTextareaWidth)
	ta.SetHeight(DefaultTextareaHeight)
	ta.MaxHeight = MaxTextareaHeight
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	ta.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return PromptChar
		}
		return PromptContinue
	})

	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()

	s := spinner.New()
	s.Spinner = spinner.Dot

	convList := newConvList(DefaultTextareaWidth, DefaultListHeight)
	slashList := newSlashList(DefaultTextareaWidth, DefaultListHeight)
	searchInput := textinput.New()
	searchInput.Placeholder = "Type to search..."
	searchInput.Prompt = "Search: "
	searchInput.CharLimit = 100

	return Model{
		ctx:             ctx,
		client:          client,
		viewState:       ViewWelcome,
		conversationID:  newConversationID(),
		textarea:        ta,
		spinner:         s,
		convList:        convList,
		convSearchInput: searchInput,
		slashList:       slashList,
		slashCommands: []SlashCommand{
			{Name: SlashCmdNew, Description: "Start a new conversation"},
			{Name: SlashCmdResume, Description: "Resume an existing conversation"},
			{Name: SlashCmdQuit, Description: "Exit the assistant"},
		},
	}
}

func newConversationID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 4)
	m.convList.SetWidth(width)
	m.convList.SetHeight(height - 6)
	m.slashList.SetWidth(width)
	m.slashList.SetHeight(10)
	m.convSearchInput.Width = width - 10
}

func (m *Model) AddUserMessage(content string) {
	msg := ChatMessage{
		ID:          strconv.FormatInt(time.Now().UnixNano(), 10),
		Role:        RoleUser,
		Content:     content,
		CreatedDate: time.Now(),
	}
	m.messages = append(m.messages, msg)
}

func (m *Model) AddAssistantMessage(apiMsg clientapi.AssistantMessageClientModel) {
	content := renderMessageContent(apiMsg.Data)
	msg := ChatMessage{
		ID:          apiMsg.Id,
		Role:        apiMsg.Role,
		Content:     content,
		RawData:     apiMsg.Data,
		CreatedDate: apiMsg.CreatedDate,
	}
	m.messages = append(m.messages, msg)
}

func (m *Model) AddMessageFromHistory(apiMsg clientapi.AssistantMessageClientModel) {
	var content string

	if apiMsg.Role == RoleUserMessage {
		content = extractUserMessageContent(apiMsg.Data)
	} else {
		content = renderMessageContent(apiMsg.Data)
	}

	msg := ChatMessage{
		ID:          apiMsg.Id,
		Role:        apiMsg.Role,
		Content:     content,
		RawData:     apiMsg.Data,
		CreatedDate: apiMsg.CreatedDate,
	}
	m.messages = append(m.messages, msg)
}

func extractUserMessageContent(data []clientapi.AssistantMessageDataClientModel) string {
	for _, d := range data {
		if d.HasMarkdown() {
			md := d.GetMarkdown()
			return md.Content
		}
	}
	return ""
}

func renderMessageContent(data []clientapi.AssistantMessageDataClientModel) string {
	var content strings.Builder
	for _, d := range data {
		if d.HasMarkdown() {
			md := d.GetMarkdown()
			if text := strings.TrimSpace(md.Content); text != "" {
				if content.Len() > 0 {
					content.WriteString("\n")
				}
				content.WriteString(text)
			}
		}
		if d.HasClientSearchCta() {
			cta := d.GetClientSearchCta()
			if content.Len() > 0 {
				content.WriteString("\n")
			}
			content.WriteString(renderSearchCTA(cta))
		}
		if d.HasClientRequestCta() {
			cta := d.GetClientRequestCta()
			if content.Len() > 0 {
				content.WriteString("\n\n")
			}
			content.WriteString(renderRequestCTA(cta))
		}
	}
	return content.String()
}

func renderSearchCTA(cta clientapi.AssistantMessageDataClientModelClientSearchCta) string {
	var content string

	if cta.HasResources() {
		resources := cta.GetResources()
		resourceData := resources.GetData()
		if len(resourceData) > 0 {
			header := fmt.Sprintf("Resources (%d of %d)", len(resourceData), resources.Total)
			content += "\n" + titleStyle.Render(header) + "\n"
			content += resourceLabelStyle.Render(strings.Repeat("─", len(header))) + "\n"

			for i, r := range resourceData {
				content += renderResourceCard(i+1, r)
			}
			if resources.HasMore {
				remaining := int(resources.Total) - len(resourceData)
				content += resourceLabelStyle.Render(fmt.Sprintf("  (and %d more...)", remaining)) + "\n"
			}
		}
	}

	if cta.HasBundles() {
		bundles := cta.GetBundles()
		bundleData := bundles.GetData()
		if len(bundleData) > 0 {
			header := fmt.Sprintf("Bundles (%d of %d)", len(bundleData), bundles.Total)
			content += "\n" + titleStyle.Render(header) + "\n"
			content += resourceLabelStyle.Render(strings.Repeat("─", len(header))) + "\n"
			for i, b := range bundleData {
				content += renderBundleCard(i+1, b)
			}
		}
	}

	return content
}

func renderResourceCard(index int, r clientapi.ResourceClientModel) string {
	var card strings.Builder

	card.WriteString(resourceNameStyle.Render(fmt.Sprintf("[%d] %s", index, r.Name)))
	card.WriteString("\n")

	card.WriteString("    ")
	card.WriteString(resourceLabelStyle.Render("Path: "))
	card.WriteString(resourceValueStyle.Render(r.SourceId))
	card.WriteString("\n")

	typeInfo := r.Type.Name
	if r.Type.DisplayPath != "" && r.Type.DisplayPath != r.Type.Name {
		typeInfo = r.Type.Name + " · " + r.Type.DisplayPath
	}
	card.WriteString("    ")
	card.WriteString(resourceLabelStyle.Render("Type: "))
	card.WriteString(resourceValueStyle.Render(typeInfo))
	card.WriteString("\n")

	integrationInfo := r.Integration.Name
	if r.Integration.TypeDisplayName != "" {
		integrationInfo = r.Integration.Name + " (" + r.Integration.TypeDisplayName + ")"
	}
	card.WriteString("    ")
	card.WriteString(resourceLabelStyle.Render("Integration: "))
	card.WriteString(resourceValueStyle.Render(integrationInfo))
	card.WriteString("\n")

	card.WriteString("\n")
	return card.String()
}

func renderBundleCard(index int, b clientapi.BundleClientModel) string {
	return resourceNameStyle.Render(fmt.Sprintf("[%d] %s", index, b.Name)) + "\n" +
		"    " + resourceLabelStyle.Render("Type: ") + resourceValueStyle.Render(ResourceTypeBundleDisplayName) + "\n\n"
}

func renderRequestCTA(cta clientapi.AssistantMessageDataClientModelClientRequestCta) string {
	var inner strings.Builder

	inner.WriteString(titleStyle.Render("Access Request"))
	inner.WriteString("\n")
	inner.WriteString(resourceLabelStyle.Render("──────────────"))

	if cta.HasResourcesRequest() {
		req := cta.GetResourcesRequest()
		for _, e := range req.GetEntitlements() {
			resource := e.GetResource()
			permission := e.GetPermission()

			inner.WriteString("\n")
			inner.WriteString(resourceLabelStyle.Render("Resource:      "))
			inner.WriteString(resourceNameStyle.Render(resource.Name))

			if resource.Integration.Name != "" {
				info := resource.Integration.Name
				if resource.Integration.TypeDisplayName != "" {
					info += " (" + resource.Integration.TypeDisplayName + ")"
				}
				inner.WriteString("\n")
				inner.WriteString(resourceLabelStyle.Render("Integration:   "))
				inner.WriteString(resourceValueStyle.Render(info))
			}

			if resource.Type.Name != "" {
				inner.WriteString("\n")
				inner.WriteString(resourceLabelStyle.Render("Type:          "))
				inner.WriteString(resourceValueStyle.Render(resource.Type.Name))
			}

			if resource.SourceId != "" {
				inner.WriteString("\n")
				inner.WriteString(resourceLabelStyle.Render("Path:          "))
				inner.WriteString(resourceValueStyle.Render(resource.SourceId))
			}

			inner.WriteString("\n")
			inner.WriteString(resourceLabelStyle.Render("Permission:    "))
			inner.WriteString(resourceValueStyle.Render(permission.Name))
		}

		if req.Justification != "" {
			inner.WriteString("\n")
			inner.WriteString(resourceLabelStyle.Render("Justification: "))
			inner.WriteString(resourceValueStyle.Render(req.Justification))
		}

		if req.RequiresApproval {
			inner.WriteString("\n")
			inner.WriteString(ctaWarningStyle.Render("⚠ Requires Approval"))
		}
	}

	if cta.HasBundlesRequest() {
		req := cta.GetBundlesRequest()

		inner.WriteString("\n")
		inner.WriteString(resourceLabelStyle.Render("Type:          "))
		inner.WriteString(resourceValueStyle.Render("Bundle"))

		if req.Justification != "" {
			inner.WriteString("\n")
			inner.WriteString(resourceLabelStyle.Render("Justification: "))
			inner.WriteString(resourceValueStyle.Render(req.Justification))
		}

		if req.RequiresApproval {
			inner.WriteString("\n")
			inner.WriteString(ctaWarningStyle.Render("⚠ Requires Approval"))
		}
	}

	return requestBoxStyle.Render(inner.String())
}

func (m *Model) StartNewConversation() {
	m.conversationID = newConversationID()
	m.messages = nil
	m.suggestions = nil
	m.viewState = ViewWelcome
	m.err = nil
}

func FormatMessage(msg ChatMessage) string {
	var content strings.Builder
	messageText := strings.TrimSpace(msg.Content)

	if msg.Role == RoleUser || msg.Role == RoleUserMessage {
		content.WriteString(userLabelStyle.Render("You"))
		content.WriteString("\n")
		content.WriteString(indentLines(messageText, "  "))
	} else {
		content.WriteString(assistantLabelStyle.Render("Apono Assist"))
		content.WriteString("\n")
		content.WriteString(indentLines(messageText, "  "))
	}

	return content.String()
}

func indentLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func FormatSuggestions(suggestions []clientapi.AssistantConversationSuggestionClientModel) string {
	if len(suggestions) == 0 {
		return ""
	}
	var chips []string
	for _, s := range suggestions {
		chips = append(chips, suggestionChipStyle.Render(s.Content))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, chips...)
}

func (m *Model) GetMatchingSlashCommands(prefix string) []SlashCommand {
	if prefix == "" || prefix[0] != '/' {
		return nil
	}

	var matches []SlashCommand
	for _, cmd := range m.slashCommands {
		if strings.HasPrefix(cmd.Name, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}
