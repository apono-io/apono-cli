package urihandler

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/apono-io/apono-cli/pkg/utils"
)

func TestHandlerShellTemplate_invokesPATHResolvedApono(t *testing.T) {
	wantSubstrings := []string{
		`exec apono access use`,
		`export _APONO_ACCOUNT_ID_="$account"`,
		`--client "$client"`,
		`exit 64`,
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(handlerShellTemplate, want) {
			t.Errorf("expected handler.sh body to contain %q, got:\n%s", want, handlerShellTemplate)
		}
	}
	if strings.Contains(handlerShellTemplate, "__APONO_BINARY__") {
		t.Errorf("handler.sh should not contain __APONO_BINARY__ placeholder (uses PATH-resolved apono now), got:\n%s", handlerShellTemplate)
	}
}

func TestHandlerShellTemplate_writesTraceAndFailureContext(t *testing.T) {
	wantSubstrings := []string{
		filepath.Join(bundleParentDir, handlerLogFileName), // log path agrees with HandlerLogPath
		"log INFO",              // step logging
		"command -v apono",      // explicit resolve check before exec
		"trap",                  // failure backstop
		"PATH=$PATH",            // PATH captured on failure
		"exec apono access use", // still hands off to the CLI
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(handlerShellTemplate, want) {
			t.Errorf("expected handler.sh to contain %q, got:\n%s", want, handlerShellTemplate)
		}
	}
}

func TestRegister_rejectsNonDarwin(t *testing.T) {
	if runtime.GOOS == utils.DarwinOS {
		t.Skip("non-darwin guard test")
	}

	if err := Register(&bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Errorf("expected non-darwin to error mentioning macOS, got %v", err)
	}
}

func TestUnregister_rejectsNonDarwin(t *testing.T) {
	if runtime.GOOS == utils.DarwinOS {
		t.Skip("non-darwin guard test")
	}

	if err := Unregister(&bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Errorf("expected non-darwin to error mentioning macOS, got %v", err)
	}
}
