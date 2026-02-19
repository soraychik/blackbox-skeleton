package storage

import (
	"strings"
	"testing"
)

func TestDiffCycle(t *testing.T) {
	de := NewDiffEngine()

	oldContent := []byte("line1\nline2\nline3\nline4\nline5\n")
	newContent := []byte("line1\nline2\nline3\nlineX\nline5\n")

	patch, err := de.CreateDiff(oldContent, newContent, 1)
	if err != nil {
		t.Fatal("Error creating diff:", err)
	}

	t.Logf("Patch:\n%s", patch)

	result, err := de.ApplyDiff(oldContent, patch)
	if err != nil {
		t.Fatal("Error applying diff:", err)
	}

	t.Logf("Result:\n%s", string(result))

	if string(result) != string(newContent) {
		t.Errorf("Result doesn't match newContent.\nGot: %q\nExpected: %q", string(result), string(newContent))
	}
}

func TestDiffWithRepeatedLines(t *testing.T) {
	de := NewDiffEngine()

	oldContent := []byte("!\n!\n!\nline1\n!\n!\n")
	newContent := []byte("!\n!\n!\nline2\n!\n!\n")

	patch, err := de.CreateDiff(oldContent, newContent, 1)
	if err != nil {
		t.Fatal("Error creating diff:", err)
	}

	t.Logf("Patch:\n%s", patch)

	result, err := de.ApplyDiff(oldContent, patch)
	if err != nil {
		t.Fatal("Error applying diff:", err)
	}

	t.Logf("Result:\n%s", string(result))

	if string(result) != string(newContent) {
		t.Errorf("Result doesn't match newContent.\nGot: %q\nExpected: %q", string(result), string(newContent))
	}
}

func TestDiffFullReplace(t *testing.T) {
	de := NewDiffEngine()

	oldContent := []byte("line1\nline2\nline3\n")
	newContent := []byte("new1\nnew2\nnew3\n")

	patch, err := de.CreateDiff(oldContent, newContent, 1)
	if err != nil {
		t.Fatal("Error creating diff:", err)
	}

	t.Logf("Patch:\n%s", patch)

	result, err := de.ApplyDiff(oldContent, patch)
	if err != nil {
		t.Fatal("Error applying diff:", err)
	}

	t.Logf("Result:\n%s", string(result))

	if string(result) != string(newContent) {
		t.Errorf("Result doesn't match newContent")
	}
}

func TestDiffNoChange(t *testing.T) {
	de := NewDiffEngine()

	content := []byte("line1\nline2\nline3\n")

	patch, err := de.CreateDiff(content, content, 1)
	if err != nil {
		t.Fatal("Error creating diff:", err)
	}

	t.Logf("Patch (should be empty):\n%s", patch)

	result, err := de.ApplyDiff(content, patch)
	if err != nil {
		t.Fatal("Error applying diff:", err)
	}

	if string(result) != string(content) {
		t.Errorf("Result doesn't match original content")
	}
}

func TestDiffCRLFNormalization(t *testing.T) {
	de := NewDiffEngine()

	oldContent := []byte("line1\r\nline2\r\nline3\r\n")
	newContent := []byte("line1\nline2\nlineX\n")

	patch, err := de.CreateDiff(oldContent, newContent, 1)
	if err != nil {
		t.Fatal("Error creating diff:", err)
	}

	t.Logf("Patch:\n%s", patch)

	result, err := de.ApplyDiff(oldContent, patch)
	if err != nil {
		t.Fatal("Error applying diff:", err)
	}

	t.Logf("Result:\n%s", string(result))

	expected := "line1\nline2\nlineX\n"
	if string(result) != expected {
		t.Errorf("Result doesn't match.\nGot: %q\nExpected: %q", string(result), expected)
	}
}

func TestDiffManyRepeatedLines(t *testing.T) {
	de := NewDiffEngine()

	oldLines := make([]string, 100)
	for i := range oldLines {
		oldLines[i] = "!"
	}
	oldLines[50] = "config"
	oldContent := []byte(strings.Join(oldLines, "\n") + "\n")

	newLines := make([]string, 100)
	for i := range newLines {
		newLines[i] = "!"
	}
	newLines[50] = "config2"
	newContent := []byte(strings.Join(newLines, "\n") + "\n")

	patch, err := de.CreateDiff(oldContent, newContent, 1)
	if err != nil {
		t.Fatal("Error creating diff:", err)
	}

	t.Logf("Patch:\n%s", patch)

	result, err := de.ApplyDiff(oldContent, patch)
	if err != nil {
		t.Fatal("Error applying diff:", err)
	}

	if string(result) != string(newContent) {
		t.Errorf("Result doesn't match.\nGot len: %d, expected len: %d", len(result), len(newContent))
		t.Errorf("First diff at position 50: got %q, expected %q",
			strings.Split(string(result), "\n")[50],
			strings.Split(string(newContent), "\n")[50])
	}
}
