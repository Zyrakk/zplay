package world

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLocal_Directory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "level.dat"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	worldDir, err := ResolveLocal(dir)
	if err != nil {
		t.Fatalf("ResolveLocal failed: %v", err)
	}
	if worldDir != dir {
		t.Errorf("expected %s, got %s", dir, worldDir)
	}
}

func TestResolveLocal_DirectoryNested(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "MyWorld")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "level.dat"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	worldDir, err := ResolveLocal(dir)
	if err != nil {
		t.Fatalf("ResolveLocal failed: %v", err)
	}
	if worldDir != nested {
		t.Errorf("expected %s, got %s", nested, worldDir)
	}
}

func TestResolveLocal_NoLevelDat(t *testing.T) {
	dir := t.TempDir()
	_, err := ResolveLocal(dir)
	if err == nil {
		t.Error("expected error for directory without level.dat")
	}
}

func TestResolveLocal_ZipFile(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "world.zip")
	createTestZip(t, zipPath, map[string][]byte{
		"level.dat": []byte("test"),
	})

	worldDir, err := ResolveLocal(zipPath)
	if err != nil {
		t.Fatalf("ResolveLocal failed: %v", err)
	}

	levelDat := filepath.Join(worldDir, "level.dat")
	if _, err := os.Stat(levelDat); os.IsNotExist(err) {
		t.Error("expected level.dat in extracted directory")
	}
}

func TestResolveLocal_TarGzFile(t *testing.T) {
	dir := t.TempDir()
	tgzPath := filepath.Join(dir, "world.tar.gz")
	createTestTarGz(t, tgzPath, map[string][]byte{
		"level.dat": []byte("test"),
	})

	worldDir, err := ResolveLocal(tgzPath)
	if err != nil {
		t.Fatalf("ResolveLocal failed: %v", err)
	}

	levelDat := filepath.Join(worldDir, "level.dat")
	if _, err := os.Stat(levelDat); os.IsNotExist(err) {
		t.Error("expected level.dat in extracted directory")
	}
}

func TestResolveLocal_ZipNestedWorld(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "world.zip")
	createTestZip(t, zipPath, map[string][]byte{
		"MyWorld/level.dat":        []byte("test"),
		"MyWorld/region/r.0.0.mca": []byte("region"),
	})

	worldDir, err := ResolveLocal(zipPath)
	if err != nil {
		t.Fatalf("ResolveLocal failed: %v", err)
	}

	levelDat := filepath.Join(worldDir, "level.dat")
	if _, err := os.Stat(levelDat); os.IsNotExist(err) {
		t.Error("expected level.dat in nested extracted directory")
	}
}

func TestResolveLocal_InvalidPath(t *testing.T) {
	_, err := ResolveLocal("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestResolveLocal_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "world.rar")
	if err := os.WriteFile(badFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveLocal(badFile)
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"https://example.com/world.zip", true},
		{"http://example.com/world.zip", true},
		{"/path/to/world", false},
		{"./world", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := IsURL(tt.input); got != tt.want {
			t.Errorf("IsURL(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// Test helpers

func createTestZip(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func createTestTarGz(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
}
