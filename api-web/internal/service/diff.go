package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"blackbox-api/internal/cache"
	"blackbox-api/internal/models"
	"blackbox-api/internal/repository"
	"blackbox-api/internal/storage"

	"github.com/sergi/go-diff/diffmatchpatch"
)

const ContextLines = 3

type MinIOGetter interface {
	DownloadConfig(ctx context.Context, path string) ([]byte, error)
}

func ComputeDiffLines(text1, text2 string) []models.DiffLine {
	dmp := diffmatchpatch.New()
	lineChar1, lineChar2, lineArray := dmp.DiffLinesToChars(text1, text2)
	diffs := dmp.DiffMain(lineChar1, lineChar2, false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	type tempLine struct {
		Type      string
		Content   string
		LeftNum   int
		RightNum  int
		isChanged bool
	}

	var allLines []tempLine
	leftLineNum := 1
	rightLineNum := 1

	for _, d := range diffs {
		lines := strings.Split(d.Text, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		for _, line := range lines {
			tl := tempLine{LeftNum: leftLineNum, RightNum: rightLineNum}
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				tl.Type = "unchanged"
				tl.Content = line
				leftLineNum++
				rightLineNum++
			case diffmatchpatch.DiffDelete:
				tl.Type = "removed"
				tl.Content = line
				tl.isChanged = true
				leftLineNum++
			case diffmatchpatch.DiffInsert:
				tl.Type = "added"
				tl.Content = line
				tl.isChanged = true
				rightLineNum++
			}
			allLines = append(allLines, tl)
		}
	}

	hasChanges := false
	for _, line := range allLines {
		if line.isChanged {
			hasChanges = true
			break
		}
	}
	if !hasChanges {
		var result []models.DiffLine
		for _, tl := range allLines {
			result = append(result, models.DiffLine{
				Type:     tl.Type,
				Content:  tl.Content,
				LeftNum:  tl.LeftNum,
				RightNum: tl.RightNum,
			})
		}
		return result
	}

	seenIdx := make(map[int]bool)
	for i, line := range allLines {
		if !line.isChanged {
			continue
		}
		start := i - ContextLines
		if start < 0 {
			start = 0
		}
		end := i + ContextLines + 1
		if end > len(allLines) {
			end = len(allLines)
		}
		for j := start; j < end; j++ {
			seenIdx[j] = true
		}
	}

	var result []models.DiffLine
	for i, tl := range allLines {
		if seenIdx[i] {
			result = append(result, models.DiffLine{
				Type:     tl.Type,
				Content:  tl.Content,
				LeftNum:  tl.LeftNum,
				RightNum: tl.RightNum,
			})
		}
	}

	return result
}

func GetCachedVersionContent(ctx context.Context, versionRepo repository.VersionRepository, minioClient MinIOGetter, version *models.ConfigVersion) ([]byte, error) {
	return cache.GlobalContentCache.GetOrLoad(version.ID, func() ([]byte, error) {
		return ReconstructVersionContent(ctx, versionRepo, minioClient, version)
	})
}

func ReconstructVersionContent(ctx context.Context, versionRepo repository.VersionRepository, minioClient MinIOGetter, version *models.ConfigVersion) ([]byte, error) {
	log.Printf("debug: reconstructVersionContent called with StorageType=%q, ParentVersionID=%v, StoragePath=%q",
		version.StorageType, version.ParentVersionID, version.StoragePath)

	if version.StorageType == "base" {
		return minioClient.DownloadConfig(ctx, version.StoragePath)
	}

	if version.StorageType == "diff" && version.ParentVersionID != nil {
		parentVersion, err := versionRepo.GetByID(ctx, *version.ParentVersionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent version: %w", err)
		}

		baseContent, err := GetCachedVersionContent(ctx, versionRepo, minioClient, parentVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to reconstruct parent: %w", err)
		}

		patchData, err := minioClient.DownloadConfig(ctx, version.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to download patch: %w", err)
		}

		diffEngine := storage.NewDiffEngine()
		patchContent := string(patchData)

		return diffEngine.ApplyDiff(baseContent, patchContent)
	}

	return nil, fmt.Errorf("unknown storage type: %s", version.StorageType)
}

func GetSchedulerURL() string {
	return getEnv("SCHEDULER_TRIGGER_URL", "http://scheduler:9090")
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
