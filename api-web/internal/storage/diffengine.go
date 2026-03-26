package storage

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type DiffEngine struct{}

func NewDiffEngine() *DiffEngine {
	return &DiffEngine{}
}

func (de *DiffEngine) normalize(content []byte) []byte {
	return []byte(strings.ReplaceAll(string(content), "\r\n", "\n"))
}

func (de *DiffEngine) CreateUnifiedDiff(oldContent, newContent []byte, oldName, newName string) string {
	oldContent = de.normalize(oldContent)
	newContent = de.normalize(newContent)

	oldText := string(oldContent)
	newText := string(newContent)

	dmp := diffmatchpatch.New()
	lineChar1, lineChar2, lineArray := dmp.DiffLinesToChars(oldText, newText)
	diffs := dmp.DiffMain(lineChar1, lineChar2, false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	lines := dmp.DiffCharsToLines(diffs, lineArray)

	var result strings.Builder

	result.WriteString(fmt.Sprintf("--- %s\n", oldName))
	result.WriteString(fmt.Sprintf("+++ %s\n", newName))

	result.WriteString(de.formatHunksFromDiffs(lines))

	return result.String()
}

func (de *DiffEngine) formatHunksFromDiffs(diffs []diffmatchpatch.Diff) string {
	type diffLine struct {
		typ    diffmatchpatch.Operation
		text   string
		oldPos int
		newPos int
	}

	var allLines []diffLine
	oldPos := 1
	newPos := 1

	for _, d := range diffs {
		lines := strings.Split(d.Text, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		for _, line := range lines {
			allLines = append(allLines, diffLine{
				typ:    d.Type,
				text:   line,
				oldPos: oldPos,
				newPos: newPos,
			})
			if d.Type == diffmatchpatch.DiffEqual || d.Type == diffmatchpatch.DiffDelete {
				oldPos++
			}
			if d.Type == diffmatchpatch.DiffEqual || d.Type == diffmatchpatch.DiffInsert {
				newPos++
			}
		}
	}

	if len(allLines) == 0 {
		return ""
	}

	var result strings.Builder

	const contextBefore = 3
	const contextAfter = 3

	hunkStarted := false
	hunkOldStart := 0
	hunkNewStart := 0
	var hunkContent []diffLine
	addedLines := 0
	removedLines := 0

	startHunk := func(idx int) {
		hunkStarted = true
		hunkContent = nil
		addedLines = 0
		removedLines = 0

		startIdx := idx - contextBefore
		if startIdx < 0 {
			startIdx = 0
		}

		for i := startIdx; i < idx; i++ {
			hunkContent = append(hunkContent, allLines[i])
		}

		hunkOldStart = allLines[idx].oldPos
		hunkNewStart = allLines[idx].newPos

		addedCtx := 0
		removedCtx := 0
		for _, c := range hunkContent {
			if c.typ == diffmatchpatch.DiffEqual {
				addedCtx++
				removedCtx++
			}
		}
		hunkOldStart = hunkOldStart - removedCtx
		hunkNewStart = hunkNewStart - addedCtx
	}

	flushHunk := func() {
		if !hunkStarted {
			return
		}

		oldCount := 0
		newCount := 0
		for _, lc := range hunkContent {
			if lc.typ == diffmatchpatch.DiffEqual || lc.typ == diffmatchpatch.DiffDelete {
				oldCount++
			}
			if lc.typ == diffmatchpatch.DiffEqual || lc.typ == diffmatchpatch.DiffInsert {
				newCount++
			}
		}

		result.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", hunkOldStart, oldCount, hunkNewStart, newCount))
		for _, line := range hunkContent {
			prefix := " "
			if line.typ == diffmatchpatch.DiffDelete {
				prefix = "-"
			} else if line.typ == diffmatchpatch.DiffInsert {
				prefix = "+"
			}
			result.WriteString(prefix + line.text + "\n")
		}
		hunkStarted = false
		hunkContent = nil
	}

	for i, line := range allLines {
		if line.typ == diffmatchpatch.DiffEqual {
			if hunkStarted {
				hunkContent = append(hunkContent, line)
				contextLines := 0
				for _, c := range hunkContent {
					if c.typ == diffmatchpatch.DiffEqual {
						contextLines++
					}
				}
				if contextLines >= contextBefore+contextAfter {
					flushHunk()
				}
			}
		} else if line.typ == diffmatchpatch.DiffDelete || line.typ == diffmatchpatch.DiffInsert {
			if !hunkStarted {
				startHunk(i)
			}
			hunkContent = append(hunkContent, line)
			if line.typ == diffmatchpatch.DiffDelete {
				removedLines++
			} else {
				addedLines++
			}
		}
	}

	flushHunk()

	return result.String()
}

func (de *DiffEngine) CreateDiff(oldContent, newContent []byte, baseVersionID int) (string, error) {
	oldContent = de.normalize(oldContent)
	newContent = de.normalize(newContent)

	oldName := fmt.Sprintf("version-%d", baseVersionID)
	newName := "new"
	return de.CreateUnifiedDiff(oldContent, newContent, oldName, newName), nil
}

func (de *DiffEngine) ApplyDiff(baseContent []byte, patchContent string) ([]byte, error) {
	baseContent = de.normalize(baseContent)
	baseLines := strings.Split(string(baseContent), "\n")
	patchLines := strings.Split(patchContent, "\n")

	var result []string
	baseIdx := 0
	i := 0

	for i < len(patchLines) {
		line := patchLines[i]

		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			i++
			continue
		}

		if strings.HasPrefix(line, "@@") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				oldStart := 0
				fmt.Sscanf(parts[1], "-%d", &oldStart)
				for baseIdx < oldStart-1 && baseIdx < len(baseLines) {
					result = append(result, baseLines[baseIdx])
					baseIdx++
				}
			}
			i++
			continue
		}

		if len(line) == 0 {
			i++
			continue
		}

		firstChar := line[0]

		if firstChar == '-' {
			content := line[1:]
			if baseIdx < len(baseLines) && baseLines[baseIdx] == content {
				baseIdx++
				i++
				continue
			}
			if baseIdx < len(baseLines) {
				baseIdx++
				i++
				continue
			}
			i++
		} else if firstChar == '+' {
			content := line[1:]
			result = append(result, content)
			i++
		} else if firstChar == ' ' || firstChar == '\t' {
			content := line[1:]
			if baseIdx < len(baseLines) && baseLines[baseIdx] == content {
				result = append(result, baseLines[baseIdx])
				baseIdx++
			} else if baseIdx < len(baseLines) {
				result = append(result, baseLines[baseIdx])
				baseIdx++
			}
			i++
		} else {
			if baseIdx < len(baseLines) {
				result = append(result, baseLines[baseIdx])
				baseIdx++
			}
			i++
		}
	}

	for baseIdx < len(baseLines) {
		result = append(result, baseLines[baseIdx])
		baseIdx++
	}

	return []byte(strings.Join(result, "\n")), nil
}

func (de *DiffEngine) CompressPatch(patchContent string) ([]byte, error) {
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)

	if _, err := gz.Write([]byte(patchContent)); err != nil {
		return nil, fmt.Errorf("Failed to compress patch: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("Failed to close gzip writer: %w", err)
	}

	return compressed.Bytes(), nil
}

func (de *DiffEngine) DecompressPatch(compressedData []byte) (string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return "", fmt.Errorf("Failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	patchData, err := io.ReadAll(gz)
	if err != nil {
		return "", fmt.Errorf("Failed to read decompressed data: %w", err)
	}

	return string(patchData), nil
}

func (de *DiffEngine) CalculateDiffSize(oldContent, newContent []byte) (int, int) {
	patch := de.CreateUnifiedDiff(oldContent, newContent, "old", "new")

	compressedPatch, err := de.CompressPatch(patch)
	if err != nil {
		return len(newContent), len(newContent)
	}

	return len(compressedPatch), len(newContent)
}

func (de *DiffEngine) ShouldUseDiff(oldContent, newContent []byte, threshold float64) bool {
	if len(oldContent) == 0 {
		return false
	}

	diffSize, fullSize := de.CalculateDiffSize(oldContent, newContent)

	if diffSize >= fullSize {
		return false
	}

	reductionRatio := float64(fullSize-diffSize) / float64(fullSize)
	return reductionRatio >= threshold
}

func (de *DiffEngine) GetStats(oldContent, newContent []byte) map[string]interface{} {
	patch := de.CreateUnifiedDiff(oldContent, newContent, "old", "new")
	diffSizeUncompressed := len(patch)

	compressedPatch, err := de.CompressPatch(patch)
	diffSize := len(compressedPatch)
	if err != nil {
		diffSize = diffSizeUncompressed
	}
	fullSize := len(newContent)

	shouldUseDiff := de.ShouldUseDiff(oldContent, newContent, 0.1)

	savings := fullSize - diffSize
	savingsPercent := float64(0)
	if fullSize > 0 {
		savingsPercent = float64(savings) / float64(fullSize) * 100
	}

	return map[string]interface{}{
		"original_size":          len(oldContent),
		"new_size":               len(newContent),
		"diff_size":              diffSize,
		"diff_size_uncompressed": diffSizeUncompressed,
		"full_size":              fullSize,
		"savings_bytes":          savings,
		"savings_percent":        savingsPercent,
		"should_use_diff":        shouldUseDiff,
	}
}

func (de *DiffEngine) ParseUnifiedDiffStats(diffContent string) (addedLines, removedLines int) {
	lines := strings.Split(diffContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			addedLines++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removedLines++
		}
	}
	return addedLines, removedLines
}
