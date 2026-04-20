package actions

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"

	"github.com/spf13/cobra"
)

// Supported URI actions:
//   apono://terminal/{sessionId} — always open Terminal with `apono access use <sessionId> --run`
//   apono://connect/{sessionId}  — smart routing: open the best app for the session type, fall back to Terminal

func ProtocolHandle() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "handle <uri>",
		Short:  "Handle an apono:// URI (invoked by the macOS URL handler app)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != darwinOS {
				return fmt.Errorf("protocol handler is only supported on macOS")
			}

			if len(args) == 0 {
				return fmt.Errorf("missing URI argument")
			}

			action, sessionID, err := parseURI(args[0])
			if err != nil {
				return err
			}

			switch action {
			case "terminal":
				return openTerminal(sessionID)
			case "connect":
				return handleConnect(cmd, sessionID)
			default:
				return fmt.Errorf("unsupported action %q", action)
			}
		},
	}

	return cmd
}

// validSessionID matches session IDs like "kubernetes-k8s-prod-bb743e" or "aws-rds-postgresql-558143244530/apono-dev-dba923".
var validSessionID = regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)

func parseURI(rawURI string) (action string, sessionID string, err error) {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return "", "", fmt.Errorf("invalid URI: %w", err)
	}

	if parsed.Scheme != "apono" {
		return "", "", fmt.Errorf("unexpected scheme %q, expected \"apono\"", parsed.Scheme)
	}

	// URI: apono://{action}/{sessionId}
	// url.Parse puts the action in Host and /{sessionId} in Path
	action = parsed.Host
	if action == "" {
		return "", "", fmt.Errorf("missing action in URI")
	}

	sessionID = strings.TrimPrefix(parsed.Path, "/")
	if sessionID == "" {
		return "", "", fmt.Errorf("missing session ID in URI")
	}

	if !validSessionID.MatchString(sessionID) {
		return "", "", fmt.Errorf("invalid session ID: %q", sessionID)
	}

	return action, sessionID, nil
}

func openTerminal(sessionID string) error {
	accessCommand := fmt.Sprintf(
		"export PATH=/usr/local/bin:/opt/homebrew/bin:$PATH && apono access use %s --run",
		sessionID,
	)
	script := fmt.Sprintf(`tell application "Terminal"
	activate
	do script "%s"
end tell`, accessCommand)

	osascript := exec.Command("osascript", "-e", script) //nolint:gosec // script is built from validated sessionID
	if output, err := osascript.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to open Terminal.app: %s: %s", err, string(output))
	}

	return nil
}

// SQL-ish integration types mapped to DBeaver driver names.
var sqlIntegrationTypes = map[string]string{
	"postgresql":         "postgresql",
	"aws-rds-postgresql": "postgresql",
	"mysql":              "mysql",
	"aws-rds-mysql":      "mysql",
	"mariadb":            "mariadb",
	"mssql":              "sqlserver",
	"clickhouse":         "clickhouse",
}

// handleConnect fetches session details and routes to the best app.
// Falls back to Terminal if the session type has no app handler or the app isn't installed.
func handleConnect(cmd *cobra.Command, sessionID string) error {
	client, err := aponoapi.GetClient(cmd.Context())
	if err != nil {
		return openTerminal(sessionID)
	}

	session, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSession(cmd.Context(), sessionID).Execute()
	if err != nil {
		return openTerminal(sessionID)
	}

	integrationType := session.Integration.Type

	switch {
	case isSQLIntegration(integrationType):
		if connectErr := connectSQL(cmd, client, sessionID, integrationType); connectErr != nil {
			// DBeaver not installed or failed — fall back to Terminal
			return openTerminal(sessionID)
		}

		return nil

	// TODO: add more cases here as we support more app types

	default:
		return openTerminal(sessionID)
	}
}

func isSQLIntegration(integrationType string) bool {
	_, ok := sqlIntegrationTypes[integrationType]
	return ok
}

func connectSQL(cmd *cobra.Command, client *aponoapi.AponoClient, sessionID string, integrationType string) error {
	dbeaverDriver := sqlIntegrationTypes[integrationType]

	dbeaverPath := findDBeaver()
	if dbeaverPath == "" {
		return fmt.Errorf("DBeaver not found")
	}

	accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(cmd.Context(), sessionID).Execute()
	if err != nil {
		return err
	}

	return openDBeaver(dbeaverPath, dbeaverDriver, accessDetails.GetJson())
}

func findDBeaver() string {
	paths := []string{
		"/Applications/DBeaver.app/Contents/MacOS/dbeaver",
		"/Applications/DBeaver Community.app/Contents/MacOS/dbeaver",
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			home+"/Applications/DBeaver.app/Contents/MacOS/dbeaver",
			home+"/Applications/DBeaver Community.app/Contents/MacOS/dbeaver",
		)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

func openDBeaver(dbeaverPath string, driver string, details map[string]any) error {
	host, _ := details["host"].(string)
	port, _ := details["port"].(string)
	dbName, _ := details["db_name"].(string)
	username, _ := details["username"].(string)
	password, _ := details["password"].(string)

	conParts := []string{
		"driver=" + driver,
		"host=" + host,
		"port=" + port,
		"database=" + dbName,
		"user=" + username,
		"password=" + password,
		"name=Apono: " + dbName,
		"connect=true",
		"openConsole=true",
		"create=true",
		"save=false",
	}

	conStr := strings.Join(conParts, "|")

	dbeaverCmd := exec.Command(dbeaverPath, "-con", conStr) //nolint:gosec // dbeaverPath is from known install locations
	if err := dbeaverCmd.Start(); err != nil {
		return fmt.Errorf("failed to launch DBeaver: %w", err)
	}

	return nil
}
