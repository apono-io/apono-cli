package selectors

import (
	"strings"
	"testing"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func TestRunLauncherClientSelector_emptyInput_returnsError(t *testing.T) {
	_, err := RunLauncherClientSelector(nil)
	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}
	if !strings.Contains(err.Error(), "no launcher clients") {
		t.Errorf("expected error to mention no launcher clients, got %q", err.Error())
	}

	_, err = RunLauncherClientSelector([]clientapi.LauncherClientModel{})
	if err == nil {
		t.Fatal("expected error for empty slice, got nil")
	}
}
