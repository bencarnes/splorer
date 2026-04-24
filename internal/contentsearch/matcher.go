package contentsearch

import (
	"fmt"
	"regexp"
	"strings"
)

// Mode is whether the pattern is interpreted as a plain substring or a regex.
type Mode int

const (
	ModeExact Mode = iota // pattern is a literal substring
	ModeRegex             // pattern is a RE2 regex (stdlib regexp)
)

// Options bundles everything the matcher needs to know about a search.
type Options struct {
	Pattern    string
	Mode       Mode
	IgnoreCase bool
	Extensions string // raw comma-separated input from the UI; normalized by the walker
}

// matcher reports whether a line contains a hit for the search options.
type matcher interface {
	Match(line string) bool
}

// buildMatcher returns a matcher ready to scan lines, or an error if the
// options produce an invalid regex. Callers should surface the error in the
// UI before starting the walk.
func buildMatcher(opts Options) (matcher, error) {
	if opts.Pattern == "" {
		return nil, fmt.Errorf("pattern is empty")
	}
	if opts.Mode == ModeRegex {
		src := opts.Pattern
		if opts.IgnoreCase {
			src = "(?i)" + src
		}
		re, err := regexp.Compile(src)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
		return regexMatcher{re: re}, nil
	}
	// Exact: case-insensitive mode lowercases the pattern once so the line
	// comparison only has to lowercase one side.
	if opts.IgnoreCase {
		return exactMatcher{pattern: strings.ToLower(opts.Pattern), ignoreCase: true}, nil
	}
	return exactMatcher{pattern: opts.Pattern}, nil
}

type exactMatcher struct {
	pattern    string
	ignoreCase bool
}

func (m exactMatcher) Match(line string) bool {
	if m.ignoreCase {
		return strings.Contains(strings.ToLower(line), m.pattern)
	}
	return strings.Contains(line, m.pattern)
}

type regexMatcher struct {
	re *regexp.Regexp
}

func (m regexMatcher) Match(line string) bool {
	return m.re.MatchString(line)
}

// parseExtensions turns the user's raw comma-separated extension field into a
// normalized set. Both ".go" and "go" are accepted; leading/trailing
// whitespace is trimmed; matching is case-insensitive. An empty or all-blank
// input returns nil, meaning "no filter — accept all files".
func parseExtensions(raw string) map[string]bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		ext := strings.TrimSpace(part)
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		out[strings.ToLower(ext)] = true
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
