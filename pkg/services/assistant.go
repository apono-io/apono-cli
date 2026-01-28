package services

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"
)

func SendAssistantMessage(ctx context.Context, client *aponoapi.AponoClient, req *clientapi.AssistantMessageRequestModel) (*clientapi.AssistantMessageResponseClientModel, error) {
	resp, _, err := client.ClientAPI.AccessAssistantAPI.
		SendMessageToAssistant(ctx).
		AssistantMessageRequestModel(*req).
		Execute()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func ListAssistantConversations(ctx context.Context, client *aponoapi.AponoClient) ([]clientapi.AssistantConversationClientModel, error) {
	return utils.GetAllPages(ctx, client,
		func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.AssistantConversationClientModel, *clientapi.PaginationClientInfoModel, error) {
			resp, _, err := client.ClientAPI.AccessAssistantAPI.
				ListAssistantConversations(ctx).
				Skip(skip).
				Limit(1000).
				Execute()
			if err != nil {
				return nil, nil, err
			}
			return resp.Data, &resp.Pagination, nil
		})
}

func GetAssistantConversationHistory(ctx context.Context, client *aponoapi.AponoClient, conversationID string) ([]clientapi.AssistantMessageClientModel, error) {
	resp, _, err := client.ClientAPI.AccessAssistantAPI.
		GetAssistantConversationHistory(ctx, conversationID).
		Limit(100).
		Execute()
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}
