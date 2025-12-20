package assist

import (
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

type assistantResponseMsg struct {
	Response *clientapi.AssistantMessageResponseClientModel
	Err      error
}

type conversationListMsg struct {
	Conversations []clientapi.AssistantConversationClientModel
	Err           error
}

type conversationHistoryMsg struct {
	ConversationID string
	Messages       []clientapi.AssistantMessageClientModel
	Err            error
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

type accessRequestResultMsg struct {
	RequestID     string
	Request       *clientapi.AccessRequestClientModel
	WaitForStatus bool
	Err           error
}
