package shellinit

import (
	"strings"
	"testing"
)

func TestScript_SupportedShells(t *testing.T) {
	shells := []string{"bash", "zsh", "powershell", "pwsh"}
	for _, sh := range shells {
		t.Run(sh, func(t *testing.T) {
			s, err := Script(sh)
			if err != nil {
				t.Fatalf("Script(%q) error: %v", sh, err)
			}
			if s == "" {
				t.Fatalf("Script(%q) returned empty string", sh)
			}
			// Every wrapper must reference the --cd-file flag (otherwise it
			// has no way to recover the final directory from the TUI) and
			// must invoke the splorer binary.
			if !strings.Contains(s, "--cd-file") {
				t.Errorf("Script(%q) missing --cd-file flag: %q", sh, s)
			}
			if !strings.Contains(s, "splorer") {
				t.Errorf("Script(%q) missing splorer invocation", sh)
			}
		})
	}
}

// bash and zsh currently share the same wrapper, but the contract is that
// Script accepts both shell names — regardless of whether they ever diverge.
func TestScript_BashAndZshReturnNonEmptyForBoth(t *testing.T) {
	bash, err := Script("bash")
	if err != nil {
		t.Fatalf("bash: %v", err)
	}
	zsh, err := Script("zsh")
	if err != nil {
		t.Fatalf("zsh: %v", err)
	}
	if bash == "" || zsh == "" {
		t.Fatal("bash/zsh scripts must not be empty")
	}
}

// The PowerShell wrapper must invoke splorer.exe (with the extension) so
// PowerShell's command resolver doesn't match the wrapper function itself
// and recurse forever.
func TestScript_PowerShellAvoidsRecursion(t *testing.T) {
	s, err := Script("powershell")
	if err != nil {
		t.Fatalf("powershell: %v", err)
	}
	if !strings.Contains(s, "splorer.exe") {
		t.Errorf("PowerShell wrapper must call splorer.exe (with extension) "+
			"to avoid recursion; got:\n%s", s)
	}
}

// The bash/zsh wrapper must use `command splorer` to bypass any function or
// alias of the same name, otherwise it would recurse into itself.
func TestScript_BashAvoidsRecursion(t *testing.T) {
	s, err := Script("bash")
	if err != nil {
		t.Fatalf("bash: %v", err)
	}
	if !strings.Contains(s, "command splorer") {
		t.Errorf("bash wrapper must use `command splorer` to bypass the "+
			"function of the same name; got:\n%s", s)
	}
}

func TestScript_UnsupportedShell(t *testing.T) {
	_, err := Script("fish")
	if err == nil {
		t.Error("expected error for unsupported shell, got nil")
	}
}

func TestScript_EmptyShell(t *testing.T) {
	_, err := Script("")
	if err == nil {
		t.Error("expected error for empty shell name, got nil")
	}
}
