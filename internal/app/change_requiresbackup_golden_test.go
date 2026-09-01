package app

import "testing"

// TestChangeKinds_RequiresBackup_GoldenList is roadmap.md §4 exit criterion 4's milestone-level
// proof: P4 ("before hasp modifies anything it did not create, the prior state is preserved") in
// one place, walked against every concrete Change kind this codebase currently defines, rather
// than trusting each kind's own scattered per-file test (change_movefile_test.go,
// change_writeregion_test.go, change_symlink_test.go, change_marker_test.go, change_remove_test.go,
// change_writekeyfile_test.go, change_replacewithsymlink_test.go,
// change_replacesymlinkwithfile_test.go — still present, and each independently correct) to add up
// to the whole rule. TestApplier_BacksUpOnlyWhenRequired (plan_test.go) already proves, generically
// via a stubChange, that Applier.Apply calls Backups.Snapshot before Change.Apply whenever
// RequiresBackup() is true and never otherwise — this test does not duplicate that wiring proof, it
// proves each real Change kind's RequiresBackup() answer is the P4-correct one to feed that wiring.
//
// **Judgment call, stated explicitly:** this table is necessarily a manually maintained
// enumeration, not a reflection-driven one — Go has no supported way to enumerate every type in a
// package implementing an interface the way reflect.TypeOf(Settings{}) enumerates a struct's own
// fields for the settings field-set golden test (settings_test.go). A Change kind added without a
// corresponding row here will still compile and pass every other test; the discipline this test
// buys is that every kind *currently* known gets its P4 answer checked together, in one place,
// against the rule stated once, rather than only ever asserted next to each kind's own
// implementation. Extending this table is the expected way a future Change kind stays honest.
// Nine rows cover the eight concrete Change kinds: WriteKeyFile appears twice, once for each of
// its two P4-relevant AllowOverwrite states.
func TestChangeKinds_RequiresBackup_GoldenList(t *testing.T) {
	cases := []struct {
		name   string
		change Change
		want   bool
		reason string
	}{
		{
			name:   "MoveFile",
			change: MoveFile{From: "/a", To: "/b", Reason: "test"},
			want:   true,
			reason: "From is a file hasp did not just create in this Plan (P4)",
		},
		{
			name:   "WriteRegion",
			change: WriteRegion{File: "/a", Marker: "test", Before: []byte("before"), After: []byte("after")},
			want:   true,
			reason: "rewrites existing bytes of a file hasp did not necessarily create outright (D7's co-owned config case)",
		},
		{
			name:   "CreateSymlink",
			change: CreateSymlink{Path: "/a", Target: "/b"},
			want:   false,
			reason: "a pure creation; nothing existed at Path before it",
		},
		{
			name:   "CreateMarker",
			change: CreateMarker{Dir: "/a", Header: []byte("# managed by hasp\n")},
			want:   false,
			reason: "a pure creation; nothing existed at the marker's path before it",
		},
		{
			name:   "Remove",
			change: Remove{Path: "/a", Reason: "test"},
			want:   true,
			reason: "any deletion must be recoverable (T4: always true)",
		},
		{
			name:   "WriteKeyFile/AllowOverwrite=false (new key: fresh generation)",
			change: WriteKeyFile{Path: "/a", Contents: []byte("x"), Mode: 0o600, AllowOverwrite: false},
			want:   false,
			reason: "new key never overwrites; nothing existed at Path before a fresh generation",
		},
		{
			name:   "WriteKeyFile/AllowOverwrite=true (edit key --replace-material)",
			change: WriteKeyFile{Path: "/a", Contents: []byte("x"), Mode: 0o600, AllowOverwrite: true},
			want:   true,
			reason: "overwrites key material hasp did not just create in this Plan (T22)",
		},
		{
			name:   "ReplaceWithSymlink",
			change: ReplaceWithSymlink{From: "/a", To: "/b"},
			want:   true,
			reason: "From is a file hasp did not just create in this Plan (D4, P4, adopt key)",
		},
		{
			name:   "ReplaceSymlinkWithFile",
			change: ReplaceSymlinkWithFile{Alias: "/a", Source: "/b"},
			want:   true,
			reason: "the data reachable through Alias is snapshotted before it is replaced (D4, P4, release key)",
		},
	}

	if len(cases) != 9 {
		t.Fatalf("golden list has %d entries, want 9 — internal/app/change_*.go defines exactly 8 concrete Change kinds as of M2 (MoveFile, WriteRegion, CreateSymlink, CreateMarker, Remove, WriteKeyFile, ReplaceWithSymlink, ReplaceSymlinkWithFile), and WriteKeyFile appears twice here since its P4 answer is conditional on AllowOverwrite; a mismatch here means either this table or the comment above it has drifted from the real Change-kind count and both need reconciling by hand", len(cases))
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.change.RequiresBackup(); got != tc.want {
				t.Errorf("%T.RequiresBackup() = %v, want %v (P4: %s)", tc.change, got, tc.want, tc.reason)
			}
		})
	}
}
