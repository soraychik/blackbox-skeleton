package service

import (
	"regexp"

	"blackbox-api/internal/models"
)

func CompilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	var out []*regexp.Regexp
	for _, p := range patterns {
		if p == "" {
			continue
		}
		literal := regexp.QuoteMeta(p)
		re, err := regexp.Compile(literal)
		if err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, nil
}

func CompileSearchRegex(input string, caseSensitive bool) (*regexp.Regexp, error) {
	pattern := input
	if !caseSensitive {
		pattern = "(?i)" + pattern
	}
	if re, err := regexp.Compile(pattern); err == nil {
		return re, nil
	}

	literal := regexp.QuoteMeta(input)
	if !caseSensitive {
		literal = "(?i)" + literal
	}
	return regexp.Compile(literal)
}

func AnyLineMatches(lines []string, res []*regexp.Regexp) bool {
	for _, line := range lines {
		for _, re := range res {
			if re.MatchString(line) {
				return true
			}
		}
	}
	return false
}

func FindSnippetLines(lines []string, matches [][]int, contextLines int) []models.SnippetLine {
	matchingLines := make(map[int]bool)
	for _, match := range matches {
		startLine := 0
		pos := 0
		for i, line := range lines {
			lineLen := len(line) + 1
			if pos+lineLen > match[0] {
				startLine = i
				break
			}
			pos += lineLen
		}
		endLine := startLine
		pos = 0
		for i, line := range lines {
			lineLen := len(line) + 1
			if pos+lineLen > match[1] {
				endLine = i
				break
			}
			pos += lineLen
		}
		for i := startLine; i <= endLine; i++ {
			matchingLines[i] = true
		}
	}

	seenLines := make(map[int]bool)
	var snippetLines []models.SnippetLine

	for _, match := range matches {
		startLine := 0
		pos := 0
		for i, line := range lines {
			lineLen := len(line) + 1
			if pos+lineLen > match[0] {
				startLine = i
				break
			}
			pos += lineLen
		}

		endLine := startLine
		pos = 0
		for i, line := range lines {
			lineLen := len(line) + 1
			if pos+lineLen > match[1] {
				endLine = i
				break
			}
			pos += lineLen
		}

		for i := startLine - contextLines; i <= endLine+contextLines; i++ {
			if i < 0 || i >= len(lines) {
				continue
			}
			if seenLines[i] {
				continue
			}
			seenLines[i] = true

			snippetLines = append(snippetLines, models.SnippetLine{
				Line:  i + 1,
				Text:  lines[i],
				Match: matchingLines[i],
			})
		}
	}

	return snippetLines
}
