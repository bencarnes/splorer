package contentsearch

import "testing"

func TestBuildMatcher_Exact(t *testing.T) {
	m, err := buildMatcher(Options{Pattern: "func", Mode: ModeExact})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Match("type Foo func()") {
		t.Error("exact matcher should find substring")
	}
	if m.Match("type Foo") {
		t.Error("exact matcher should not match absent substring")
	}
	if m.Match("FUNC()") {
		t.Error("exact matcher should be case-sensitive by default")
	}
}

func TestBuildMatcher_ExactIgnoreCase(t *testing.T) {
	m, err := buildMatcher(Options{Pattern: "FUNC", Mode: ModeExact, IgnoreCase: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Match("type foo func()") {
		t.Error("case-insensitive exact matcher should match regardless of case")
	}
	if m.Match("just a line") {
		t.Error("case-insensitive exact matcher must still reject non-matches")
	}
}

func TestBuildMatcher_Regex(t *testing.T) {
	m, err := buildMatcher(Options{Pattern: `^func\s+\w+`, Mode: ModeRegex})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Match("func Foo()") {
		t.Error("regex should match line starting with func")
	}
	if m.Match("type Foo func()") {
		t.Error("regex anchored at start should not match mid-line occurrence")
	}
}

func TestBuildMatcher_RegexIgnoreCase(t *testing.T) {
	m, err := buildMatcher(Options{Pattern: `^func`, Mode: ModeRegex, IgnoreCase: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Match("FUNC Foo()") {
		t.Error("case-insensitive regex should match uppercase FUNC")
	}
}

func TestBuildMatcher_InvalidRegexReturnsError(t *testing.T) {
	_, err := buildMatcher(Options{Pattern: "(unclosed", Mode: ModeRegex})
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestBuildMatcher_EmptyPattern(t *testing.T) {
	_, err := buildMatcher(Options{Pattern: "", Mode: ModeExact})
	if err == nil {
		t.Error("expected error for empty pattern")
	}
}

func TestParseExtensions_Empty(t *testing.T) {
	if got := parseExtensions(""); got != nil {
		t.Errorf("empty input: want nil, got %v", got)
	}
	if got := parseExtensions("   "); got != nil {
		t.Errorf("whitespace input: want nil, got %v", got)
	}
}

func TestParseExtensions_NormalizesForms(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{".go", []string{".go"}},
		{"go", []string{".go"}},
		{".go,.md", []string{".go", ".md"}},
		{"go, md", []string{".go", ".md"}},
		{"  go  ,  md  ", []string{".go", ".md"}},
		{".GO", []string{".go"}},
		{",,,,go,,,", []string{".go"}},
	}
	for _, c := range cases {
		got := parseExtensions(c.in)
		if len(got) != len(c.want) {
			t.Errorf("parseExtensions(%q) has %d entries, want %d: %v",
				c.in, len(got), len(c.want), got)
			continue
		}
		for _, ext := range c.want {
			if !got[ext] {
				t.Errorf("parseExtensions(%q) missing %q: %v", c.in, ext, got)
			}
		}
	}
}
