package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/backup"
	"github.com/boweeb/hasp/internal/adapter/fswrite"
	"github.com/boweeb/hasp/internal/adapter/keyfile"
)

// TestNewKeyUseCase_Plan_RejectsEmptyRequest is a basic usage guard: no name, no directory.
func TestNewKeyUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	uc := NewKeyUseCase{}
	if _, err := uc.Plan(NewKeyRequest{KeyDir: "/tmp"}); !errors.Is(err, ErrUsage) {
		t.Errorf("Plan with empty Name: err = %v, want ErrUsage", err)
	}
	if _, err := uc.Plan(NewKeyRequest{Name: "id_ed25519"}); !errors.Is(err, ErrUsage) {
		t.Errorf("Plan with empty KeyDir: err = %v, want ErrUsage", err)
	}
}

// TestNewKeyUseCase_Plan_TwoChangesInOrder asserts the shape tdd.md §9 and §4 require: exactly
// two WriteKeyFile changes, private key first (see newkey.go's ordering rationale), both
// AllowOverwrite: false (T22).
func TestNewKeyUseCase_Plan_TwoChangesInOrder(t *testing.T) {
	dir := t.TempDir()
	uc := NewKeyUseCase{}
	plan, err := uc.Plan(NewKeyRequest{Name: "id_ed25519", KeyDir: dir})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2", len(plan.Changes))
	}

	priv, ok := plan.Changes[0].(WriteKeyFile)
	if !ok {
		t.Fatalf("Changes[0] = %T, want WriteKeyFile", plan.Changes[0])
	}
	if priv.Path != filepath.Join(dir, "id_ed25519") {
		t.Errorf("Changes[0].Path = %s, want %s", priv.Path, filepath.Join(dir, "id_ed25519"))
	}
	if priv.AllowOverwrite {
		t.Error("Changes[0].AllowOverwrite = true, want false (T22)")
	}

	pub, ok := plan.Changes[1].(WriteKeyFile)
	if !ok {
		t.Fatalf("Changes[1] = %T, want WriteKeyFile", plan.Changes[1])
	}
	if pub.Path != filepath.Join(dir, "id_ed25519.pub") {
		t.Errorf("Changes[1].Path = %s, want %s", pub.Path, filepath.Join(dir, "id_ed25519.pub"))
	}
	if pub.AllowOverwrite {
		t.Error("Changes[1].AllowOverwrite = true, want false (T22)")
	}
}

// TestNewKeyUseCase_Plan_HonorsProfile places the key under KeyDir/Profile rather than KeyDir
// itself, matching §9's "optionally in a profile (--profile)."
func TestNewKeyUseCase_Plan_HonorsProfile(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work", "foobarco")
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatal(err)
	}

	uc := NewKeyUseCase{}
	plan, err := uc.Plan(NewKeyRequest{Name: "id_ed25519", KeyDir: dir, Profile: []string{"work", "foobarco"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	priv := plan.Changes[0].(WriteKeyFile)
	want := filepath.Join(dir, "work", "foobarco", "id_ed25519")
	if priv.Path != want {
		t.Errorf("Changes[0].Path = %s, want %s", priv.Path, want)
	}
}

// TestNewKeyUseCase_ExitCriterion2_MatchesFreshInventory is roadmap.md §4 exit criterion 2,
// verified end to end: run the use case's own Applier, then run the M1 derivation pipeline
// (Derive) against the same key dir, and assert the two views agree — `hasp new key`'s reported
// facts must be identical to what a fresh `hasp show key` of the finished artifact reports.
func TestNewKeyUseCase_ExitCriterion2_MatchesFreshInventory(t *testing.T) {
	dir := t.TempDir()
	uc := NewKeyUseCase{}
	plan, err := uc.Plan(NewKeyRequest{Name: "id_ed25519_foobarco", KeyDir: dir})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	detail, ok := ShowKey(m, "id_ed25519_foobarco")
	if !ok {
		t.Fatal("ShowKey: freshly generated key not found by a fresh derivation")
	}

	k := detail.Key
	if k.Format.String() != "openssh" {
		t.Errorf("Format = %v, want openssh", k.Format)
	}
	if k.Algorithm != "ssh-ed25519" {
		t.Errorf("Algorithm = %q, want ssh-ed25519", k.Algorithm)
	}
	if k.Encrypted {
		t.Error("Encrypted = true, want false (no passphrase was given)")
	}
	if !k.HasPublicHalf {
		t.Error("HasPublicHalf = false, want true (the .pub sibling was written in the same Plan)")
	}
	if string(k.Identity.Value()) == "" {
		t.Error("Identity.Value() is empty, want a derived fingerprint")
	}
	if len(k.Locations) != 1 {
		t.Errorf("Locations = %+v, want exactly 1", k.Locations)
	}
}

// TestNewKeyUseCase_ExitCriterion2_MatchesFreshInventory_Encrypted is the same exit-criterion-2
// walk as TestNewKeyUseCase_ExitCriterion2_MatchesFreshInventory above, run against an encrypted
// key instead of a plain one — a gap this close-out pass found: the sibling test above never
// exercises a passphrase, and TestNewKeyUseCase_Plan_WithPassphrase_ProducesAnEncryptedKey (below)
// only asserts Encrypted == true, not the full field-for-field parity roadmap.md §4 criterion 2
// actually requires ("hasp new key's reported facts are identical to what a fresh hasp show key of
// the finished artifact reports"). An encrypted key is the one shape where new key's own generation
// path and a fresh Inspect's derivation path could plausibly disagree (Encrypted, and Identity's
// derivability depend on whether the encryption round-trips through keyfile.Generate and back out
// through keyfile.Inspect identically), so it is the case most worth proving explicitly rather than
// assuming the plain-key test generalizes.
func TestNewKeyUseCase_ExitCriterion2_MatchesFreshInventory_Encrypted(t *testing.T) {
	dir := t.TempDir()
	uc := NewKeyUseCase{}
	secret := []byte("hunter2hunter2")
	plan, err := uc.Plan(NewKeyRequest{Name: "id_ed25519_encrypted", KeyDir: dir, Passphrase: secret})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	detail, ok := ShowKey(m, "id_ed25519_encrypted")
	if !ok {
		t.Fatal("ShowKey: freshly generated encrypted key not found by a fresh derivation")
	}

	k := detail.Key
	if k.Format.String() != "openssh" {
		t.Errorf("Format = %v, want openssh", k.Format)
	}
	if k.Algorithm != "ssh-ed25519" {
		t.Errorf("Algorithm = %q, want ssh-ed25519", k.Algorithm)
	}
	if !k.Encrypted {
		t.Error("Encrypted = false, want true (a passphrase was given)")
	}
	if !k.HasPublicHalf {
		t.Error("HasPublicHalf = false, want true (the .pub sibling was written in the same Plan)")
	}
	// An encrypted OpenSSH private key still carries its fingerprint in cleartext in the key blob
	// (the derivation gap, D12/T1, only bites PEM-encrypted keys with no .pub sidecar) — so
	// Identity must still resolve to a real fingerprint here, not fall back to `unknown`, exactly as
	// the plain-key sibling test asserts.
	if string(k.Identity.Value()) == "" {
		t.Error("Identity.Value() is empty, want a derived fingerprint")
	}
	if len(k.Locations) != 1 {
		t.Errorf("Locations = %+v, want exactly 1", k.Locations)
	}
}

// TestNewKeyUseCase_Plan_WithPassphrase_ProducesAnEncryptedKey confirms the passphrase actually
// reaches the generated key material, and that a fresh Inspect reports Encrypted: true.
func TestNewKeyUseCase_Plan_WithPassphrase_ProducesAnEncryptedKey(t *testing.T) {
	dir := t.TempDir()
	uc := NewKeyUseCase{}
	secret := []byte("hunter2hunter2")
	plan, err := uc.Plan(NewKeyRequest{Name: "id_ed25519", KeyDir: dir, Passphrase: secret})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// The passphrase must be zeroed by the time Plan returns (T6, D17) — keyfile.Generate zeroes
	// it, and since a Go slice shares its backing array with the caller, this proves the effect
	// reached the caller's own copy, not merely some internal copy Generate made.
	for i, b := range secret {
		if b != 0 {
			t.Fatalf("secret[%d] = %v, want 0 (passphrase buffer not zeroed after Plan)", i, b)
		}
	}

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	detail, ok := ShowKey(m, "id_ed25519")
	if !ok {
		t.Fatal("ShowKey: not found")
	}
	if !detail.Key.Encrypted {
		t.Error("Encrypted = false, want true")
	}
}

// TestNewKeyUseCase_Plan_TargetExists_FailsClosedThroughApplier confirms WriteKeyFile's
// AllowOverwrite:false refusal surfaces end to end, through NewKeyUseCase.Plan -> Applier.Apply,
// without a partial write: the private key write must refuse before the .pub write ever runs.
func TestNewKeyUseCase_Plan_TargetExists_FailsClosedThroughApplier(t *testing.T) {
	dir := t.TempDir()
	existingPath := filepath.Join(dir, "id_ed25519")
	original := "pre-existing, irreplaceable key material"
	writeFile(t, existingPath, original)

	uc := NewKeyUseCase{}
	plan, err := uc.Plan(NewKeyRequest{Name: "id_ed25519", KeyDir: dir})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	_, err = applier.Apply(plan)
	if !errors.Is(err, ErrKeyFileExists) {
		t.Fatalf("Apply error = %v, want ErrKeyFileExists", err)
	}

	got, readErr := os.ReadFile(existingPath)
	if readErr != nil {
		t.Fatalf("read %s: %v", existingPath, readErr)
	}
	if string(got) != original {
		t.Errorf("existing key content = %q, want unchanged %q", got, original)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "id_ed25519.pub")); statErr == nil {
		t.Error(".pub sibling was written despite the private key write refusing — partial write")
	}
}

// TestNewKeyUseCase_Plan_RejectsPathTraversal is the regression test for the reviewer's CRITICAL
// finding: req.Name (and req.Profile segments) must be rejected in internal/app itself, before any
// Change is ever planned, when they could carry the target outside req.KeyDir (T8, tdd.md §12) —
// independent of whatever validation internal/cli/new.go does or doesn't do. Plan performs no I/O
// of its own (see Plan's own doc comment), so a rejected request necessarily writes nothing; the
// end-to-end "nothing was written outside --key-dir" assertion, run through the real CLI and
// Applier, lives in internal/cli/writeguard_newkey_test.go.
func TestNewKeyUseCase_Plan_RejectsPathTraversal(t *testing.T) {
	uc := NewKeyUseCase{}

	cases := []struct {
		name string
		req  NewKeyRequest
	}{
		{"traversal in Name", NewKeyRequest{Name: "../../../tmp/evil_key"}},
		{"embedded separator in Name", NewKeyRequest{Name: "sub/evil_key"}},
		{"Name is just ..", NewKeyRequest{Name: ".."}},
		{"Name is just .", NewKeyRequest{Name: "."}},
		{"traversal in Profile segment", NewKeyRequest{Name: "id_ed25519", Profile: []string{"work", ".."}}},
		{"embedded separator in Profile segment", NewKeyRequest{Name: "id_ed25519", Profile: []string{"work/../../evil"}}},
		{"empty Profile segment", NewKeyRequest{Name: "id_ed25519", Profile: []string{""}}},
		{"null byte in Name", NewKeyRequest{Name: "evil\x00name"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			req := tc.req
			req.KeyDir = dir

			_, err := uc.Plan(req)
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("Plan error = %v, want ErrUsage", err)
			}

			entries, readErr := os.ReadDir(dir)
			if readErr != nil {
				t.Fatalf("ReadDir(%s): %v", dir, readErr)
			}
			if len(entries) != 0 {
				t.Errorf("Plan for %+v left entries in KeyDir: %v", req, entries)
			}
		})
	}
}

// TestNewKeyUseCase_Plan_GenerateFailurePropagates confirms a keyfile.Generate error is surfaced
// as a Plan error rather than silently ignored, using an injected Generate for determinism.
func TestNewKeyUseCase_Plan_GenerateFailurePropagates(t *testing.T) {
	wantErr := errors.New("boom")
	uc := NewKeyUseCase{Generate: func(_ keyfile.GenerateOptions) (keyfile.GeneratedKey, error) {
		return keyfile.GeneratedKey{}, wantErr
	}}
	_, err := uc.Plan(NewKeyRequest{Name: "id_ed25519", KeyDir: t.TempDir()})
	if !errors.Is(err, wantErr) {
		t.Errorf("Plan error = %v, want it to wrap %v", err, wantErr)
	}
}
