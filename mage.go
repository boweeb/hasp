//go:build ignore

// Command mage.go is the zero-install Mage bootstrap (T32): `go run mage.go <target>` needs no
// separately installed mage binary. Per magefile.org, "because of the peculiarities of `go run`,
// if you run this way, go run will only ever exit with an error code of 0 or 1" — callers must
// treat any non-zero exit as a single boolean failure and never branch on a specific code.
package main

import (
	"os"

	"github.com/magefile/mage/mage"
)

func main() {
	os.Exit(mage.Main())
}
