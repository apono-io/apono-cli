package selectors

import (
	"context"
	"fmt"
	"sort"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/services"
)

func RunSessionsSelector(ctx context.Context, client *aponoapi.AponoClient, requestID string) (*clientapi.AccessSessionClientModel, error) {
	var requestIdsFilter []string
	if requestID != "" {
		requestIdsFilter = []string{requestID}
	}

	sessions, err := services.ListAccessSessions(ctx, client, nil, nil, requestIdsFilter)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no active access found, create a new request by running this command: apono request create")
	}

	sessionByID := make(map[string]clientapi.AccessSessionClientModel)
	var options []listselect.SelectOption
	for _, session := range sessions {
		options = append(options, listselect.SelectOption{
			ID:    session.Id,
			Label: fmt.Sprintf("%s/%s", session.Type.Name, session.Name),
		})
		sessionByID[session.Id] = session
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	sessionsInput := listselect.SelectInput{
		Title:         "Select access",
		PostTitle:     "Selected access",
		Options:       options,
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedItems, err := listselect.LaunchSelector(sessionsInput)
	if err != nil {
		return nil, err
	}

	selectedSession, ok := sessionByID[selectedItems[0].ID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	return &selectedSession, nil
}
