package utils

import "github.com/spf13/cobra"

func IsFlagSet(cmd *cobra.Command, flagName string) bool {
	flag := cmd.Flag(flagName)
	return flag != nil && flag.Changed
}
