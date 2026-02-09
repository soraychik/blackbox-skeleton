package storage

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type DiffOperation struct {
	Type    string `json:"type"`    // "add", "delete", "equal"
	Content string `json:"content"` // содержимое операции
	Length  int    `json:"length"`  // длина контента
}

type DiffPatch struct {
	BaseVersionID int             `json:"base_version_id"`
	DiffOps       []DiffOperation `json:"diff_ops"`
	CreatedAt     int64           `json:"created_at"`
	Checksum      string          `json:"checksum"`
}

type DiffEngine struct {
	dmp *diffmatchpatch.DiffMatchPatch
}

func NewDiffEngine() *DiffEngine {
	return &DiffEngine{
		dmp: diffmatchpatch.New(),
	}
}

func (de *DiffEngine) CreateDiff(oldContent, newContent []byte, baseVersionID int) (*DiffPatch, error) {
	oldStr := string(oldContent)
	newStr := string(newContent)

	diffs := de.dmp.DiffMain(oldStr, newStr, false)

	var diffOps []DiffOperation
	for _, diff := range diffs {
		opType := "equal"
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			opType = "add"
		case diffmatchpatch.DiffDelete:
			opType = "delete"
		}

		diffOps = append(diffOps, DiffOperation{
			Type:    opType,
			Content: diff.Text,
			Length:  len(diff.Text),
		})
	}

	patch := &DiffPatch{
		BaseVersionID: baseVersionID,
		DiffOps:       diffOps,
		CreatedAt:     0, // будет установлен при сохранении
	}

	// Вычисляем checksum патча
	patchData, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch for checksum: %w", err)
	}

	hash := sha256.Sum256(patchData)
	patch.Checksum = fmt.Sprintf("%x", hash)

	return patch, nil
}

func (de *DiffEngine) ApplyDiff(baseContent []byte, patch *DiffPatch) ([]byte, error) {
	if patch == nil {
		return nil, fmt.Errorf("patch is nil")
	}

	// Валидация checksum
	patchData, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch for validation: %w", err)
	}

	hash := sha256.Sum256(patchData)
	expectedChecksum := fmt.Sprintf("%x", hash)
	if patch.Checksum != expectedChecksum {
		return nil, fmt.Errorf("patch checksum validation failed: expected %s, got %s",
			expectedChecksum, patch.Checksum)
	}

	// Конвертируем DiffOps обратно в формат diffmatchpatch
	var diffs []diffmatchpatch.Diff
	for _, op := range patch.DiffOps {
		diffType := diffmatchpatch.DiffEqual
		switch op.Type {
		case "add":
			diffType = diffmatchpatch.DiffInsert
		case "delete":
			diffType = diffmatchpatch.DiffDelete
		}

		diffs = append(diffs, diffmatchpatch.Diff{
			Type: diffType,
			Text: op.Content,
		})
	}

	// Применяем патч - пересобираем текст из diffs
	var result strings.Builder
	for _, diff := range diffs {
		if diff.Type != diffmatchpatch.DiffDelete {
			result.WriteString(diff.Text)
		}
	}

	return []byte(result.String()), nil
}

func (de *DiffEngine) CompressPatch(patch *DiffPatch) ([]byte, error) {
	patchData, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch: %w", err)
	}

	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)

	if _, err := gz.Write(patchData); err != nil {
		return nil, fmt.Errorf("failed to compress patch: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return compressed.Bytes(), nil
}

func (de *DiffEngine) DecompressPatch(compressedData []byte) (*DiffPatch, error) {
	gz, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	patchData, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	var patch DiffPatch
	if err := json.Unmarshal(patchData, &patch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal patch: %w", err)
	}

	return &patch, nil
}

func (de *DiffEngine) CalculateDiffSize(oldContent, newContent []byte) (int, int) {
	patch, err := de.CreateDiff(oldContent, newContent, 0)
	if err != nil {
		return len(newContent), len(newContent) // fallback to full size
	}

	compressedPatch, err := de.CompressPatch(patch)
	if err != nil {
		return len(newContent), len(newContent) // fallback to full size
	}

	return len(compressedPatch), len(newContent)
}

func (de *DiffEngine) ShouldUseDiff(oldContent, newContent []byte, threshold float64) bool {
	if len(oldContent) == 0 {
		return false // для первого файла всегда используем full
	}

	diffSize, fullSize := de.CalculateDiffSize(oldContent, newContent)

	if diffSize >= fullSize {
		return false // diff больше или равен full
	}

	reductionRatio := float64(fullSize-diffSize) / float64(fullSize)
	return reductionRatio >= threshold
}

func (de *DiffEngine) GetStats(oldContent, newContent []byte) map[string]interface{} {
	diffSize, fullSize := de.CalculateDiffSize(oldContent, newContent)
	shouldUseDiff := de.ShouldUseDiff(oldContent, newContent, 0.1) // 10% threshold

	savings := fullSize - diffSize
	savingsPercent := float64(savings) / float64(fullSize) * 100

	return map[string]interface{}{
		"original_size":   len(oldContent),
		"new_size":        len(newContent),
		"diff_size":       diffSize,
		"full_size":       fullSize,
		"savings_bytes":   savings,
		"savings_percent": savingsPercent,
		"should_use_diff": shouldUseDiff,
	}
}
