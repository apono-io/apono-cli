package connect

import (
	"testing"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func TestIsInstalled_terminalAlwaysTrue(t *testing.T) {
	got := IsInstalled(clientapi.LauncherClientModel{
		Id:           "terminal",
		LauncherType: ClientKindTERMINAL,
	})
	if !got {
		t.Errorf("IsInstalled(TERMINAL) = false, want true")
	}
}
