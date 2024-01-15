package interactive

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
)

func RunSessionDetailsTypeSelector(session *clientapi.AccessSessionClientModel) (string, error) {
	var options []listselect.SelectOption
	for _, method := range session.ConnectionMethods {
		options = append(options, listselect.SelectOption{
			ID:    method,
			Label: cases.Title(language.English).String(method),
		})
	}

	sessionsInput := listselect.SelectInput{
		Title:     "Select connection method",
		PostTitle: "Selected connection method",
		Options:   options,
	}

	selectedItems, err := listselect.LaunchSelector(sessionsInput)
	if err != nil {
		return "", err
	}

	return selectedItems[0].ID, nil
}
