package connect

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const passwordWithSpecials = `p@ss w&rd!`

func TestWithoutSecrets(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		secrets []string
		want    string
	}{
		{
			name: "nil error stays nil",
			err:  nil, secrets: []string{"hunter2"},
			want: "",
		},
		{
			name:    "every occurrence of every secret is replaced",
			err:     errors.New("psql://u:hunter2@h rejected hunter2 (encoded hunter%32)"),
			secrets: []string{"hunter2", "hunter%32"},
			want:    "psql://u:***@h rejected *** (encoded ***)",
		},
		{
			name:    "blank secret leaves the text intact",
			err:     errors.New("nothing to hide"),
			secrets: []string{""},
			want:    "nothing to hide",
		},
		{
			name:    "repeated secret is applied once",
			err:     errors.New("hunter2"),
			secrets: []string{"hunter2", "hunter2"},
			want:    "***",
		},
		{
			name:    "no secrets leaves the text intact",
			err:     errors.New("connection refused"),
			secrets: nil,
			want:    "connection refused",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := withoutSecrets(tc.err, tc.secrets)
			if tc.want == "" {
				if got != nil {
					t.Errorf("withoutSecrets() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("withoutSecrets() = nil, want %q", tc.want)
			}
			if got.Error() != tc.want {
				t.Errorf("withoutSecrets() = %q, want %q", got.Error(), tc.want)
			}
		})
	}
}

func TestWithoutSecrets_doesNotExposeTheOriginalError(t *testing.T) {
	original := errors.New("psql failed for hunter2")

	got := withoutSecrets(original, []string{"hunter2"})

	if errors.Is(got, original) {
		t.Error("the redacted error must not unwrap back to the original")
	}
}

func TestEncodePassword_url_escapesSpecialChars(t *testing.T) {
	got := encodePassword(passwordWithSpecials, passwordEncodingURL)
	want := `p%40ss+w%26rd%21`
	if got != want {
		t.Errorf("encodePassword url = %q, want %q", got, want)
	}
}

func TestEncodePassword_raw_passthrough(t *testing.T) {
	in := passwordWithSpecials
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
	want := passwordWithSpecials
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

func TestReadCachedPassword_stripsTrailingNewlineFromPayload(t *testing.T) {
	// auth_command typically does `echo PASSWORD | base64 > cache`, which adds a
	// trailing newline before base64-encoding. Shell `$(base64 -d -i ...)` strips
	// it; our Go reader has to do the same or URL-encoding glues a `%0A` onto
	// the password.
	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheDir := filepath.Join(home, ".apono", "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	rawWithNewline := passwordWithSpecials + "\n"
	if err := os.WriteFile(filepath.Join(cacheDir, "sess-1"), []byte(base64.StdEncoding.EncodeToString([]byte(rawWithNewline))), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	got, err := readCachedPassword("sess-1")
	if err != nil {
		t.Fatalf("readCachedPassword: %v", err)
	}
	if got != passwordWithSpecials {
		t.Errorf("readCachedPassword = %q, want %q (no trailing newline)", got, passwordWithSpecials)
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
