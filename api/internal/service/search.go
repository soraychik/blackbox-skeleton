package service

import (
	"regexp"
	"sort"

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

// FindSnippetLines — оригинальная версия, оставлена для совместимости.
func FindSnippetLines(lines []string, matches [][]int, contextLines int) []models.SnippetLine {
	return FindSnippetLinesOptimized(lines, matches, contextLines)
}

// FindSnippetLinesOptimized строит индекс смещений строк один раз,
// затем использует бинарный поиск для каждого match.
// Сложность: O(N + M×logN) вместо O(N×M) у оригинала,
// где N = кол-во строк, M = кол-во совпадений.
func FindSnippetLinesOptimized(lines []string, matches [][]int, contextLines int) []models.SnippetLine {
	if len(lines) == 0 || len(matches) == 0 {
		return nil
	}

	// Строим массив начальных смещений каждой строки — один проход O(N).
	offsets := make([]int, len(lines))
	pos := 0
	for i, line := range lines {
		offsets[i] = pos
		pos += len(line) + 1 // +1 за \n
	}

	// findLine возвращает номер строки для байтового смещения через бинарный поиск O(logN).
	findLine := func(byteOffset int) int {
		idx := sort.SearchInts(offsets, byteOffset)
		// SearchInts возвращает первый индекс >= byteOffset.
		// Если точного совпадения нет — берём предыдущую строку.
		if idx < len(offsets) && offsets[idx] == byteOffset {
			return idx
		}
		if idx > 0 {
			return idx - 1
		}
		return 0
	}

	// Определяем какие строки являются совпадениями.
	matchingLines := make(map[int]bool)
	for _, match := range matches {
		start := findLine(match[0])
		end := findLine(match[1])
		for i := start; i <= end; i++ {
			matchingLines[i] = true
		}
	}

	// Собираем сниппеты с контекстом, без дублей.
	seenLines := make(map[int]bool)
	var snippetLines []models.SnippetLine

	for _, match := range matches {
		startLine := findLine(match[0])
		endLine := findLine(match[1])

		from := startLine - contextLines
		if from < 0 {
			from = 0
		}
		to := endLine + contextLines
		if to >= len(lines) {
			to = len(lines) - 1
		}

		for i := from; i <= to; i++ {
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