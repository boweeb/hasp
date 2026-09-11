package cli_test

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/boweeb/hasp/internal/adapter/fpscheme"
	"github.com/boweeb/hasp/internal/cli"
	"github.com/boweeb/hasp/internal/cli/render"
	"github.com/boweeb/hasp/internal/domain"
)

// testPassphrase mirrors tools/genfixtures/main.go's own constant of the same name and value —
// duplicated per-package, as internal/adapter/keyfile's, internal/adapter/sshagent's, and
// internal/app's own test suites already do, since tools/genfixtures is a `main` package and
// cannot be imported.
const testPassphrase = "hasp-test-fixture-passphrase"

// investigatedKeyPartial and keyDetailPartial decode only the --investigate-specific fields out
// of a key.show/key.list envelope's `data`, deliberately omitting domain.Key's own fields
// (identity, locations, ...): domain.KeyIdentity is an interface with no UnmarshalJSON (only its
// concrete, unexported implementations carry MarshalJSON, internal/domain/identity.go), so
// encoding/json cannot decode straight into app.InvestigatedKeyDetail from outside internal/domain
// — encoding/json silently ignores JSON object keys with no matching Go field, so declaring only
// the fields these tests need is sufficient and does not mask a missing "schemes"/"origins"/
// "commentSource" key.
type investigatedKeyPartial struct {
	Schemes       []domain.SchemeFingerprint `json:"schemes"`
	Origins       []domain.Origin            `json:"origins"`
	AgentComment  string                     `json:"agentComment"`
	CommentSource domain.FactSource          `json:"commentSource"`
}

type keyDetailPartial struct {
	Key investigatedKeyPartial `json:"key"`
}

// runInvestigateJSON runs `hasp <args...> --investigate --json --key-dir keyDir` and decodes the
// resulting envelope, failing the test on any non-zero exit or decode error.
func runInvestigateJSON(t *testing.T, keyDir string, args ...string) render.Envelope {
	t.Helper()
	var out bytes.Buffer
	root := cli.NewRootCmd()
	full := append(append([]string{}, args...), "--investigate", "--json", "--key-dir", keyDir)
	root.SetArgs(full)
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatalf("hasp %v: %v\n%s", full, err, out.String())
	}
	var env render.Envelope
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v\n%s", err, out.String())
	}
	return env
}

// TestInvestigate_Criterion2_EveryRegisteredSchemeAppears is roadmap.md §5.6 exit criterion 2:
// `hasp show key --investigate --json` emits every registered scheme's value or an explicit
// unknown with a machine-readable reason; no scheme is silently omitted. Asserted against
// len(fpscheme.Schemes()) so a fifth scheme added to the registry without a corresponding output
// change would fail this test rather than pass silently.
func TestInvestigate_Criterion2_EveryRegisteredSchemeAppears(t *testing.T) {
	fixturesDir := repoFixturesDir(t)
	dir := t.TempDir()
	copyFixtureInto(t, fixturesDir, "rsa-pem-plain-pub", filepath.Join(dir, "id_rsa"))
	copyFixtureInto(t, fixturesDir, "rsa-pem-plain-pub.pub", filepath.Join(dir, "id_rsa.pub"))

	env := runInvestigateJSON(t, dir, "show", "key", "id_rsa")
	if env.Kind != "key.show" {
		t.Fatalf("kind = %q, want key.show", env.Kind)
	}

	var detail keyDetailPartial
	if err := json.Unmarshal(env.Data, &detail); err != nil {
		t.Fatalf("decode data: %v", err)
	}

	wantCount := len(fpscheme.Schemes())
	if len(detail.Key.Schemes) != wantCount {
		t.Fatalf("len(Schemes) = %d, want %d (one per registered scheme, roadmap.md §5.6 criterion 2)", len(detail.Key.Schemes), wantCount)
	}
	for _, sf := range detail.Key.Schemes {
		if sf.Value == "" && (sf.Confidence != domain.ConfidenceUnknown || sf.Reason == "") {
			t.Errorf("scheme %s: neither a non-empty value nor (confidence unknown + a non-empty reason): %+v", sf.Scheme, sf)
		}
	}
}

// TestInvestigate_Criterion3_ConfidenceAndCommentSourceInClosedSets is roadmap.md §5.6 exit
// criterion 3, walked structurally over the wire JSON — decode and inspect, never regex over
// text — so a fifth confidence value or a third fact source, however deeply nested, is caught
// regardless of which field carries it.
func TestInvestigate_Criterion3_ConfidenceAndCommentSourceInClosedSets(t *testing.T) {
	fixturesDir := repoFixturesDir(t)
	dir := t.TempDir()
	copyFixtureInto(t, fixturesDir, "rsa-pem-plain-pub", filepath.Join(dir, "id_rsa"))
	copyFixtureInto(t, fixturesDir, "rsa-pem-plain-pub.pub", filepath.Join(dir, "id_rsa.pub"))
	copyFixtureInto(t, fixturesDir, "ed25519-openssh-plain-pub", filepath.Join(dir, "id_ed25519"))
	copyFixtureInto(t, fixturesDir, "ed25519-openssh-plain-pub.pub", filepath.Join(dir, "id_ed25519.pub"))
	copyFixtureInto(t, fixturesDir, "rsa-openssh-encrypted-pub", filepath.Join(dir, "id_rsa_encrypted"))
	copyFixtureInto(t, fixturesDir, "rsa-openssh-encrypted-pub.pub", filepath.Join(dir, "id_rsa_encrypted.pub"))
	copyFixtureInto(t, fixturesDir, "ed25519-openssh-encrypted-nopub", filepath.Join(dir, "id_ed25519_agent"))

	// A real fake agent, loaded with id_ed25519_agent's own decrypted key, so this walk actually
	// observes a "commentSource": "agent-sourced" value at least once — otherwise this test would
	// only ever exercise commentSource's *absent* (FactSourcePlainRead) case.
	priv := decryptedFixturePrivateKey(t, filepath.Join(dir, "id_ed25519_agent"))
	keyring := agent.NewKeyring()
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv, Comment: "hasp-test-criterion3-comment"}); err != nil {
		t.Fatalf("keyring.Add: %v", err)
	}
	t.Setenv("SSH_AUTH_SOCK", pipedAgentSocket(t, keyring))

	env := runInvestigateJSON(t, dir, "list", "key")
	if env.Kind != "key.list" {
		t.Fatalf("kind = %q, want key.list", env.Kind)
	}

	var tree any
	if err := json.Unmarshal(env.Data, &tree); err != nil {
		t.Fatalf("decode data: %v", err)
	}

	confidences := map[string]bool{}
	for _, c := range domain.AllConfidences() {
		confidences[c.String()] = true
	}
	factSources := map[string]bool{}
	for _, s := range domain.AllFactSources() {
		factSources[string(s)] = true
	}
	// FactSourcePlainRead's own wire value is "" and is omitted from JSON entirely
	// (`commentSource,omitempty`), so an *absent* commentSource key is exactly as valid as a
	// present one equal to "agent-sourced" — this walk only ever checks keys that are present.
	factSources[""] = true

	var seenConfidence, seenCommentSource int
	walkJSON(tree, func(key string, value any) {
		switch key {
		case "confidence":
			s, ok := value.(string)
			if !ok {
				t.Fatalf("confidence value %v is not a string", value)
			}
			seenConfidence++
			if !confidences[s] {
				t.Errorf("confidence = %q, not in domain.AllConfidences()", s)
			}
		case "commentSource":
			s, ok := value.(string)
			if !ok {
				t.Fatalf("commentSource value %v is not a string", value)
			}
			seenCommentSource++
			if !factSources[s] {
				t.Errorf("commentSource = %q, not in domain.AllFactSources()", s)
			}
		}
	})
	if seenConfidence == 0 {
		t.Error("walked the whole --investigate JSON output and found zero \"confidence\" keys — this test proves nothing")
	}
	if seenCommentSource == 0 {
		t.Error("walked the whole --investigate JSON output and found zero \"commentSource\" keys — the agent fixture should have produced at least one")
	}
}

// walkJSON recursively visits every key/value pair in a decoded JSON tree (the shape
// encoding/json produces into an `any`: map[string]any, []any, or a scalar), invoking visit for
// every object key found at any depth.
func walkJSON(v any, visit func(key string, value any)) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			visit(k, val)
			walkJSON(val, visit)
		}
	case []any:
		for _, item := range t {
			walkJSON(item, visit)
		}
	}
}

// pipedAgentSocket starts a real fake ssh-agent (agent.NewKeyring + agent.ServeAgent) listening
// on a short-lived, real Unix socket and returns its path. os.MkdirTemp("", "hasp") is used
// rather than t.TempDir() deliberately: t.TempDir() nests under the test's own (long) name and
// can push the socket path past AF_UNIX's ~108-byte cap on Linux, which internal/adapter/sshagent
// and internal/app's own equivalent fixtures already avoid the same way. The test using this
// helper skips, rather than fails, if the resulting path still exceeds the cap.
func pipedAgentSocket(t *testing.T, keyring agent.Agent) string {
	t.Helper()
	socketDir, err := os.MkdirTemp("", "hasp")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketDir) })

	sockPath := filepath.Join(socketDir, "agent.sock")
	const maxUnixSocketPath = 100 // conservative margin under Linux's ~108-byte AF_UNIX cap
	if len(sockPath) > maxUnixSocketPath {
		t.Skipf("socket path %q (%d bytes) is too close to AF_UNIX's ~108-byte cap on this system; skipping", sockPath, len(sockPath))
	}

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("net.Listen(unix, %s): %v", sockPath, err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { _ = agent.ServeAgent(keyring, conn) }()
		}
	}()
	return sockPath
}

// TestInvestigate_Criterion4_AgentSourcedComment_RealFakeAgent is roadmap.md §5.6 exit criterion
// 4's positive case: with a key loaded into a real fake ssh-agent reached over SSH_AUTH_SOCK, the
// comment of an OpenSSH-format **encrypted** key is reported and labelled agent-sourced. The
// fixture (testdata/keys/ed25519-openssh-encrypted-nopub) carries an empty comment by
// construction (genfixtures used `ssh-keygen -C ""`), so the comment must come from
// agent.AddedKey.Comment or this test proves nothing.
func TestInvestigate_Criterion4_AgentSourcedComment_RealFakeAgent(t *testing.T) {
	fixturesDir := repoFixturesDir(t)
	dir := t.TempDir()
	copyFixtureInto(t, fixturesDir, "ed25519-openssh-encrypted-nopub", filepath.Join(dir, "id_ed25519"))

	// keyfile.OpenMaterial with the correct passphrase, purely to get the *decrypted* private
	// key ready for agent.AddedKey — the agent needs the raw key, not the encrypted file.
	priv := decryptedFixturePrivateKey(t, filepath.Join(dir, "id_ed25519"))

	keyring := agent.NewKeyring()
	const wantComment = "hasp-test-agent-comment"
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv, Comment: wantComment}); err != nil {
		t.Fatalf("keyring.Add: %v", err)
	}
	sockPath := pipedAgentSocket(t, keyring)
	t.Setenv("SSH_AUTH_SOCK", sockPath)

	env := runInvestigateJSON(t, dir, "show", "key", "id_ed25519")
	var detail keyDetailPartial
	if err := json.Unmarshal(env.Data, &detail); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if detail.Key.CommentSource != domain.FactSourceAgent {
		t.Fatalf("CommentSource = %q, want %q", detail.Key.CommentSource, domain.FactSourceAgent)
	}
	if detail.Key.AgentComment != wantComment {
		t.Fatalf("AgentComment = %q, want %q", detail.Key.AgentComment, wantComment)
	}
}

// TestInvestigate_Criterion4_NoSocket_DegradesAndExitsZero is roadmap.md §5.6 exit criterion 4's
// negative case: with SSH_AUTH_SOCK unset, the same command degrades to no agent comment and
// exits 0 — never an error, matching sshagent.List's own fail-open contract (T38, tdd.md §11).
func TestInvestigate_Criterion4_NoSocket_DegradesAndExitsZero(t *testing.T) {
	fixturesDir := repoFixturesDir(t)
	dir := t.TempDir()
	copyFixtureInto(t, fixturesDir, "ed25519-openssh-encrypted-nopub", filepath.Join(dir, "id_ed25519"))

	t.Setenv("SSH_AUTH_SOCK", "") // explicit unset, regardless of the ambient test environment

	env := runInvestigateJSON(t, dir, "show", "key", "id_ed25519") // exits non-zero (t.Fatal) if this errors at all
	var detail keyDetailPartial
	if err := json.Unmarshal(env.Data, &detail); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if detail.Key.CommentSource == domain.FactSourceAgent {
		t.Errorf("CommentSource = %q, want anything but agent-sourced with no agent running", detail.Key.CommentSource)
	}
	if detail.Key.AgentComment != "" {
		t.Errorf("AgentComment = %q, want empty with no agent running", detail.Key.AgentComment)
	}
}

// decryptedFixturePrivateKey decrypts path with testPassphrase and returns the raw private key,
// ready for agent.AddedKey.PrivateKey — duplicated here (package cli_test cannot see
// internal/adapter/keyfile's unexported helpers, and importing it directly for one call is
// simpler than reaching into another package's test-only helpers).
func decryptedFixturePrivateKey(t *testing.T, path string) any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	key, err := ssh.ParseRawPrivateKeyWithPassphrase(raw, []byte(testPassphrase))
	if err != nil {
		t.Fatalf("decrypt %s: %v", path, err)
	}
	return key
}
