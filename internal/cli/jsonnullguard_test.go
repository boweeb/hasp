package cli_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// jsonNullGuardArrayFields is every array-typed JSON field name reachable from any --json kind's
// data payload, across the whole current compatibility surface (T52/T53): domain.Key.Locations
// ("locations"), domain.Key.Profiles/domain.Host.Profiles ("profiles"), domain.Host.Patterns
// ("patterns"), domain.Host.Bindings ("bindings"), domain.Origin.Because ("because"),
// app.InvestigatedKey.Schemes ("schemes"), app.InvestigatedKey.Origins/app.FindMatch.Origins
// ("origins"), app.ProfileDetail.Keys ("keys"), app.KeyDetail.Hosts/app.ProfileDetail.Hosts
// ("hosts"), app.Finding.Detail's one documented map-value shape, {"paths": [...]} on a
// duplicate-key finding ("paths", revision cycle 1, verifier finding 2 — see the check.report
// subtest below and T53), and render.Envelope.Data itself ("data"). Some of these (locations,
// patterns, schemes) are never actually empty in practice given how hasp derives them — included
// anyway, both for completeness against a future change to that invariant and because the cost of
// checking one more literal substring is zero.
//
// This is deliberately a fixed, named list, not a generic "scan every null in the JSON" walk:
// distinguishing an array-typed null (a bug, T14) from a legitimate one (a nil pointer, a nil
// `any`, or a nil map — all correctly left as JSON null by normalizeNilSlices, T52) from raw
// bytes alone requires exactly the type information this list encodes by hand once. Checking
// raw bytes at all, rather than decoding through encoding/json, is the point (see
// assertNoArrayTypedNull's own doc comment) — a byte-level literal-substring check on a known
// field-name set is the direct, auditable way to do that.
var jsonNullGuardArrayFields = []string{
	"data", "profiles", "hosts", "bindings", "origins", "because", "schemes", "keys", "locations", "patterns", "paths",
}

// assertNoArrayTypedNull fails t if raw contains an array-typed field rendered as JSON null
// (`"field": null`, the exact spacing render.JSON's indented encoder produces) for any field in
// jsonNullGuardArrayFields. It operates on raw bytes rather than decoding raw back into Go
// values, deliberately: encoding/json decodes both a JSON "null" and a JSON "[]" into the same
// nil/zero-length Go slice, so a round trip through json.Unmarshal cannot distinguish the two —
// exactly the defect T52 exists to catch, and exactly what internal/cli/defaultread_golden_test.go's
// TestOriginsForInvestigate_NonRSA_ReturnsIdiomaticNil-adjacent tests already knew, one call site
// at a time, before T52 made the guarantee general.
func assertNoArrayTypedNull(t *testing.T, kind string, raw []byte) {
	t.Helper()
	for _, field := range jsonNullGuardArrayFields {
		needle := []byte(`"` + field + `": null`)
		if bytes.Contains(raw, needle) {
			t.Errorf("%s: raw output contains %s (array-typed null, T14/T52 violation):\n%s", kind, needle, raw)
		}
	}
}

// TestJSONKinds_NeverEmitArrayTypedNull is T52's mechanical end-to-end guard: every --json kind
// hasp exposes is run here, against a fixture deliberately built so every array-typed field a
// kind's own payload type can carry comes back genuinely empty — the exact shape (a key with no
// profile, a host with no binding, a profile with no members, a non-RSA key's investigate-mode
// origins, a duplicate key's paths list) that used to surface as a literal JSON null before T52's
// recursive normalizeNilSlices existed. Every kind tdd.md §9's command grid exposes under --json
// is enumerated here for completeness — key.list, key.show, key.find, host.list, host.show,
// host.find, profile.list, profile.show, profile.find, check.report, and both --investigate
// variants tdd.md §18 adds on top of key.list/key.show — but enumeration is not the same claim as
// coverage: profile.list and profile.find are **structural no-ops** for this regression class
// (revision cycle 1, verifier finding 2, recorded permanently at T53, docs/tech-decision-log.md).
// app.ProfileSummary and domain.Profile (internal/app/profile.go, internal/domain/profile.go)
// carry no nested array-typed field at all — the only array either kind's payload can ever carry
// is the top-level `data` envelope array itself, which was already covered by marshalData's
// pre-T52 top-level-only special case, so neither subtest can fail for a defect normalizeNilSlices
// exists to catch; they run anyway so a future field added to either type is caught the moment it
// starts carrying a nested array, not silently. Every other kind below, including check.report's
// map-nested app.Finding.Detail["paths"] case, does exercise a genuine T52-class regression.
//
// The fixture, once, shared by every subtest below:
//   - customEd25519FixtureKeyPEM (this file's own fixture, byte-identical to
//     defaultread_golden_test.go's ed25519FixtureKeyPEM) is written under a filename
//     ("custom_ed25519") deliberately absent from internal/app.implicitDefaultNames (T16), so it
//     is never picked up by ssh's own implicit default-identity probing — the one thing that
//     would otherwise force a non-empty Bindings on the "bare" host stanza below regardless of
//     what the config file says.
//   - The key sits at the key directory's top level, under no ".hasp"-marked profile directory,
//     so domain.Key.Profiles is nil at the source.
//   - A second, independent copy of the same key bytes (custom_ed25519_dup, see below) triggers
//     check_key.go's detectDuplicateKeyConfirmed, so check.report's Detail["paths"] map-nested
//     slice case is actually exercised, not merely theoretically in scope (revision cycle 1,
//     verifier finding 2).
//   - The "bare" Host stanza carries no IdentityFile line and (given the point above) matches no
//     implicit default identity either, so domain.Host.Bindings, and therefore
//     domain.Host.Profiles (T5, derived from Bindings), are both nil at the source.
//   - The "lonely" profile directory carries a ".hasp" marker and nothing else — no key, no host
//     stanza attributes to it — so app.ProfileDetail.Keys and .Hosts are both nil at the source.
//
// Every one of those is the ordinary, idiomatic Go zero value for "no results" a real, empty
// ~/.ssh corner can produce validly (an unaffiliated key, an unbound host stanza, an empty
// profile directory, a duplicate key with a real paths list) — this test's whole point is that
// none of them may ever reach the wire as a literal JSON null.
func TestJSONKinds_NeverEmitArrayTypedNull(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "custom_ed25519"), customEd25519FixtureKeyPEM)

	// A second, independent real file carrying the exact same key bytes as custom_ed25519 —
	// deliberately, so deriveKeys (internal/app/pipeline.go) groups the two by fingerprint into
	// one domain.Key with two non-alias Locations, which is check_key.go's own trigger for
	// detectDuplicateKeyConfirmed ("two or more non-alias Locations"). This is the fixture's one
	// addition for revision cycle 1, verifier finding 2: without it, no finding in this fixture
	// ever populated a map[string]any value with a slice (Finding.Detail's one documented
	// "paths" shape, T52's own cited proof that the map-values-are-walked guarantee matters), so
	// check.report's subtest exercised nothing beyond the top-level "data" array every other
	// subtest here already covers structurally. "custom_ed25519" sorts before
	// "custom_ed25519_dup" lexicographically (a prefix of a longer string always sorts first), so
	// projectKey's tie-break still picks "custom_ed25519" as the primary location — key.show's
	// subtest below is unaffected by this addition.
	writeFile(t, filepath.Join(dir, "custom_ed25519_dup"), customEd25519FixtureKeyPEM)

	writeFile(t, filepath.Join(dir, "lonely", ".hasp"), "# empty managed profile, deliberately unaffiliated\n")

	writeFile(t, filepath.Join(dir, "config"), ""+
		"Host bare\n"+
		"    HostName bare.example.com\n", // no IdentityFile: T16 implicit-default probing finds nothing here either
	)

	run := func(t *testing.T, kind string, args ...string) []byte {
		t.Helper()
		var out bytes.Buffer
		root := cli.NewRootCmd()
		root.SetArgs(append(append([]string{}, args...), "--key-dir", dir, "--json"))
		root.SetOut(&out)
		root.SetErr(&out)
		if err := root.Execute(); err != nil {
			t.Fatalf("hasp %v: %v\n%s", args, err, out.String())
		}
		raw := out.Bytes()
		assertNoArrayTypedNull(t, kind, raw)
		return raw
	}

	t.Run("key.list", func(t *testing.T) {
		raw := run(t, "key.list", "list", "key")
		if !bytes.Contains(raw, []byte(`"profiles": []`)) {
			t.Errorf("key.list: want \"profiles\": [] present at all: %s", raw)
		}
	})

	t.Run("key.list_investigate", func(t *testing.T) {
		raw := run(t, "key.list (--investigate)", "list", "key", "--investigate")
		if !bytes.Contains(raw, []byte(`"origins": []`)) {
			t.Errorf("key.list --investigate: want \"origins\": [] present (non-RSA key): %s", raw)
		}
	})

	t.Run("key.show", func(t *testing.T) {
		raw := run(t, "key.show", "show", "key", "custom_ed25519")
		if !bytes.Contains(raw, []byte(`"hosts": []`)) {
			t.Errorf("key.show: want \"hosts\": [] present at all: %s", raw)
		}
	})

	t.Run("key.show_investigate", func(t *testing.T) {
		raw := run(t, "key.show (--investigate)", "show", "key", "custom_ed25519", "--investigate")
		if !bytes.Contains(raw, []byte(`"origins": []`)) {
			t.Errorf("show key --investigate: want \"origins\": [] present (non-RSA key): %s", raw)
		}
	})

	t.Run("key.find", func(t *testing.T) {
		// The real fingerprint of customEd25519FixtureKeyPEM is
		// SHA256:t4rn6Go9uzGHCyff1lwYWtX+swf+EQhuqmE2dHSqkFE (defaultread_golden_test.go's own
		// comment names it, since it is the same key bytes) — mangled here exactly as
		// testdata/script/key-find.txtar already does, T50's case/punctuation fold.
		raw := run(t, "key.find", "find", "key", "sha256:T4RN-6Go9-uzGH")
		if !bytes.Contains(raw, []byte(`"profiles": []`)) {
			t.Errorf("key.find: want \"profiles\": [] present at all: %s", raw)
		}
	})

	t.Run("host.list", func(t *testing.T) {
		raw := run(t, "host.list", "list", "host")
		if !bytes.Contains(raw, []byte(`"bindings": []`)) {
			t.Errorf("host.list: want \"bindings\": [] present at all: %s", raw)
		}
	})

	t.Run("host.show", func(t *testing.T) {
		raw := run(t, "host.show", "show", "host", "bare")
		if !bytes.Contains(raw, []byte(`"bindings": []`)) {
			t.Errorf("host.show: want \"bindings\": [] present at all: %s", raw)
		}
	})

	t.Run("host.find", func(t *testing.T) {
		raw := run(t, "host.find", "find", "host", "bare")
		if !bytes.Contains(raw, []byte(`"bindings": []`)) {
			t.Errorf("host.find: want \"bindings\": [] present at all: %s", raw)
		}
	})

	t.Run("profile.list", func(t *testing.T) {
		// Structural no-op (revision cycle 1, verifier finding 2; T53): app.ProfileSummary
		// carries no nested array-typed field, so this subtest cannot fail for a T52-class
		// defect — run anyway so a future field added to ProfileSummary is caught the moment it
		// starts carrying a nested array, not silently.
		run(t, "profile.list", "list", "profile")
	})

	t.Run("profile.show", func(t *testing.T) {
		raw := run(t, "profile.show", "show", "profile", "lonely")
		if !bytes.Contains(raw, []byte(`"keys": []`)) {
			t.Errorf("profile.show: want \"keys\": [] present at all: %s", raw)
		}
		if !bytes.Contains(raw, []byte(`"hosts": []`)) {
			t.Errorf("profile.show: want \"hosts\": [] present at all: %s", raw)
		}
	})

	t.Run("profile.find", func(t *testing.T) {
		// Structural no-op, same reason as profile.list above: domain.Profile carries no
		// nested array-typed field either (revision cycle 1, verifier finding 2; T53).
		run(t, "profile.find", "find", "profile", "lone")
	})

	t.Run("check.report", func(t *testing.T) {
		var out bytes.Buffer
		root := cli.NewRootCmd()
		root.SetArgs([]string{"check", "--key-dir", dir, "--json"})
		root.SetOut(&out)
		root.SetErr(&out)
		_ = root.Execute() // check exits 1 when findings exist (T14) — a non-nil error here is expected, not a test failure
		raw := out.Bytes()
		assertNoArrayTypedNull(t, "check.report", raw)
		// Fixture-specific positive assertion (revision cycle 1, verifier finding 2): unlike
		// every other subtest above, this one previously had none — assertNoArrayTypedNull alone
		// cannot prove the fixture ever produced app.Finding.Detail["paths"] at all, only that it
		// isn't null when present. custom_ed25519/custom_ed25519_dup's shared fingerprint raises
		// detectDuplicateKeyConfirmed (internal/app/check_key.go), whose Detail carries a real,
		// non-empty "paths" list naming both files — asserted here so this subtest can actually
		// fail if that map-nested slice ever stops reaching the wire, structurally, not just by
		// accident of assertNoArrayTypedNull finding nothing to complain about.
		if !bytes.Contains(raw, []byte(`"paths": [`)) {
			t.Errorf("check.report: want \"paths\": [...] present (duplicate-key-confirmed finding's map-nested slice): %s", raw)
		}
		if !bytes.Contains(raw, []byte("custom_ed25519_dup")) {
			t.Errorf("check.report: want the duplicate key's second location present inside \"paths\": %s", raw)
		}
	})
}

// customEd25519FixtureKeyPEM is byte-identical to defaultread_golden_test.go's own
// ed25519FixtureKeyPEM — the same committed, throwaway inline key — written under a different
// variable name here only because this file writes it under a non-default filename
// ("custom_ed25519") for T16 implicit-default-probing reasons this file's own doc comment
// explains; the key bytes, and therefore its fingerprint, are identical either way.
const customEd25519FixtureKeyPEM = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACAEEd2lOVWo2Yf1SBuQBQ6kipAJm/tP0lwuxYKzGZ0nYgAAAIguUITVLlCE
1QAAAAtzc2gtZWQyNTUxOQAAACAEEd2lOVWo2Yf1SBuQBQ6kipAJm/tP0lwuxYKzGZ0nYg
AAAEDeq64CYCWqT5OaGWlM4yFGnPx2Oi600gc1LeQFjUfbFQQR3aU5VajZh/VIG5AFDqSK
kAmb+0/SXC7FgrMZnSdiAAAAAAECAwQF
-----END OPENSSH PRIVATE KEY-----
`
