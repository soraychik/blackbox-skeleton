package service

import (
	"testing"
)

func TestCompilePatterns(t *testing.T) {
	patterns := []string{"exact", "another"}
	res, err := CompilePatterns(patterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(res))
	}

	if !res[0].MatchString("exact") {
		t.Error("pattern should match")
	}
	if !res[0].MatchString("contains exact here") {
		t.Error("pattern should match as substring")
	}
}

func TestCompilePatterns_Empty(t *testing.T) {
	res, err := CompilePatterns([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(res))
	}
}

func TestCompilePatterns_WithEmptyString(t *testing.T) {
	patterns := []string{"valid", "", "also valid"}
	res, err := CompilePatterns(patterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 patterns (empty skipped), got %d", len(res))
	}
}

func TestCompilePatterns_SpecialChars(t *testing.T) {
	patterns := []string{`test\.*`, `with[special]chars`}
	res, err := CompilePatterns(patterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(res))
	}
}

func TestCompileSearchRegex_Simple(t *testing.T) {
	re, err := CompileSearchRegex("test", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !re.MatchString("this is a test string") {
		t.Error("pattern should match")
	}
}

func TestCompileSearchRegex_CaseSensitive(t *testing.T) {
	re, err := CompileSearchRegex("Test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if re.MatchString("test") {
		t.Error("case-sensitive pattern should not match lowercase")
	}
	if !re.MatchString("Test") {
		t.Error("case-sensitive pattern should match exact case")
	}
}

func TestCompileSearchRegex_CaseInsensitive(t *testing.T) {
	re, err := CompileSearchRegex("Test", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !re.MatchString("test") {
		t.Error("case-insensitive pattern should match lowercase")
	}
	if !re.MatchString("TEST") {
		t.Error("case-insensitive pattern should match uppercase")
	}
}

func TestCompileSearchRegex_InvalidRegexFallback(t *testing.T) {
	re, err := CompileSearchRegex("[invalid(regex", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if re.String() == "[invalid(regex" {
		t.Error("invalid regex should be escaped to literal")
	}
	if !re.MatchString("[invalid(regex") {
		t.Error("escaped literal should match")
	}
}

func TestAnyLineMatches(t *testing.T) {
	patterns := []string{"error", "warning"}
	res, err := CompilePatterns(patterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := []string{"normal line", "error: something failed", "another normal line"}
	if !AnyLineMatches(lines, res) {
		t.Error("should match 'error' in lines")
	}

	lines2 := []string{"normal", "also normal", "nothing here"}
	if AnyLineMatches(lines2, res) {
		t.Error("should not match any pattern")
	}
}

func TestAnyLineMatches_Empty(t *testing.T) {
	patterns := []string{"error"}
	res, err := CompilePatterns(patterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if AnyLineMatches([]string{}, res) {
		t.Error("empty lines should not match")
	}

	if AnyLineMatches(nil, res) {
		t.Error("nil lines should not match")
	}
}

func TestAnyLineMatches_EmptyPatterns(t *testing.T) {
	res, err := CompilePatterns([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if AnyLineMatches([]string{"test"}, res) {
		t.Error("empty patterns should not match anything")
	}
}

func TestFindSnippetLines(t *testing.T) {
	lines := []string{"line1", "line2", "pattern here", "line4", "line5"}
	matches := [][]int{
		{12, 19}, // "pattern here" - starts at byte 12 (after "line1\nline2\n")
	}

	snippets := FindSnippetLines(lines, matches, 1)

	if len(snippets) == 0 {
		t.Fatal("expected snippets")
	}

	matchCount := 0
	for _, s := range snippets {
		if s.Match {
			matchCount++
		}
	}

	if matchCount != 1 {
		t.Errorf("expected 1 matching line, got %d", matchCount)
	}
}

func TestFindSnippetLines_ContextOverlap(t *testing.T) {
	lines := []string{"line1", "pattern", "line3", "pattern", "line5"}
	matches := [][]int{
		{6, 12},  // "pattern" at index 1
		{18, 24}, // "pattern" at index 3
	}

	snippets := FindSnippetLines(lines, matches, 1)

	if len(snippets) == 0 {
		t.Fatal("expected snippets")
	}

	seenLines := make(map[int]bool)
	for _, s := range snippets {
		if seenLines[s.Line] {
			t.Errorf("line %d appears multiple times", s.Line)
		}
		seenLines[s.Line] = true
	}
}

func TestFindSnippetLines_EdgeCases(t *testing.T) {
	lines := []string{"pattern"}
	matches := [][]int{{0, 7}}

	snippets := FindSnippetLines(lines, matches, 2)

	if len(snippets) == 0 {
		t.Fatal("expected snippets")
	}

	for _, s := range snippets {
		if s.Line < 1 || s.Line > len(lines) {
			t.Errorf("line number %d out of range", s.Line)
		}
	}
}
