package main

import (
	"fmt"
	"io"
	"net/http"
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

// TestRunRoundTripWithVideo downloads a test video from test-videos.co.uk,
// encrypts it, decrypts it, and verifies the round-trip.
func TestRunRoundTripWithVideo(t *testing.T) {
	// Use the 1MB Big Buck Bunny video (640x360, 10s, H.264 MP4)
	videoURL := "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/360/Big_Buck_Bunny_360_10s_1MB.mp4"
	
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	videoPath := filepath.Join(dir, "original.mp4")
	encryptedPath := filepath.Join(dir, "encrypted.mp4")
	decryptedPath := filepath.Join(dir, "decrypted.mp4")

	// Download the test video
	t.Logf("Downloading test video from %s", videoURL)
	if err := downloadFile(videoURL, videoPath); err != nil {
		t.Fatalf("failed to download video: %v", err)
	}

	// Create a key file
	mustWriteFile(t, keyPath, []byte("test-video-key"))

	// Encrypt the video
	t.Log("Encrypting video...")
	if err := run([]string{"xorchid.exe", keyPath, videoPath, encryptedPath}); err != nil {
		t.Fatalf("encrypt video: %v", err)
	}

	// Decrypt the video
	t.Log("Decrypting video...")
	if err := run([]string{"xorchid.exe", keyPath, encryptedPath, decryptedPath}); err != nil {
		t.Fatalf("decrypt video: %v", err)
	}

	// Verify the decrypted file matches the original
	t.Log("Verifying decrypted file...")
	if err := compareFiles(videoPath, decryptedPath); err != nil {
		t.Fatalf("file comparison failed: %v", err)
	}

	t.Log("Video round-trip test passed!")
}

// downloadFile downloads a file from a URL to the specified path
func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	return out.Sync()
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
