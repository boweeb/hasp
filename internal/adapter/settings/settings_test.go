package settings

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad_MissingFileIsNotAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist", "settings.toml")

	before, statErr := os.Stat(filepath.Dir(path))
	if statErr == nil {
		t.Fatalf("fixture error: %s unexpectedly exists", filepath.Dir(path))
	}
	_ = before

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%s) = %v, want nil error for a missing file", path, err)
	}
	if got != (Settings{}) {
		t.Errorf("Load(%s) = %+v, want the zero value", path, got)
	}

	// Load must not create the file or its parent directory (hasp reads settings, never writes
	// them, D16) — a missing file staying missing after Load is itself part of the contract.
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Load(%s) created a file where none existed", path)
	}
}

func TestLoad_DecodesAValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	content := "# hasp reads this file. hasp never writes to it.\n[new_key]\ndefault_passphrase_mode = \"stdin\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}
	want := Settings{NewKey: NewKeySettings{DefaultPassphraseMode: "stdin"}}
	if got != want {
		t.Errorf("Load(%s) = %+v, want %+v", path, got, want)
	}
}

func TestLoad_MalformedFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	if err := os.WriteFile(path, []byte("this is not [ valid toml"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatalf("Load(%s) = nil error, want a decode error for malformed TOML", path)
	}
}

// TestSettings_FieldSetIsGolden is the explicit §12 guard test: the v1 keyset is exactly
// [new_key] default_passphrase_mode (D16's admission rule), and this asserts the exact field set
// of the decoded Go struct by name, so a PR adding a field changes this test's expected list
// visibly — the same discipline T1's "no decrypt call site" and T13's "go list -deps" guards use.
func TestSettings_FieldSetIsGolden(t *testing.T) {
	assertFieldSet(t, reflect.TypeOf(Settings{}), []string{"NewKey"})
	assertFieldSet(t, reflect.TypeOf(NewKeySettings{}), []string{"DefaultPassphraseMode"})
}

func assertFieldSet(t *testing.T, typ reflect.Type, want []string) {
	t.Helper()
	var got []string
	for i := 0; i < typ.NumField(); i++ {
		got = append(got, typ.Field(i).Name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s field set = %v, want %v", typ.Name(), got, want)
	}
}

// TestSettings_FileStaysByteIdentical is §12's paired guard: a test needing a settings fixture
// writes it as setup, never as a side effect of the code under test — Load must never mutate the
// file it reads.
func TestSettings_FileStaysByteIdentical(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	content := []byte("[new_key]\ndefault_passphrase_mode = \"none\"\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(content) {
		t.Errorf("settings file changed after Load: got %q, want %q", after, content)
	}
}
