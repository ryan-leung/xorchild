package xor

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestEncryptDecryptFileRoundTripWithLocalFixture(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.bin")
	encryptedPath := filepath.Join(dir, "encrypted.bin")
	decryptedPath := filepath.Join(dir, "decrypted.bin")

	fixturePath := filepath.Join("..", "tests", "test.bin")
	keyPath := filepath.Join("..", "tests", "key.txt")

	if err := copyFirstNBytes(fixturePath, inputPath, 2*1024*1024); err != nil {
		t.Fatalf("prepare local fixture sample: %v", err)
	}

	if err := EncryptDecryptFile(keyPath, inputPath, encryptedPath); err != nil {
		t.Fatalf("encrypt file: %v", err)
	}

	if err := EncryptDecryptFile(keyPath, encryptedPath, decryptedPath); err != nil {
		t.Fatalf("decrypt file: %v", err)
	}

	originalHash, err := fileSHA256(inputPath)
	if err != nil {
		t.Fatalf("hash original file: %v", err)
	}

	decryptedHash, err := fileSHA256(decryptedPath)
	if err != nil {
		t.Fatalf("hash decrypted file: %v", err)
	}

	if originalHash != decryptedHash {
		t.Fatalf("file round trip mismatch: got %x want %x", decryptedHash, originalHash)
	}
}

func TestEncryptDecryptWithProgressAbortsOnWriteError(t *testing.T) {
	done := make(chan error, 1)
	key := []byte("secret")
	input := bytes.NewReader(bytes.Repeat([]byte("abcdef"), 400000))

	go func() {
		done <- encryptDecryptWithProgress(input, &errWriter{failAfter: 1}, key, int64(input.Len()))
	}()

	select {
	case err := <-done:
		if !errors.Is(err, errInjectedWrite) {
			t.Fatalf("got %v want %v", err, errInjectedWrite)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("encryptDecryptWithProgress did not return after write error")
	}
}

var errInjectedWrite = errors.New("injected write failure")

type errWriter struct {
	writes    int
	failAfter int
}

func (w *errWriter) Write(p []byte) (int, error) {
	if w.writes >= w.failAfter {
		return 0, errInjectedWrite
	}
	w.writes++
	return len(p), nil
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

func fileSHA256(path string) ([32]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return [32]byte{}, err
	}

	var sum [32]byte
	copy(sum[:], hasher.Sum(nil))
	return sum, nil
}
