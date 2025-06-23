package main

import (
	"io/fs"
	"os"
	"reflect"
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

func TestDifferenceFiles(t *testing.T) {
	path1 := "./testdata/difference_files/path1"
	path2 := "./testdata/difference_files/path2"

	expectedDifferences1 := map[string]struct{}{
		"file1.txt": {},
		"file2.txt": {},
	}
	expectedDifferences2 := map[string]struct{}{
		"file3.txt": {},
		"file4.txt": {},
	}

	files1, err := listFiles(os.DirFS(path1))
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	files2, err := listFiles(os.DirFS(path2))
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	differences1 := differenceFiles(files1, files2)
	if !reflect.DeepEqual(differences1, expectedDifferences1) {
		t.Fatalf("expected %v differences, got %v", expectedDifferences1, differences1)
	}

	differences2 := differenceFiles(files2, files1)
	if !reflect.DeepEqual(differences2, expectedDifferences2) {
		t.Fatalf("expected %v differences, got %v", expectedDifferences2, differences2)
	}
}

func TestIsFileContentDifferent(t *testing.T) {
	fs1 := os.DirFS("./testdata/difference_file_content/path1")
	fs2 := os.DirFS("./testdata/difference_file_content/path2")

	testCases := []struct {
		name          string
		fsA           fs.FS
		fsB           fs.FS
		path          string
		wantDifferent bool
		wantErr       bool
	}{
		{
			name:          "same content",
			fsA:           fs1,
			fsB:           fs2,
			path:          "file1.txt",
			wantDifferent: false,
			wantErr:       false,
		},
		{
			name:          "different content",
			fsA:           fs1,
			fsB:           fs2,
			path:          "file2.txt",
			wantDifferent: true,
			wantErr:       false,
		},
		{
			name:          "file not found",
			fsA:           fs1,
			fsB:           fs2,
			path:          "file3.txt",
			wantDifferent: false,
			wantErr:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotDifferent, err := isFileContentDifferent(tc.fsA, tc.fsB, tc.path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("isFileContentDifferent() error = %v, wantErr %v", err, tc.wantErr)
			}
			if gotDifferent != tc.wantDifferent {
				t.Errorf("isFileContentDifferent() = %v, want %v", gotDifferent, tc.wantDifferent)
			}
		})
	}
}
