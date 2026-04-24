// Package shellinit emits shell-specific wrapper functions that let splorer
// leave the user's shell in the last directory they navigated to.
//
// The wrapper creates a temp file, runs the real splorer binary with
// --cd-file pointing at it, and after splorer exits reads the path from the
// file and cd's to it. This pattern (also used by lf, nnn, yazi, zoxide) is
// necessary because a child process cannot change its parent shell's cwd —
// the shell has to do it itself on the child's behalf.
package shellinit

import "fmt"

// bashScript is the wrapper function for bash and zsh. `command splorer`
// bypasses any alias or function with the same name so we reach the real
// binary. The function preserves splorer's exit code so `$?` still works.
const bashScript = `splorer() {
    local cd_file
    cd_file="$(mktemp)" || return 1
    command splorer --cd-file "$cd_file" "$@"
    local rc=$?
    if [ -s "$cd_file" ]; then
        local dest
        dest="$(cat "$cd_file")"
        [ -n "$dest" ] && cd -- "$dest"
    fi
    rm -f -- "$cd_file"
    return $rc
}
`

// powershellScript is the wrapper function for Windows PowerShell and
// PowerShell 7+. We invoke "splorer.exe" (with the extension) so PowerShell's
// command resolver matches the Application, not this function — avoiding
// recursion.
const powershellScript = `function splorer {
    $cdFile = [System.IO.Path]::GetTempFileName()
    try {
        splorer.exe --cd-file $cdFile @args
        $dest = Get-Content -LiteralPath $cdFile -Raw -ErrorAction SilentlyContinue
        if ($dest) { Set-Location -LiteralPath $dest.Trim() }
    } finally {
        Remove-Item -LiteralPath $cdFile -Force -ErrorAction SilentlyContinue
    }
}
`

// Script returns the wrapper function source for the given shell.
// Supported shells: bash, zsh, powershell (also: pwsh as an alias).
func Script(shell string) (string, error) {
	switch shell {
	case "bash", "zsh":
		return bashScript, nil
	case "powershell", "pwsh":
		return powershellScript, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: bash, zsh, powershell)", shell)
	}
}
