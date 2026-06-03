package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunRoundTrip(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	plainPath := filepath.Join(dir, "plain")
	encryptedPath := filepath.Join(dir, "encrypted")
	decryptedPath := filepath.Join(dir, "decrypted")

	mustWriteFile(t, keyPath, []byte("abc"))
	plain := []byte{0x00, 0x01, 0x02, 0xff, 'h', 'i'}
	mustWriteFile(t, plainPath, plain)

	if err := run([]string{"xorchid.exe", keyPath, plainPath, encryptedPath}); err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if err := run([]string{"xorchid.exe", keyPath, encryptedPath, decryptedPath}); err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	got, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("got %v want %v", got, plain)
	}
}

func TestRunRejectsEmptyKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	inputPath := filepath.Join(dir, "input")
	outputPath := filepath.Join(dir, "output")

	mustWriteFile(t, keyPath, nil)
	mustWriteFile(t, inputPath, []byte("plain"))

	if err := run([]string{"xorchid.exe", keyPath, inputPath, outputPath}); err == nil {
		t.Fatal("expected empty key error")
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRunRoundTripWithLocalFixture(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	inputPath := filepath.Join(dir, "input.bin")
	encryptedPath := filepath.Join(dir, "encrypted.bin")
	decryptedPath := filepath.Join(dir, "decrypted.bin")

	fixturePath := filepath.Join("tests", "test.bin")
	if err := copyFirstNBytes(fixturePath, inputPath, 2*1024*1024); err != nil {
		t.Fatalf("prepare local fixture sample: %v", err)
	}

	mustWriteFile(t, keyPath, []byte("test-video-key"))

	if err := run([]string{"xorchid.exe", keyPath, inputPath, encryptedPath}); err != nil {
		t.Fatalf("encrypt fixture: %v", err)
	}

	if err := run([]string{"xorchid.exe", keyPath, encryptedPath, decryptedPath}); err != nil {
		t.Fatalf("decrypt fixture: %v", err)
	}

	if err := compareFiles(inputPath, decryptedPath); err != nil {
		t.Fatalf("file comparison failed: %v", err)
	}
}

// compareFiles compares two files byte-by-byte
func compareFiles(path1, path2 string) error {
	f1, err := os.Open(path1)
	if err != nil {
		return err
	}
	defer f1.Close()

	f2, err := os.Open(path2)
	if err != nil {
		return err
	}
	defer f2.Close()

	const bufferSize = 8192
	buf1 := make([]byte, bufferSize)
	buf2 := make([]byte, bufferSize)

	for {
		n1, err1 := f1.Read(buf1)
		n2, err2 := f2.Read(buf2)

		if n1 != n2 || (err1 != nil && err2 != nil && err1.Error() != err2.Error()) {
			return io.ErrUnexpectedEOF
		}

		if err1 == io.EOF && err2 == io.EOF {
			return nil
		}

		if err1 != nil {
			return err1
		}
		if err2 != nil {
			return err2
		}

		if !bytesEqual(buf1[:n1], buf2[:n2]) {
			return io.ErrShortBuffer
		}
	}
}

// bytesEqual compares two byte slices for equality
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func copyFirstNBytes(srcPath, dstPath string, n int64) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.CopyN(dst, src, n); err != nil {
		return err
	}

	return nil
}
