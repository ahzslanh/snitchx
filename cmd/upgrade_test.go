package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func writeTestTarGz(t *testing.T, entries map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range entries {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %q: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write body %q: %v", name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	return buf.Bytes()
}

func TestReleaseBinaryName(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{"windows", "snitchx.exe"},
		{"linux", "snitchx"},
		{"darwin", "snitchx"},
	}

	for _, tt := range tests {
		if got := releaseBinaryName(tt.goos); got != tt.want {
			t.Errorf("releaseBinaryName(%q) = %q, want %q", tt.goos, got, tt.want)
		}
	}
}

func TestReleaseArchiveName(t *testing.T) {
	tests := []struct {
		version string
		goos    string
		goarch  string
		want    string
	}{
		{"v1.2.3", "windows", "amd64", "snitchx_1.2.3_windows_amd64.tar.gz"},
		{"1.2.3", "linux", "arm64", "snitchx_1.2.3_linux_arm64.tar.gz"},
		{"v0.1.0", "darwin", "arm64", "snitchx_0.1.0_darwin_arm64.tar.gz"},
	}

	for _, tt := range tests {
		if got := releaseArchiveName(tt.version, tt.goos, tt.goarch); got != tt.want {
			t.Errorf("releaseArchiveName(%q, %q, %q) = %q, want %q",
				tt.version, tt.goos, tt.goarch, got, tt.want)
		}
	}
}

func TestExtractBinaryFromTarGzNamed(t *testing.T) {
	archive := writeTestTarGz(t, map[string]string{
		"snitchx_1.2.3_windows_amd64/snitchx.exe": "windows-binary",
		"snitchx_1.2.3_windows_amd64/README.md":   "docs",
	})

	destDir := t.TempDir()
	path, err := extractBinaryFromTarGzNamed(bytes.NewReader(archive), destDir, "snitchx.exe")
	if err != nil {
		t.Fatalf("extractBinaryFromTarGzNamed: %v", err)
	}

	wantPath := filepath.Join(destDir, "snitchx.exe")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read extracted binary: %v", err)
	}
	if string(content) != "windows-binary" {
		t.Errorf("extracted content = %q, want %q", content, "windows-binary")
	}
}

func TestExtractBinaryFromTarGzNamedMissing(t *testing.T) {
	archive := writeTestTarGz(t, map[string]string{
		"snitchx_1.2.3_linux_amd64/snitchx": "linux-binary",
	})

	_, err := extractBinaryFromTarGzNamed(bytes.NewReader(archive), t.TempDir(), "snitchx.exe")
	if err == nil {
		t.Fatal("expected an error when the requested binary is absent")
	}
}

func TestInstallBinaryReplacesAndRemovesBackup(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "snitchx.exe")
	src := filepath.Join(dir, "new-binary")
	backup := dst + ".bak"

	if err := os.WriteFile(dst, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("new-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := installBinary(src, dst); err != nil {
		t.Fatalf("installBinary: %v", err)
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new-binary" {
		t.Errorf("installed content = %q, want %q", content, "new-binary")
	}

	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Errorf("backup %q should have been removed, stat err = %v", backup, err)
	}
}

func TestInstallBinaryRestoresBackupOnFailure(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "snitchx.exe")
	missingSrc := filepath.Join(dir, "does-not-exist")
	backup := dst + ".bak"

	if err := os.WriteFile(dst, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := installBinary(missingSrc, dst); err == nil {
		t.Fatal("expected installBinary to fail when source is missing")
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("original binary was not restored: %v", err)
	}
	if string(content) != "old-binary" {
		t.Errorf("restored content = %q, want %q", content, "old-binary")
	}

	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Errorf("backup %q should not remain after restore, stat err = %v", backup, err)
	}
}
