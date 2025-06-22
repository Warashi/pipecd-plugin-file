package main

import (
	"os"
	"testing"
)

func TestListFiles(t *testing.T) {
	path := "./testdata/list_files"
	expectedFiles := []string{"file1.txt", "file2.txt", "subdir/file3.txt"}

	files, err := listFiles(os.DirFS(path))
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	if len(files) != len(expectedFiles) {
		t.Fatalf("expected %d files, got %d", len(expectedFiles), len(files))
	}

	for _, expectedFile := range expectedFiles {
		if _, found := files[expectedFile]; !found {
			t.Errorf("expected file %s not found in the list", expectedFile)
		}
	}
}
