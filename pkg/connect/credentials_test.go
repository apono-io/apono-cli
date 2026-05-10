package connect

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestEncodePassword_url_escapesSpecialChars(t *testing.T) {
	got := encodePassword(`p@ss w&rd!`, passwordEncodingURL)
	want := `p%40ss+w%26rd%21`
	if got != want {
		t.Errorf("encodePassword url = %q, want %q", got, want)
	}
}

func TestEncodePassword_raw_passthrough(t *testing.T) {
	in := `p@ss w&rd!`
	if got := encodePassword(in, ""); got != in {
		t.Errorf("encodePassword empty-encoding = %q, want unchanged %q", got, in)
	}
	if got := encodePassword(in, "raw"); got != in {
		t.Errorf("encodePassword raw = %q, want unchanged %q", got, in)
	}
}

func TestEncodePassword_unknownEncoding_fallsBackToRaw(t *testing.T) {
	in := `secret`
	if got := encodePassword(in, "rot13-not-a-thing"); got != in {
		t.Errorf("encodePassword unknown encoding = %q, want unchanged %q", got, in)
	}
}

func TestReadCachedPassword_returnsDecodedContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheDir := filepath.Join(home, ".apono", "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	sessionID := "session-123"
	want := `p@ss w&rd!`
	if err := os.WriteFile(filepath.Join(cacheDir, sessionID), []byte(base64.StdEncoding.EncodeToString([]byte(want))), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	got, err := readCachedPassword(sessionID)
	if err != nil {
		t.Fatalf("readCachedPassword: %v", err)
	}
	if got != want {
		t.Errorf("readCachedPassword = %q, want %q", got, want)
	}
}

func TestReadCachedPassword_missingFile_errorWrapped(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, err := readCachedPassword("does-not-exist")
	if err == nil {
		t.Fatal("expected error for missing cache file, got nil")
	}
}

func TestReadCachedPassword_invalidBase64_errorWrapped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheDir := filepath.Join(home, ".apono", "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "session-123"), []byte("not-valid-base64!!!"), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	_, err := readCachedPassword("session-123")
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}
