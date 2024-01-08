package inputs

import (
	"fmt"
)

const (
	integrationInputTitle   = "Select Integration:"
	resourceTypeInputTitle  = "Select Resource Type:"
	resourceInputTitle      = "Select Resources:"
	permissionInputTitle    = "Select Permissions:"
	justificationInputTitle = "Enter Justification:"
)

var (
	beforeSelectIcon   = prefixIconColor.Sprint("?")
	afterSelectIcon    = prefixIconColor.Sprint("✓")
	abortingInputTitle = quitTextStyle.Render("Aborting...")
	//integrationInputHelp = "Use ↑/↓ to select, → to select, ← to deselect, and ↵ to confirm."
)

func getIntegrationTitle(integrationName string) string {
	if integrationName != "" {
		return fmt.Sprintf("%s %s %s", afterSelectIcon, integrationInputTitle, chosenItemsColor.Sprint(integrationName))
	} else {
		return fmt.Sprintf("%s %s", beforeSelectIcon, integrationInputTitle)
	}
}

func getResourceTypeTitle(resourceTypeName string) string {
	if resourceTypeName != "" {
		return fmt.Sprintf("%s %s %s", afterSelectIcon, resourceTypeInputTitle, chosenItemsColor.Sprint(resourceTypeName))
	} else {
		return fmt.Sprintf("%s %s", beforeSelectIcon, resourceTypeInputTitle)
	}
}

func getResourcesTitle(resourceNames []string) string {
	if len(resourceNames) > 0 {
		return fmt.Sprintf("%s %s %s", afterSelectIcon, resourceInputTitle, joinNames(resourceNames))
	} else {
		return fmt.Sprintf("%s %s", beforeSelectIcon, resourceInputTitle)
	}
}

func getPermissionsTitle(permissionNames []string) string {
	if len(permissionNames) > 0 {
		return fmt.Sprintf("%s %s %s", afterSelectIcon, permissionInputTitle, joinNames(permissionNames))
	} else {
		return fmt.Sprintf("%s %s", beforeSelectIcon, permissionInputTitle)
	}
}

func getJustificationTitle(justification string) string {
	if justification != "" {
		return fmt.Sprintf("%s %s %s", afterSelectIcon, justificationInputTitle, chosenItemsColor.Sprint(justification))
	} else {
		return fmt.Sprintf("%s %s", beforeSelectIcon, justificationInputTitle)
	}
}

func joinNames(names []string) string {
	var result string
	for i, name := range names {
		if i != len(names)-1 {
			result += chosenItemsColor.Sprint(name) + ", "
		} else {
			result += chosenItemsColor.Sprint(name)
		}
	}

	return result
}
