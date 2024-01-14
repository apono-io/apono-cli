package interactive

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func RunSessionDetailsTypeSelector(session *clientapi.AccessSessionClientModel) (string, error) {
	if len(session.ConnectionMethods) == 0 {
		return "", fmt.Errorf("no available connection methods")
	}

	if len(session.ConnectionMethods) == 1 {
		return session.ConnectionMethods[0], nil
	}

	var options []listselect.SelectOption
	for _, method := range session.ConnectionMethods {
		options = append(options, listselect.SelectOption{
			ID:    method,
			Label: cases.Title(language.English).String(method),
		})
	}

	sessionsInput := listselect.SelectInput{
		Title:   styles.BeforeSelectingItemsTitleStyle("Select connection method"),
		Options: options,
		PostMessage: func(s []listselect.SelectOption) string {
			return styles.AfterSelectingItemsTitleStyle("Selected connection method", []string{s[0].Label})
		},
	}

	selectedItems, err := listselect.LaunchSelector(sessionsInput)
	if err != nil {
		return "", err
	}

	return selectedItems[0].ID, nil
}
