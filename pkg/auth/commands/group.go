package commands

import "github.com/spf13/cobra"

var Group = &cobra.Group{
	ID:    "auth",
	Title: "Authentication Commands",
}
