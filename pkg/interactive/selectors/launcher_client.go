package selectors

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
)

// RunLauncherClientSelector shows an interactive picker over the provided
// launcher clients and returns the selected client's id. Errors if the
// list is empty or no TTY is available (the latter surfaces from the
// underlying listselect.LaunchSelector).
func RunLauncherClientSelector(clients []clientapi.LauncherClientModel) (string, error) {
	if len(clients) == 0 {
		return "", fmt.Errorf("no launcher clients available for this session")
	}

	options := make([]listselect.SelectOption, 0, len(clients))
	for _, c := range clients {
		options = append(options, listselect.SelectOption{
			ID:    c.Id,
			Label: c.DisplayName,
		})
	}

	input := listselect.SelectInput{
		Title:                "Select how to open this session",
		PostTitle:            "Opening with",
		Options:              options,
		AutoSelectSingleItem: false,
	}

	selected, err := listselect.LaunchSelector(input)
	if err != nil {
		return "", err
	}
	return selected[0].ID, nil
}
