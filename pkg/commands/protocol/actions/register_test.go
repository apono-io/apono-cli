package actions

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

func TestHandlerScriptBody_substitutesAponoBinaryPath(t *testing.T) {
	body := handlerScriptBody("/fake/path/to/apono")

	if strings.Contains(body, "__APONO_BINARY__") {
		t.Errorf("expected __APONO_BINARY__ placeholder to be replaced, got:\n%s", body)
	}
	if !strings.Contains(body, `exec "/fake/path/to/apono" access use`) {
		t.Errorf("expected handler.sh to invoke the absolute apono path, got:\n%s", body)
	}
}

func TestHandlerScriptBody_setsAccountEnvAndRequiredFlags(t *testing.T) {
	body := handlerScriptBody("/usr/local/bin/apono")

	wantSubstrings := []string{
		`export _APONO_ACCOUNT_ID_="$account"`,
		`--client "$client"`,
		`exit 64`,
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(body, want) {
			t.Errorf("expected handler.sh body to contain %q, got:\n%s", want, body)
		}
	}
}

func TestHandlerScriptBody_overwritesPreviousPath(t *testing.T) {
	first := handlerScriptBody("/fake/first")
	second := handlerScriptBody("/fake/second")

	if !strings.Contains(second, `"/fake/second"`) {
		t.Errorf("expected second body to contain second path, got:\n%s", second)
	}
	if strings.Contains(second, `"/fake/first"`) {
		t.Errorf("each call should be independent — second body shouldn't carry first path, got:\n%s", second)
	}
	if first == second {
		t.Error("different inputs should produce different bodies")
	}
}

func TestRegister_rejectsNonDarwin(t *testing.T) {
	if runtime.GOOS == darwinOS {
		t.Skip("non-darwin guard test")
	}

	cmd := Register()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Errorf("expected non-darwin to error mentioning macOS, got %v", err)
	}
}

func TestUnregister_rejectsNonDarwin(t *testing.T) {
	if runtime.GOOS == darwinOS {
		t.Skip("non-darwin guard test")
	}

	cmd := Unregister()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Errorf("expected non-darwin to error mentioning macOS, got %v", err)
	}
}
