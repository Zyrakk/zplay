package world

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ResolveLocal takes a local path (directory or archive) and returns the path
// to a directory containing level.dat. Archives are extracted to a temp directory.
func ResolveLocal(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("path not found: %w", err)
	}

	if info.IsDir() {
		return findWorldRoot(path)
	}

	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		extractDir, err := os.MkdirTemp("", "zplay-world-*")
		if err != nil {
			return "", fmt.Errorf("creating temp dir: %w", err)
		}
		if err := extractZip(path, extractDir); err != nil {
			os.RemoveAll(extractDir)
			return "", fmt.Errorf("extracting zip: %w", err)
		}
		return findWorldRoot(extractDir)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		extractDir, err := os.MkdirTemp("", "zplay-world-*")
		if err != nil {
			return "", fmt.Errorf("creating temp dir: %w", err)
		}
		if err := extractTarGz(path, extractDir); err != nil {
			os.RemoveAll(extractDir)
			return "", fmt.Errorf("extracting tar.gz: %w", err)
		}
		return findWorldRoot(extractDir)
	default:
		return "", fmt.Errorf("unsupported format: %s (supported: directory, .zip, .tar.gz, .tgz)", filepath.Base(path))
	}
}

// Download fetches a URL to a temp file and returns the local file path.
func Download(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
	}

	ext := ".zip"
	lower := strings.ToLower(url)
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		ext = ".tar.gz"
	}

	tmpFile, err := os.CreateTemp("", "zplay-download-*"+ext)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("saving download: %w", err)
	}

	return tmpFile.Name(), nil
}

// IsURL returns true if the input looks like a URL.
func IsURL(input string) bool {
	return strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")
}

// findWorldRoot looks for level.dat at root or one level deep.
func findWorldRoot(dir string) (string, error) {
	if _, err := os.Stat(filepath.Join(dir, "level.dat")); err == nil {
		return dir, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())
			if _, err := os.Stat(filepath.Join(subdir, "level.dat")); err == nil {
				return subdir, nil
			}
		}
	}

	return "", fmt.Errorf("not a valid Minecraft world: level.dat not found in %s", dir)
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)

		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in tar: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}

	return nil
}
