package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestHideRunnableUsageLine_ChangesTemplate guards against the replace silently becoming a no-op:
// the default template must differ from the narrowed one, and the narrowed one must carry the
// annotation condition.
func TestHideRunnableUsageLine_ChangesTemplate(t *testing.T) {
	root := &cobra.Command{Use: "x"}
	def := root.UsageTemplate()
	got := hideRunnableUsageLine(def)
	if got == def {
		t.Fatal("hideRunnableUsageLine left the template unchanged")
	}
	if !strings.Contains(got, helpOnlyAnnotation) || strings.Contains(got, runnableUsageCond) {
		t.Fatalf("narrowed template missing annotation condition:\n%s", got)
	}
}

func TestHideRunnableUsageLine_PanicsWhenTargetMissing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for a template without the expected Usage condition")
		}
	}()
	hideRunnableUsageLine("no usage here")
}

// TestStripHelpOnlyRunners_OnlyHelpOnly: every help-only command becomes non-Runnable; `check`
// (a real RunE) and leaf commands stay Runnable.
func TestStripHelpOnlyRunners_OnlyHelpOnly(t *testing.T) {
	root := NewRootCmd()
	StripHelpOnlyRunners(root)
	if root.Runnable() {
		t.Error("root still Runnable after strip")
	}
	for _, name := range []string{"list", "show", "find", "new", "adopt", "release", "edit"} {
		c, _, err := root.Find([]string{name})
		if err != nil || c.Runnable() {
			t.Errorf("%s: want found and non-Runnable (err=%v)", name, err)
		}
	}
	for _, path := range [][]string{{"check"}, {"version"}, {"list", "key"}} {
		c, _, err := root.Find(path)
		if err != nil || !c.Runnable() {
			t.Errorf("%v: want Runnable (err=%v)", path, err)
		}
	}
}

// TestEveryParentValidatesArgs: a verb parent added to NewRootCmd outside the validated loop would
// silently swallow unknown nouns again (cobra only validates root on its own).
func TestEveryParentValidatesArgs(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.HasSubCommands() && c.Args == nil {
			t.Errorf("%s has subcommands but nil Args", c.CommandPath())
		}
		for _, sub := range c.Commands() {
			if sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			walk(sub)
		}
	}
	walk(NewRootCmd())
}

// TestEveryRunnableCommandValidatesArgs: a Runnable command with nil Args accepts any positional
// silently — cobra skips legacyArgs for a command without subcommands — so every Runnable command
// must declare a validator (T55). cobra's own help (its positionals are a topic path), completion,
// and __complete (flag parsing disabled; args are the line being completed) are skipped: hasp does
// not own their argument contracts. completion and __complete are only added during Execute, so
// they are skipped here by name in case that ever changes.
func TestEveryRunnableCommandValidatesArgs(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		switch c.Name() {
		case "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return
		}
		if c.Runnable() && c.Args == nil {
			t.Errorf("%s is Runnable but has nil Args", c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(NewRootCmd())
}
