package banner

import (
	"fmt"
	"io"

	"github.com/gookit/color"

	"github.com/apono-io/apono-cli/pkg/version"
)

const logo = `
    _    ____   ___  _   _  ___       ____ _     ___
   / \  |  _ \ / _ \| \ | |/ _ \     / ___| |   |_ _|
  / _ \ | |_) | | | |  \| | | | |   | |   | |    | |
 / ___ \|  __/| |_| | |\  | |_| |   | |___| |___ | |
/_/   \_\_|    \___/|_| \_|\___/     \____|_____|___|`

const separator = "────────────────────────────────────────────────────────────────"

// UserSessionInfo contains enriched user and account information
type UserSessionInfo struct {
	AccountID   string
	AccountName string
	UserID      string
	UserName    string
	UserEmail   string
}

func Display(w io.Writer, versionInfo *version.VersionInfo, sessionInfo *UserSessionInfo, profileName string) error {
	fmt.Fprintln(w, color.Cyan.Sprint(logo))
	fmt.Fprintln(w)

	printField(w, "Version", versionInfo.Version)
	printField(w, "Commit", truncateCommit(versionInfo.Commit))
	printField(w, "Build Date", versionInfo.BuildDate)

	printField(w, "Profile", color.Cyan.Sprint(profileName))

	if sessionInfo != nil {
		if sessionInfo.AccountName != "" {
			accountDisplay := fmt.Sprintf("%s (%s)", sessionInfo.AccountName, sessionInfo.AccountID)
			printField(w, "Account", color.Green.Sprint(accountDisplay))
		}

		if sessionInfo.UserEmail != "" {
			userDisplay := fmt.Sprintf("%s (%s)", sessionInfo.UserName, sessionInfo.UserEmail)
			printField(w, "User", color.Green.Sprint(userDisplay))
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, color.Gray.Sprint(separator))
	fmt.Fprintln(w)

	return nil
}

func printField(w io.Writer, label, value string) {
	labelFormatted := color.Gray.Sprintf("%-12s", label+":")
	fmt.Fprintf(w, "%s %s\n", labelFormatted, value)
}

func truncateCommit(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}
