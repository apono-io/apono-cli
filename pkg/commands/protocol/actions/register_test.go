package actions

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildAppBundle_writesExpectedLayout(t *testing.T) {
	if runtime.GOOS != darwinOS {
		t.Skip("osacompile/defaults/codesign require macOS")
	}

	tmp := t.TempDir()
	bundleDir := filepath.Join(tmp, bundleDirName)
	const fakeBinary = "/fake/path/to/apono"

	if err := buildAppBundle(bundleDir, fakeBinary); err != nil {
		t.Fatalf("buildAppBundle: %v", err)
	}

	wantPaths := []string{
		filepath.Join(bundleDir, "Contents", "Info.plist"),
		filepath.Join(bundleDir, "Contents", "MacOS", "applet"),
		filepath.Join(bundleDir, "Contents", "Resources", "Scripts", "main.scpt"),
		filepath.Join(bundleDir, "Contents", "Resources", "handler.sh"),
	}
	for _, p := range wantPaths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
}

func TestBuildAppBundle_handlerScriptIsExecutableAndSubstituted(t *testing.T) {
	if runtime.GOOS != darwinOS {
		t.Skip("osacompile required")
	}

	tmp := t.TempDir()
	bundleDir := filepath.Join(tmp, bundleDirName)
	const fakeBinary = "/fake/path/to/apono"

	if err := buildAppBundle(bundleDir, fakeBinary); err != nil {
		t.Fatalf("buildAppBundle: %v", err)
	}

	handlerPath := filepath.Join(bundleDir, "Contents", "Resources", "handler.sh")
	body, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatalf("read handler.sh: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, `exec "/fake/path/to/apono" access use`) {
		t.Errorf("expected handler.sh to invoke the absolute apono path, got:\n%s", bodyStr)
	}
	if strings.Contains(bodyStr, "__APONO_BINARY__") {
		t.Errorf("expected __APONO_BINARY__ placeholder to be replaced, got:\n%s", bodyStr)
	}

	info, err := os.Stat(handlerPath)
	if err != nil {
		t.Fatalf("stat handler.sh: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Errorf("expected handler.sh perm 0o755, got %o", perm)
	}
}

func TestBuildAppBundle_infoPlistHasURLScheme(t *testing.T) {
	if runtime.GOOS != darwinOS {
		t.Skip("defaults required")
	}

	tmp := t.TempDir()
	bundleDir := filepath.Join(tmp, bundleDirName)

	if err := buildAppBundle(bundleDir, "/fake/apono"); err != nil {
		t.Fatalf("buildAppBundle: %v", err)
	}

	plistPath := filepath.Join(bundleDir, "Contents", "Info.plist")

	cases := []struct {
		key         string
		mustContain string
	}{
		{"CFBundleIdentifier", bundleIdentifier},
		{"CFBundleName", bundleDisplayName},
		{"CFBundleDisplayName", bundleDisplayName},
		{"LSUIElement", "1"},
		{"CFBundleURLTypes", urlScheme},
	}
	for _, tc := range cases {
		out, err := exec.Command("defaults", "read", plistPath, tc.key).CombinedOutput()
		if err != nil {
			t.Errorf("defaults read %s failed: %v: %s", tc.key, err, string(out))
			continue
		}
		if !bytes.Contains(out, []byte(tc.mustContain)) {
			t.Errorf("Info.plist key %s: expected to contain %q, got %q", tc.key, tc.mustContain, string(out))
		}
	}
}

func TestBuildAppBundle_overwritesPreviousBundle(t *testing.T) {
	if runtime.GOOS != darwinOS {
		t.Skip("osacompile required")
	}

	tmp := t.TempDir()
	bundleDir := filepath.Join(tmp, bundleDirName)

	if err := buildAppBundle(bundleDir, "/fake/first"); err != nil {
		t.Fatalf("first build: %v", err)
	}
	if err := buildAppBundle(bundleDir, "/fake/second"); err != nil {
		t.Fatalf("second build: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(bundleDir, "Contents", "Resources", "handler.sh"))
	if err != nil {
		t.Fatalf("read handler.sh: %v", err)
	}
	if !strings.Contains(string(body), `"/fake/second"`) {
		t.Errorf("expected second build's binary path in handler.sh, got:\n%s", string(body))
	}
	if strings.Contains(string(body), `"/fake/first"`) {
		t.Errorf("expected first build's path to be overwritten, got:\n%s", string(body))
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
