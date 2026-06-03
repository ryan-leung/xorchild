package xor

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecryptBytesRoundTrip(t *testing.T) {
	key := []byte("secret")
	input := []byte{0x00, 0x01, 0x02, 0xff, 'h', 'i'}

	encrypted := EncryptDecryptBytes(input, key)
	decrypted := EncryptDecryptBytes(encrypted, key)

	if string(decrypted) != string(input) {
		t.Fatalf("round trip mismatch: got %v want %v", decrypted, input)
	}
}

func TestEncryptDecryptMatchesStringAPI(t *testing.T) {
	input := "hello"
	key := "k"

	got := EncryptDecryptBytes([]byte(input), []byte(key))
	want := EncryptDecrypt(input, key)

	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEncryptDecryptStreamRoundTrip(t *testing.T) {
	key := []byte("secret")
	input := bytes.Repeat([]byte{0x00, 0x01, 0x02, 0xff, 'h', 'i'}, 200000)

	var encrypted bytes.Buffer
	if err := EncryptDecryptStream(bytes.NewReader(input), &encrypted, key); err != nil {
		t.Fatalf("encrypt stream: %v", err)
	}

	var decrypted bytes.Buffer
	if err := EncryptDecryptStream(bytes.NewReader(encrypted.Bytes()), &decrypted, key); err != nil {
		t.Fatalf("decrypt stream: %v", err)
	}

	if !bytes.Equal(decrypted.Bytes(), input) {
		t.Fatal("stream round trip mismatch")
	}
}

// TestEncryptDecryptStreamWithVideo downloads a test video from test-videos.co.uk
// and tests the stream-based encryption/decryption.
func TestEncryptDecryptStreamWithVideo(t *testing.T) {
	// Use the 1MB Big Buck Bunny video (640x360, 10s, H.264 MP4)
	videoURL := "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/360/Big_Buck_Bunny_360_10s_1MB.mp4"
	
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "original.mp4")
	encryptedPath := filepath.Join(dir, "encrypted.mp4")
	decryptedPath := filepath.Join(dir, "decrypted.mp4")

	// Download the test video
	t.Logf("Downloading test video from %s", videoURL)
	if err := downloadFile(videoURL, videoPath); err != nil {
		t.Fatalf("failed to download video: %v", err)
	}

	// Read the original video
	originalData, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatalf("failed to read video: %v", err)
	}

	key := []byte("stream-test-key")

	// Encrypt the video using stream
	t.Log("Encrypting video stream...")
	inputFile, err := os.Open(videoPath)
	if err != nil {
		t.Fatalf("failed to open video: %v", err)
	}
	defer inputFile.Close()

	encryptedFile, err := os.Create(encryptedPath)
	if err != nil {
		t.Fatalf("failed to create encrypted file: %v", err)
	}
	defer encryptedFile.Close()

	if err := EncryptDecryptStream(inputFile, encryptedFile, key); err != nil {
		t.Fatalf("encrypt stream: %v", err)
	}
	encryptedFile.Close()

	// Decrypt the video using stream
	t.Log("Decrypting video stream...")
	encryptedFile, err = os.Open(encryptedPath)
	if err != nil {
		t.Fatalf("failed to open encrypted file: %v", err)
	}
	defer encryptedFile.Close()

	decryptedFile, err := os.Create(decryptedPath)
	if err != nil {
		t.Fatalf("failed to create decrypted file: %v", err)
	}
	defer decryptedFile.Close()

	if err := EncryptDecryptStream(encryptedFile, decryptedFile, key); err != nil {
		t.Fatalf("decrypt stream: %v", err)
	}
	decryptedFile.Close()

	// Verify the decrypted file matches the original
	t.Log("Verifying decrypted file...")
	decryptedData, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Fatalf("failed to read decrypted file: %v", err)
	}

	if !bytes.Equal(decryptedData, originalData) {
		t.Fatal("video stream round trip mismatch")
	}

	t.Log("Video stream round-trip test passed!")
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
