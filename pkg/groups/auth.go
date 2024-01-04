package groups

import "github.com/spf13/cobra"

var AuthCommandsGroup = &cobra.Group{
	ID:    "auth",
	Title: "Authentication Commands",
}
