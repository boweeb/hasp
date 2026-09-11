package fpscheme

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNoNetworkGuard mechanically enforces T35's "no scheme ever opens a socket" boundary
// (roadmap.md §5.6 exit criterion 6), using the substitute mechanism T47 records.
//
// tdd.md §12 originally described running each scheme's Compute inside a net.Dialer whose
// Control callback fails socket creation — but that mechanism cannot observe a package that
// never dials in the first place: a test-constructed net.Dialer has no effect on code under test
// that holds no reference to it, since nothing in this package's Compute functions ever touches
// net.Dialer, net.Dial, or anything downstream of them. This guard instead asserts, the same
// mechanical way T13's internal/domain layering guard and T32's cmd/hasp layering guard already
// do, that this package's own source never imports the packages a socket or a subprocess would
// require.
//
// Deliberately a DIRECT-import check (go list -f '{{.Imports}}'), not go list -deps' full
// transitive closure the way T13's domain guard and T32's cmd/hasp guard both use: verified this
// session, golang.org/x/crypto/ssh — a dependency this package requires per tdd.md §18's table,
// for ssh.FingerprintSHA256 and the SSH wire-format marshaling the ssh-native-sha256 and
// legacy-ssh-md5 schemes need — itself directly imports "net" for its own Dial/Listen/Conn
// machinery, entirely unrelated to anything this package calls. A transitive check would
// therefore fail permanently on that inherited import regardless of whether fpscheme's own code
// ever dials, which would make the guard worthless — an alarm that never stops ringing gets
// ignored. Checking this package's own direct imports is the assertion that is actually true and
// actually useful: "the fingerprint scheme registry's own source code never imports net,
// net/http, or os/exec" — exactly T35's claim, no more and no less.
func TestNoNetworkGuard(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", `{{ join .Imports "\n" }}`, ".").Output()
	if err != nil {
		t.Fatalf("go list -f imports .: %v", err)
	}

	forbidden := []string{"net", "net/http", "os/exec"}

	for _, imp := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if imp == "" {
			continue
		}
		for _, f := range forbidden {
			if imp == f || strings.HasPrefix(imp, f+"/") {
				t.Errorf("internal/adapter/fpscheme must never directly import %q, but it does (T35, T47)", f)
			}
		}
	}
}
