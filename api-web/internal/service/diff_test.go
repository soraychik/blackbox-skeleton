package service

import (
	"testing"
)

func TestComputeDiffLines_NoChanges(t *testing.T) {
	text := "line1\nline2\nline3"
	result := ComputeDiffLines(text, text)

	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	for _, line := range result {
		if line.Type != "unchanged" {
			t.Errorf("expected unchanged, got %s", line.Type)
		}
	}
}

func TestComputeDiffLines_AddedLines(t *testing.T) {
	text1 := "line1\nline2"
	text2 := "line1\nline2\nline3\nline4"

	result := ComputeDiffLines(text1, text2)

	hasAdded := false
	for _, line := range result {
		if line.Type == "added" {
			hasAdded = true
			break
		}
	}
	if !hasAdded {
		t.Error("expected at least one added line")
	}
}

func TestComputeDiffLines_RemovedLines(t *testing.T) {
	text1 := "line1\nline2\nline3\nline4"
	text2 := "line1\nline2"

	result := ComputeDiffLines(text1, text2)

	hasRemoved := false
	for _, line := range result {
		if line.Type == "removed" {
			hasRemoved = true
			break
		}
	}
	if !hasRemoved {
		t.Error("expected at least one removed line")
	}
}

func TestComputeDiffLines_ContextWindow(t *testing.T) {
	text1 := ""
	for i := 0; i < 20; i++ {
		text1 += "unchanged line " + string(rune('0'+i%10)) + "\n"
	}
	text2 := text1
	text2 += "new line at end\n"

	result := ComputeDiffLines(text1, text2)

	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}

	hasAdded := false
	for _, line := range result {
		if line.Type == "added" {
			hasAdded = true
			break
		}
	}
	if !hasAdded {
		t.Error("expected at least one added line")
	}

	for _, line := range result {
		if line.Type == "unchanged" {
			if line.LeftNum == 0 && line.RightNum == 0 {
				t.Error("line numbers should not be both zero")
			}
		}
	}
}

func TestComputeDiffLines_EmptyStrings(t *testing.T) {
	result := ComputeDiffLines("", "")
	if len(result) != 0 {
		t.Errorf("expected 0 lines for empty input, got %d", len(result))
	}
}

func TestComputeDiffLines_EmptyToContent(t *testing.T) {
	text := "line1\nline2"
	result := ComputeDiffLines("", text)

	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	hasAdded := false
	for _, line := range result {
		if line.Type == "added" {
			hasAdded = true
			break
		}
	}
	if !hasAdded {
		t.Error("expected added lines when going from empty to content")
	}
}

func TestComputeDiffLines_ContentToEmpty(t *testing.T) {
	text := "line1\nline2"
	result := ComputeDiffLines(text, "")

	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	hasRemoved := false
	for _, line := range result {
		if line.Type == "removed" {
			hasRemoved = true
			break
		}
	}
	if !hasRemoved {
		t.Error("expected removed lines when going from content to empty")
	}
}
