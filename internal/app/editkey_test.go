package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/backup"
	"github.com/boweeb/hasp/internal/adapter/fswrite"
	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/domain"
)

// realLoc/aliasLoc are small test-only constructors for domain.KeyLocation, used throughout this
// file to build the Key fixtures EditKeyRequest.Key carries.
func realLoc(path string) domain.KeyLocation  { return domain.KeyLocation{Path: path, IsAlias: false} }
func aliasLoc(path string) domain.KeyLocation { return domain.KeyLocation{Path: path, IsAlias: true} }

func TestEditKeyUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	uc := EditKeyUseCase{}
	if _, err := uc.Plan(EditKeyRequest{Key: domain.Key{Name: "id_ed25519"}, NewName: "new"}); !errors.Is(err, ErrUsage) {
		t.Errorf("Plan with empty KeyDir: err = %v, want ErrUsage", err)
	}
	if _, err := uc.Plan(EditKeyRequest{KeyDir: "/tmp", NewName: "new"}); !errors.Is(err, ErrUsage) {
		t.Errorf("Plan with empty Key: err = %v, want ErrUsage", err)
	}
}

func TestEditKeyUseCase_Plan_RejectsNoOperation(t *testing.T) {
	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{KeyDir: "/tmp", Key: domain.Key{Name: "id_ed25519"}})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan with no operation requested: err = %v, want ErrUsage", err)
	}
}

func TestEditKeyUseCase_Plan_RejectsMultipleOperations(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{
		KeyDir:       dir,
		Key:          key,
		NewName:      "renamed",
		AddAliasLeaf: "an_alias",
	})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan with two independent operations requested: err = %v, want ErrUsage", err)
	}
}

// --- rename / move -----------------------------------------------------------------------

func TestEditKeyUseCase_Plan_Rename_SingleMoveFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	plan, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, NewName: "id_ed25519_renamed"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	mv, ok := plan.Changes[0].(MoveFile)
	if !ok {
		t.Fatalf("Changes[0] = %T, want MoveFile", plan.Changes[0])
	}
	want := filepath.Join(dir, "id_ed25519_renamed")
	if mv.From != source || mv.To != want {
		t.Errorf("MoveFile = %+v, want From=%s To=%s", mv, source, want)
	}
}

func TestEditKeyUseCase_Plan_Rename_IncludesPubSidecar(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	writeFile(t, source+".pub", "ssh-ed25519 AAAA... comment\n")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	plan, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, NewName: "id_ed25519_renamed"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2 (private + .pub)", len(plan.Changes))
	}
	pub := plan.Changes[1].(MoveFile)
	if pub.From != source+".pub" || pub.To != filepath.Join(dir, "id_ed25519_renamed.pub") {
		t.Errorf("pub MoveFile = %+v", pub)
	}
}

// TestEditKeyUseCase_Plan_Rename_RefusesWithAlias is the dangling-alias precondition test: a key
// with any alias location must refuse --name/--profile before planning anything.
func TestEditKeyUseCase_Plan_Rename_RefusesWithAlias(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "work", "id_ed25519")
	writeFile(t, source, "key bytes")
	alias := filepath.Join(dir, "id_ed25519")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source), aliasLoc(alias)}}

	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, NewName: "id_ed25519_renamed"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan for a key with an alias: err = %v, want ErrUsage", err)
	}
}

func TestEditKeyUseCase_Plan_Rename_RefusesMultipleRealLocations(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "id_ed25519_a")
	b := filepath.Join(dir, "id_ed25519_b")
	writeFile(t, a, "key bytes")
	writeFile(t, b, "key bytes")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(a), realLoc(b)}}

	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, NewName: "renamed"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan for a key with 2 real locations: err = %v, want ErrUsage", err)
	}
}

func TestEditKeyUseCase_Plan_MoveProfile_RequiresExistingDestination(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, NewProfile: domain.ProfilePath{"work"}})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan --profile into a non-existent directory: err = %v, want ErrUsage", err)
	}
}

// TestEditKeyUseCase_Plan_MoveProfile_UnmanagedDestinationIsLegitimate confirms the Classify-table
// reasoning: --profile is a plain relocation and does not require the destination directory to
// carry a .hasp marker, unlike adopt.
func TestEditKeyUseCase_Plan_MoveProfile_UnmanagedDestinationIsLegitimate(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	unmanagedDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(unmanagedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	plan, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, NewProfile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan --profile into an unmanaged (but existing) directory: %v", err)
	}
	mv := plan.Changes[0].(MoveFile)
	want := filepath.Join(unmanagedDir, "id_ed25519")
	if mv.To != want {
		t.Errorf("To = %s, want %s", mv.To, want)
	}
}

// TestEditKeyUseCase_Plan_NameAndProfileCombine confirms --name and --profile together produce a
// single MoveFile that both renames and relocates in one step.
func TestEditKeyUseCase_Plan_NameAndProfileCombine(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	destDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	plan, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, NewName: "id_ed25519_foobarco", NewProfile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	mv := plan.Changes[0].(MoveFile)
	want := filepath.Join(destDir, "id_ed25519_foobarco")
	if mv.To != want {
		t.Errorf("To = %s, want %s", mv.To, want)
	}
}

func TestEditKeyUseCase_Plan_Rename_RejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	cases := []EditKeyRequest{
		{KeyDir: dir, Key: key, NewName: "../evil"},
		{KeyDir: dir, Key: key, NewName: "sub/evil"},
		{KeyDir: dir, Key: key, NewProfile: domain.ProfilePath{"work", ".."}},
		{KeyDir: dir, Key: key, NewProfile: domain.ProfilePath{"work/../../evil"}},
	}
	for i, req := range cases {
		if _, err := uc.Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}

// --- add-alias -----------------------------------------------------------------------------

func TestEditKeyUseCase_Plan_AddAlias(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "work", "id_ed25519_foobarco")
	writeFile(t, source, "key bytes")
	personalDir := filepath.Join(dir, "personal")
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	key := domain.Key{Name: "id_ed25519_foobarco", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	plan, err := uc.Plan(EditKeyRequest{
		KeyDir:          dir,
		Key:             key,
		AddAliasProfile: domain.ProfilePath{"personal"},
		AddAliasLeaf:    "id_ed25519_foobarco",
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	sym, ok := plan.Changes[0].(CreateSymlink)
	if !ok {
		t.Fatalf("Changes[0] = %T, want CreateSymlink", plan.Changes[0])
	}
	wantPath := filepath.Join(personalDir, "id_ed25519_foobarco")
	if sym.Path != wantPath || sym.Target != source {
		t.Errorf("CreateSymlink = %+v, want Path=%s Target=%s", sym, wantPath, source)
	}
}

func TestEditKeyUseCase_Plan_AddAlias_TopLevel(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "work", "id_rsa")
	writeFile(t, source, "key bytes")
	key := domain.Key{Name: "id_rsa", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	plan, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, AddAliasLeaf: "id_rsa"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	sym := plan.Changes[0].(CreateSymlink)
	if sym.Path != filepath.Join(dir, "id_rsa") {
		t.Errorf("Path = %s, want top-level %s", sym.Path, filepath.Join(dir, "id_rsa"))
	}
}

func TestEditKeyUseCase_Plan_AddAlias_RefusesExistingPath(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "work", "id_ed25519")
	writeFile(t, source, "key bytes")
	existing := filepath.Join(dir, "id_ed25519")
	writeFile(t, existing, "something else already here")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, AddAliasLeaf: "id_ed25519"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan --add-alias onto an existing path: err = %v, want ErrUsage", err)
	}
}

func TestEditKeyUseCase_Plan_AddAlias_RejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	cases := []EditKeyRequest{
		{KeyDir: dir, Key: key, AddAliasLeaf: "../evil"},
		{KeyDir: dir, Key: key, AddAliasProfile: domain.ProfilePath{".."}, AddAliasLeaf: "evil"},
	}
	for i, req := range cases {
		if _, err := uc.Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}

// --- remove-alias --------------------------------------------------------------------------

func TestEditKeyUseCase_Plan_RemoveAlias(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "work", "id_ed25519")
	writeFile(t, source, "key bytes")
	alias := filepath.Join(dir, "personal", "id_ed25519")
	if err := os.MkdirAll(filepath.Dir(alias), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source), aliasLoc(alias)}}

	uc := EditKeyUseCase{}
	plan, err := uc.Plan(EditKeyRequest{
		KeyDir:             dir,
		Key:                key,
		RemoveAliasProfile: domain.ProfilePath{"personal"},
		RemoveAliasLeaf:    "id_ed25519",
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	rm, ok := plan.Changes[0].(Remove)
	if !ok {
		t.Fatalf("Changes[0] = %T, want Remove", plan.Changes[0])
	}
	if rm.Path != alias {
		t.Errorf("Remove.Path = %s, want %s", rm.Path, alias)
	}
}

// TestEditKeyUseCase_Plan_RemoveAlias_NeverRemovesRealFile is the never-destroy-key-material
// guard: --remove-alias against a path that is the key's real (non-alias) location must refuse.
func TestEditKeyUseCase_Plan_RemoveAlias_NeverRemovesRealFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, RemoveAliasLeaf: "id_ed25519"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan --remove-alias against the real file: err = %v, want ErrUsage", err)
	}
}

func TestEditKeyUseCase_Plan_RemoveAlias_RefusesUnknownLocation(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, RemoveAliasLeaf: "not_a_real_location"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan --remove-alias against an unknown location: err = %v, want ErrUsage", err)
	}
}

func TestEditKeyUseCase_Plan_RemoveAlias_RejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(source)}}

	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, RemoveAliasProfile: domain.ProfilePath{".."}, RemoveAliasLeaf: "evil"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// --- replace-material ------------------------------------------------------------------------

// genKeyFile writes a real, parseable ed25519 private key (+ .pub) at path via keyfile.Generate,
// so keyfile.Inspect can actually classify it — a plain "key bytes" string fixture (used
// everywhere else in this file for Move/Symlink/Remove Changes that never parse their target) is
// not sufficient for --replace-material's own validation step.
func genKeyFile(t *testing.T, path string) {
	t.Helper()
	gen, err := keyfile.Generate(keyfile.GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, gen.PrivateKeyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".pub", gen.PublicKeyLine, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEditKeyUseCase_Plan_ReplaceMaterial(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "id_ed25519")
	genKeyFile(t, existing)
	replacement := filepath.Join(t.TempDir(), "new_material")
	genKeyFile(t, replacement)
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(existing)}}

	uc := EditKeyUseCase{}
	plan, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, ReplaceMaterialPath: replacement})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2 (.pub + private)", len(plan.Changes))
	}
	// .pub precedes the private key (review fix, T20 "any prefix leaves a working state"): an
	// interrupted apply between the two steps must never leave new private-key bytes sitting next
	// to a stale .pub that would make keyfile.Inspect report the wrong fingerprint.
	pub, ok := plan.Changes[0].(WriteKeyFile)
	if !ok {
		t.Fatalf("Changes[0] = %T, want WriteKeyFile", plan.Changes[0])
	}
	if pub.Path != existing+".pub" || !pub.AllowOverwrite {
		t.Errorf("Changes[0] = %+v, want Path=%s AllowOverwrite=true", pub, existing+".pub")
	}
	priv, ok := plan.Changes[1].(WriteKeyFile)
	if !ok {
		t.Fatalf("Changes[1] = %T, want WriteKeyFile", plan.Changes[1])
	}
	if priv.Path != existing || !priv.AllowOverwrite {
		t.Errorf("Changes[1] = %+v, want Path=%s AllowOverwrite=true", priv, existing)
	}
	if !priv.RequiresBackup() {
		t.Error("WriteKeyFile.RequiresBackup() = false, want true when AllowOverwrite (T22)")
	}
}

// TestEditKeyUseCase_Plan_ReplaceMaterial_RemovesStalePub confirms the reasoned .pub-sidecar
// decision: replacement material with no .pub of its own must remove the *old* .pub, not leave it
// in place, because a stale .pub would make keyfile.Inspect report the old key's fingerprint
// against the new private key bytes.
func TestEditKeyUseCase_Plan_ReplaceMaterial_RemovesStalePub(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "id_ed25519")
	genKeyFile(t, existing)

	// Replacement has a private key but deliberately no .pub sidecar.
	gen, err := keyfile.Generate(keyfile.GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	replacementDir := t.TempDir()
	replacement := filepath.Join(replacementDir, "new_material")
	if err := os.WriteFile(replacement, gen.PrivateKeyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(existing)}}

	uc := EditKeyUseCase{}
	plan, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, ReplaceMaterialPath: replacement})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2 (stale .pub Remove + private WriteKeyFile)", len(plan.Changes))
	}
	// The stale-.pub Remove precedes the private-key WriteKeyFile — same T20 ordering fix as the
	// sibling test above.
	rm, ok := plan.Changes[0].(Remove)
	if !ok {
		t.Fatalf("Changes[0] = %T, want Remove", plan.Changes[0])
	}
	if rm.Path != existing+".pub" {
		t.Errorf("Remove.Path = %s, want %s", rm.Path, existing+".pub")
	}
	if _, ok := plan.Changes[1].(WriteKeyFile); !ok {
		t.Fatalf("Changes[1] = %T, want WriteKeyFile", plan.Changes[1])
	}
}

func TestEditKeyUseCase_Plan_ReplaceMaterial_RefusesInvalidKeyFile(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "id_ed25519")
	genKeyFile(t, existing)
	notAKey := filepath.Join(t.TempDir(), "not_a_key")
	writeFile(t, notAKey, "this is definitely not PEM-encoded key material")
	key := domain.Key{Name: "id_ed25519", Locations: []domain.KeyLocation{realLoc(existing)}}

	uc := EditKeyUseCase{}
	_, err := uc.Plan(EditKeyRequest{KeyDir: dir, Key: key, ReplaceMaterialPath: notAKey})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan --replace-material with a non-key file: err = %v, want ErrUsage", err)
	}
}

// --- end-to-end (Applier + a fresh Derive) ------------------------------------------------

// TestEditKeyUseCase_Rename_RoundTrip_MatchesFreshInventory renames a key end to end (through the
// real Applier) and confirms a fresh derivation reports it under its new name, with the old name
// no longer resolving — KeyName is the stable handle across a rename (T12).
func TestEditKeyUseCase_Rename_RoundTrip_MatchesFreshInventory(t *testing.T) {
	dir := t.TempDir()
	plan, err := (NewKeyUseCase{}).Plan(NewKeyRequest{Name: "id_ed25519_old", KeyDir: dir})
	if err != nil {
		t.Fatalf("NewKeyUseCase.Plan: %v", err)
	}
	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply (new key): %v", err)
	}

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	detail, ok := ShowKey(m, "id_ed25519_old")
	if !ok {
		t.Fatal("ShowKey: freshly generated key not found")
	}

	editPlan, err := (EditKeyUseCase{}).Plan(EditKeyRequest{KeyDir: dir, Key: detail.Key, NewName: "id_ed25519_new"})
	if err != nil {
		t.Fatalf("EditKeyUseCase.Plan: %v", err)
	}
	if _, err := applier.Apply(editPlan); err != nil {
		t.Fatalf("Apply (rename): %v", err)
	}

	m2, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive after rename: %v", err)
	}
	if _, ok := ShowKey(m2, "id_ed25519_old"); ok {
		t.Error("old name still resolves after rename")
	}
	renamed, ok := ShowKey(m2, "id_ed25519_new")
	if !ok {
		t.Fatal("new name does not resolve after rename")
	}
	if renamed.Key.Identity.Value() != detail.Key.Identity.Value() {
		t.Errorf("fingerprint changed across a rename: got %s, want unchanged %s", renamed.Key.Identity.Value(), detail.Key.Identity.Value())
	}
}

// TestEditKeyUseCase_Rename_UndecidableKey_PathIdentityChangesAsExpected exercises T12's own
// stated consequence directly: renaming an undecidable key's file (identified by canonical path,
// since no fingerprint can be derived) changes its KeyIdentity by definition — the *correct*,
// expected outcome, not a defect — while KeyName (the stable handle the rename was addressed
// through) is what a caller re-resolves the key by afterward.
func TestEditKeyUseCase_Rename_UndecidableKey_PathIdentityChangesAsExpected(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "id_rsa_legacy"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	detail, ok := ShowKey(m, "id_rsa_legacy")
	if !ok {
		t.Fatal("ShowKey: fixture key not found")
	}
	if detail.Key.Identity.Kind() != domain.IdentityPath {
		t.Fatalf("Identity.Kind() = %v, want IdentityPath (this fixture is deliberately undecidable)", detail.Key.Identity.Kind())
	}
	oldIdentity := detail.Key.Identity.Value()

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	editPlan, err := (EditKeyUseCase{}).Plan(EditKeyRequest{KeyDir: dir, Key: detail.Key, NewName: "id_rsa_renamed"})
	if err != nil {
		t.Fatalf("EditKeyUseCase.Plan: %v", err)
	}
	if _, err := applier.Apply(editPlan); err != nil {
		t.Fatalf("Apply (rename): %v", err)
	}

	m2, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive after rename: %v", err)
	}
	renamed, ok := ShowKey(m2, "id_rsa_renamed")
	if !ok {
		t.Fatal("ShowKey: renamed key not found under its new KeyName")
	}
	if renamed.Key.Identity.Kind() != domain.IdentityPath {
		t.Fatalf("Identity.Kind() after rename = %v, want IdentityPath", renamed.Key.Identity.Kind())
	}
	if renamed.Key.Identity.Value() == oldIdentity {
		t.Errorf("path identity unchanged after rename: still %s, want a new resolved path (T12's stated consequence)", oldIdentity)
	}
}

// TestEditKeyUseCase_AddAlias_RoundTrip_UnionsProfiles is this phase's proof of T19's actual
// mechanism, not just that a symlink got created: after --add-alias, a fresh app.Derive scan must
// report the key's Profiles as the union over all of its Locations (D2's write path).
func TestEditKeyUseCase_AddAlias_RoundTrip_UnionsProfiles(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "work")
	mkManagedProfileDir(t, workDir)
	personalDir := filepath.Join(dir, "personal")
	mkManagedProfileDir(t, personalDir)

	plan, err := (NewKeyUseCase{}).Plan(NewKeyRequest{Name: "id_ed25519_foobarco", KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("NewKeyUseCase.Plan: %v", err)
	}
	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply (new key): %v", err)
	}

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	detail, ok := ShowKey(m, "id_ed25519_foobarco")
	if !ok {
		t.Fatal("ShowKey: key not found")
	}
	if len(detail.Key.Profiles) != 1 || !detail.Key.Profiles[0].Equal(domain.ProfilePath{"work"}) {
		t.Fatalf("Profiles before add-alias = %+v, want just [work]", detail.Key.Profiles)
	}

	editPlan, err := (EditKeyUseCase{}).Plan(EditKeyRequest{
		KeyDir:          dir,
		Key:             detail.Key,
		AddAliasProfile: domain.ProfilePath{"personal"},
		AddAliasLeaf:    "id_ed25519_foobarco",
	})
	if err != nil {
		t.Fatalf("EditKeyUseCase.Plan: %v", err)
	}
	if _, err := applier.Apply(editPlan); err != nil {
		t.Fatalf("Apply (add-alias): %v", err)
	}

	m2, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive after add-alias: %v", err)
	}
	detail2, ok := ShowKey(m2, "id_ed25519_foobarco")
	if !ok {
		t.Fatal("ShowKey: key not found after add-alias")
	}
	if len(detail2.Key.Profiles) != 2 {
		t.Fatalf("Profiles after add-alias = %+v, want 2 (work, personal)", detail2.Key.Profiles)
	}
	var haveWork, havePersonal bool
	for _, p := range detail2.Key.Profiles {
		if p.Equal(domain.ProfilePath{"work"}) {
			haveWork = true
		}
		if p.Equal(domain.ProfilePath{"personal"}) {
			havePersonal = true
		}
	}
	if !haveWork || !havePersonal {
		t.Errorf("Profiles = %+v, want the union [work personal]", detail2.Key.Profiles)
	}
}

// TestEditKeyUseCase_ReplaceMaterial_RoundTrip_BackupCapturesOldMaterialAndFingerprintChanges
// confirms both halves of --replace-material's own safety claim: the backup store holds the *old*
// bytes (readable from .hasp-backups/), and a fresh derivation reports the key's fingerprint (and
// algorithm, where relevant) genuinely changed to match the new material — not just that bytes on
// disk changed.
func TestEditKeyUseCase_ReplaceMaterial_RoundTrip_BackupCapturesOldMaterialAndFingerprintChanges(t *testing.T) {
	dir := t.TempDir()
	plan, err := (NewKeyUseCase{}).Plan(NewKeyRequest{Name: "id_ed25519", KeyDir: dir})
	if err != nil {
		t.Fatalf("NewKeyUseCase.Plan: %v", err)
	}
	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply (new key): %v", err)
	}

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	detail, ok := ShowKey(m, "id_ed25519")
	if !ok {
		t.Fatal("ShowKey: key not found")
	}
	oldFingerprint := detail.Key.Identity.Value()
	oldPrivatePath := filepath.Join(dir, "id_ed25519")
	oldBytes, err := os.ReadFile(oldPrivatePath)
	if err != nil {
		t.Fatal(err)
	}

	replacement := filepath.Join(t.TempDir(), "new_material")
	genKeyFile(t, replacement)
	newInfo, err := keyfile.Inspect(replacement)
	if err != nil {
		t.Fatal(err)
	}

	editPlan, err := (EditKeyUseCase{}).Plan(EditKeyRequest{KeyDir: dir, Key: detail.Key, ReplaceMaterialPath: replacement})
	if err != nil {
		t.Fatalf("EditKeyUseCase.Plan: %v", err)
	}
	if _, err := applier.Apply(editPlan); err != nil {
		t.Fatalf("Apply (replace-material): %v", err)
	}

	// The backup store holds the old material, readable and byte-identical to what was there
	// before.
	backupDir := filepath.Join(dir, ".hasp-backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("read %s: %v", backupDir, err)
	}
	var foundOldBackup bool
	for _, e := range entries {
		b, readErr := os.ReadFile(filepath.Join(backupDir, e.Name()))
		if readErr != nil {
			continue
		}
		if string(b) == string(oldBytes) {
			foundOldBackup = true
			break
		}
	}
	if !foundOldBackup {
		t.Errorf("no backup entry under %s matched the pre-replace material", backupDir)
	}

	m2, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive after replace-material: %v", err)
	}
	detail2, ok := ShowKey(m2, "id_ed25519")
	if !ok {
		t.Fatal("ShowKey: key not found after replace-material")
	}
	if detail2.Key.Identity.Value() == oldFingerprint {
		t.Errorf("fingerprint unchanged after replace-material: still %s", oldFingerprint)
	}
	if detail2.Key.Identity.Value() != string(newInfo.Fingerprint) {
		t.Errorf("fingerprint = %s, want the replacement material's own fingerprint %s", detail2.Key.Identity.Value(), newInfo.Fingerprint)
	}
}
