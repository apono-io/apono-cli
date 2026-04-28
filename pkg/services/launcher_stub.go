package services

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

// TODO(DVL-8786): replace stub with real BE fetch once the endpoint and clientapi regen land.
func stubLaunchersForSession(_ context.Context, _ *aponoapi.AponoClient, sessionId string) (*LauncherFetchResult, error) {
	cacheSetup := "mkdir -p ~/.apono/cache/ && echo 'c3R1Yi1wYXNz' | base64 -d > ~/.apono/cache/" + sessionId +
		" && DB_PASSWORD=$(cat ~/.apono/cache/" + sessionId + ")"

	return &LauncherFetchResult{
		ConsumedBy: ConsumedByOS,
		Launchers: []LauncherClientModel{
			{
				Id:          "dbeaver",
				DisplayName: "DBeaver",
				Kind:        LauncherKindGUI,
				Setup:       cacheSetup,
				Invocation:  "dbeaver -con 'driver=postgresql|host=localhost|port=5432|database=postgres|user=apono|password='\"$DB_PASSWORD\"'|name=apono-" + sessionId + "'",
			},
			{
				Id:          "tableplus",
				DisplayName: "TablePlus",
				Kind:        LauncherKindGUI,
				Setup:       cacheSetup,
				Invocation:  "open -a TablePlus 'postgres://apono:'\"$DB_PASSWORD\"'@localhost:5432/postgres'",
			},
			{
				Id:          "k9s",
				DisplayName: "k9s",
				Kind:        LauncherKindTUI,
				Setup:       "mkdir -p ~/.apono/cache && echo 'apiVersion: v1\\nkind: Config\\n# stub kubeconfig' > ~/.apono/cache/" + sessionId + ".kubeconfig",
				Invocation:  "k9s --kubeconfig ~/.apono/cache/" + sessionId + ".kubeconfig",
			},
			{
				Id:          "cli",
				DisplayName: "Open in Terminal",
				Kind:        LauncherKindTUI,
				Setup:       cacheSetup,
				Invocation:  "psql -h localhost -p 5432 -U apono -d postgres",
			},
		},
	}, nil
}
