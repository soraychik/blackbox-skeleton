package service

import (
	"context"
	"errors"
	"testing"

	"blackbox-api/internal/models"
)

type MockMinIOClient struct {
	DownloadConfigFn func(ctx context.Context, path string) ([]byte, error)
}

func (m *MockMinIOClient) DownloadConfig(ctx context.Context, path string) ([]byte, error) {
	if m.DownloadConfigFn != nil {
		return m.DownloadConfigFn(ctx, path)
	}
	return nil, errors.New("DownloadConfig not implemented")
}

type MockVersionRepo struct {
	GetByIDFn func(ctx context.Context, id int) (*models.ConfigVersion, error)
}

func (m *MockVersionRepo) GetByID(ctx context.Context, id int) (*models.ConfigVersion, error) {
	return m.GetByIDFn(ctx, id)
}

func (m *MockVersionRepo) GetByDevice(ctx context.Context, deviceID int, from, to string) ([]models.ConfigVersion, error) {
	return nil, nil
}

func (m *MockVersionRepo) GetPairsByDevice(ctx context.Context, deviceID int, from, to string) ([]models.VersionPair, error) {
	return nil, nil
}

func (m *MockVersionRepo) GetLastDate(ctx context.Context, deviceID int) (string, error) {
	return "", nil
}

func (m *MockVersionRepo) GetLatestForDevice(ctx context.Context, deviceID int) (*models.ConfigVersion, error) {
	return nil, nil
}

func (m *MockVersionRepo) ResolveByDate(ctx context.Context, deviceID int, date1, date2 string) (int, int, error) {
	return 0, 0, nil
}

func TestReconstructVersionContent_Base(t *testing.T) {
	version := &models.ConfigVersion{
		ID:          1,
		StorageType: "base",
		StoragePath: "configs/1/2024/01/01/abc123.txt",
	}

	repo := &MockVersionRepo{}
	minio := &MockMinIOClient{
		DownloadConfigFn: func(ctx context.Context, path string) ([]byte, error) {
			if path == "configs/1/2024/01/01/abc123.txt" {
				return []byte("config content here"), nil
			}
			return nil, errors.New("unexpected path")
		},
	}

	content, err := ReconstructVersionContent(context.Background(), repo, minio, version)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "config content here" {
		t.Errorf("expected 'config content here', got '%s'", string(content))
	}
}

func TestReconstructVersionContent_Diff(t *testing.T) {
	parentID := 1
	version := &models.ConfigVersion{
		ID:              2,
		StorageType:     "diff",
		StoragePath:     "diffs/1/2024/01/02/def456.patch",
		ParentVersionID: &parentID,
	}

	repo := &MockVersionRepo{
		GetByIDFn: func(ctx context.Context, id int) (*models.ConfigVersion, error) {
			if id == 1 {
				return &models.ConfigVersion{
					ID:          1,
					StorageType: "base",
					StoragePath: "configs/1/2024/01/01/abc123.txt",
				}, nil
			}
			return nil, errors.New("unexpected id")
		},
	}

	downloadCalls := 0
	minio := &MockMinIOClient{
		DownloadConfigFn: func(ctx context.Context, path string) ([]byte, error) {
			downloadCalls++
			if path == "configs/1/2024/01/01/abc123.txt" {
				return []byte("line1\nline2\nline3"), nil
			}
			if path == "diffs/1/2024/01/02/def456.patch" {
				return []byte(`--- version-1
+++ new
@@ -1,3 +1,4 @@
 line1
 line2
 line3
+line4
`), nil
			}
			return nil, errors.New("unexpected path")
		},
	}

	content, err := ReconstructVersionContent(context.Background(), repo, minio, version)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if downloadCalls != 2 {
		t.Errorf("expected 2 download calls, got %d", downloadCalls)
	}

	expected := "line1\nline2\nline3\nline4"
	if string(content) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(content))
	}
}

func TestReconstructVersionContent_DiffRepoError(t *testing.T) {
	parentID := 1
	version := &models.ConfigVersion{
		ID:              2,
		StorageType:     "diff",
		StoragePath:     "diffs/1/2024/01/02/def456.patch",
		ParentVersionID: &parentID,
	}

	repo := &MockVersionRepo{
		GetByIDFn: func(ctx context.Context, id int) (*models.ConfigVersion, error) {
			return nil, errors.New("database error")
		},
	}

	minio := &MockMinIOClient{}

	_, err := ReconstructVersionContent(context.Background(), repo, minio, version)
	if err == nil {
		t.Error("expected error")
	}
}

func TestReconstructVersionContent_DiffParentNil(t *testing.T) {
	version := &models.ConfigVersion{
		ID:          2,
		StorageType: "diff",
		StoragePath: "diffs/1/2024/01/02/def456.patch",
	}

	repo := &MockVersionRepo{}
	minio := &MockMinIOClient{}

	_, err := ReconstructVersionContent(context.Background(), repo, minio, version)
	if err == nil {
		t.Error("expected error when StorageType=diff but ParentVersionID=nil")
	}
}

func TestReconstructVersionContent_UnknownType(t *testing.T) {
	version := &models.ConfigVersion{
		ID:          1,
		StorageType: "unknown",
		StoragePath: "some/path",
	}

	repo := &MockVersionRepo{}
	minio := &MockMinIOClient{}

	_, err := ReconstructVersionContent(context.Background(), repo, minio, version)
	if err == nil {
		t.Error("expected error for unknown storage type")
	}
}

func TestReconstructVersionContent_MinIOError(t *testing.T) {
	version := &models.ConfigVersion{
		ID:          1,
		StorageType: "base",
		StoragePath: "configs/1/2024/01/01/abc123.txt",
	}

	repo := &MockVersionRepo{}
	minio := &MockMinIOClient{
		DownloadConfigFn: func(ctx context.Context, path string) ([]byte, error) {
			return nil, errors.New("minio connection failed")
		},
	}

	_, err := ReconstructVersionContent(context.Background(), repo, minio, version)
	if err == nil {
		t.Error("expected error from MinIO")
	}
}

func TestGetCachedVersionContent_Base(t *testing.T) {
	version := &models.ConfigVersion{
		ID:          100,
		StorageType: "base",
		StoragePath: "configs/1/2024/01/01/test.txt",
	}

	repo := &MockVersionRepo{}
	minio := &MockMinIOClient{
		DownloadConfigFn: func(ctx context.Context, path string) ([]byte, error) {
			return []byte("cached content"), nil
		},
	}

	content, err := GetCachedVersionContent(context.Background(), repo, minio, version)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "cached content" {
		t.Errorf("expected 'cached content', got '%s'", string(content))
	}

	content2, err := GetCachedVersionContent(context.Background(), repo, minio, version)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if string(content2) != "cached content" {
		t.Errorf("expected same content on second call, got '%s'", string(content2))
	}
}

func TestReconstructVersionContent_DiffCascading(t *testing.T) {
	grandParentID := 1
	parentID := 2
	version := &models.ConfigVersion{
		ID:              3,
		StorageType:     "diff",
		StoragePath:     "diffs/1/2024/01/03/def789.patch",
		ParentVersionID: &parentID,
	}

	getByIDCalls := 0
	repo := &MockVersionRepo{
		GetByIDFn: func(ctx context.Context, id int) (*models.ConfigVersion, error) {
			getByIDCalls++
			if id == 1 {
				return &models.ConfigVersion{
					ID:          1,
					StorageType: "base",
					StoragePath: "configs/1/2024/01/01/base.txt",
				}, nil
			}
			if id == 2 {
				return &models.ConfigVersion{
					ID:              2,
					StorageType:     "diff",
					StoragePath:     "diffs/1/2024/01/02/patch2.patch",
					ParentVersionID: &grandParentID,
				}, nil
			}
			return nil, errors.New("unexpected id")
		},
	}

	downloadCalls := 0
	minio := &MockMinIOClient{
		DownloadConfigFn: func(ctx context.Context, path string) ([]byte, error) {
			downloadCalls++
			if path == "configs/1/2024/01/01/base.txt" {
				return []byte("base line\n"), nil
			}
			if path == "diffs/1/2024/01/02/patch2.patch" {
				return []byte(`--- version-1
+++ new
@@ -1 +1,2 @@
 base line
+added line
`), nil
			}
			if path == "diffs/1/2024/01/03/def789.patch" {
				return []byte(`--- version-2
+++ new
@@ -1,2 +1,3 @@
 base line
+added line
+another line
`), nil
			}
			return nil, errors.New("unexpected path")
		},
	}

	_, err := ReconstructVersionContent(context.Background(), repo, minio, version)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if getByIDCalls != 2 {
		t.Errorf("expected 2 GetByID calls (parent + grandparent), got %d", getByIDCalls)
	}
	if downloadCalls < 2 {
		t.Errorf("expected at least 2 download calls, got %d", downloadCalls)
	}
}
