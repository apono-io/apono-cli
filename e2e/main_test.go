//go:build e2e

// Package e2e runs the compiled apono binary against an in-process mock of the Apono API.
//
// Run with: make e2e (or go test -tags e2e ./e2e/...).
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// binPath is the apono binary built once for the whole suite.
var binPath string

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	dir, err := os.MkdirTemp("", "apono-e2e-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create temp dir:", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()

	binPath = filepath.Join(dir, "apono")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	args := []string{"build", "-o", binPath}
	// E2E_COVERDIR collects coverage from the binary: go tool covdata textfmt -i=$E2E_COVERDIR -o e2e.out
	if os.Getenv("E2E_COVERDIR") != "" {
		args = append(args, "-cover", "-coverpkg=github.com/apono-io/apono-cli/...")
	}
	args = append(args, "github.com/apono-io/apono-cli/cmd/apono")

	build := exec.Command("go", args...) //nolint:gosec // args are built from constants in this file
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err = build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build apono binary:", err)
		return 1
	}

	return m.Run()
}
