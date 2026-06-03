package xor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// EncryptDecrypt runs XOR over the input string using the provided key.
func EncryptDecrypt(input, key string) string {
	kL := len(key)

	var tmp []string
	for i := 0; i < len(input); i++ {
		tmp = append(tmp, string(input[i]^key[i%kL]))
	}
	return strings.Join(tmp, "")
}

// EncryptDecryptBytes runs XOR over arbitrary bytes using the provided key.
func EncryptDecryptBytes(input, key []byte) []byte {
	output := make([]byte, len(input))
	for i := range input {
		output[i] = input[i] ^ key[i%len(key)]
	}
	return output
}

// EncryptDecryptFile streams XOR processing from inputPath to outputPath using keyPath.
// It displays a progress bar showing encryption/decryption progress.
func EncryptDecryptFile(keyPath, inputPath, outputPath string) error {
	same, err := samePath(inputPath, outputPath)
	if err != nil {
		return err
	}
	if same {
		return errors.New("inputfile and outputfile must be different paths")
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("read keyfile: %w", err)
	}
	if len(key) == 0 {
		return errors.New("keyfile must not be empty")
	}

	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("stat inputfile: %w", err)
	}
	if inputInfo.IsDir() {
		return errors.New("inputfile must be a file")
	}

	input, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open inputfile: %w", err)
	}
	defer input.Close()

	output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, inputInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open outputfile: %w", err)
	}
	defer output.Close()

	// Create progress bar
	p := mpb.New(
		mpb.WithWidth(64),
	)

	totalBytes := inputInfo.Size()
	bar := p.New(int64(totalBytes),
		mpb.BarStyle().Lbound("|").Filler("▓").Tip("▓").Padding("░").Rbound("|"),
		mpb.PrependDecorators(
			decor.CountersNoUnit("%d / %d"),
		),
		mpb.AppendDecorators(
			decor.CountersKibiByte("% .2f / % .2f"),
			decor.CurrentKibiByte("%.2f"),
			decor.Elapsed(decor.ET_STYLE_GO),
			decor.Name(" "),
			decor.AverageSpeed(decor.SizeB1024(0), "% .2f"),
		),
	)

	if err := EncryptDecryptStreamWithProgress(input, output, key, bar); err != nil {
		p.Wait()
		return fmt.Errorf("process inputfile: %w", err)
	}

	p.Wait()

	return nil
}

// EncryptDecryptStream reads from input, XORs each byte with key, and writes to output.
func EncryptDecryptStream(input io.Reader, output io.Writer, key []byte) error {
	const bufferSize = 1024 * 1024

	buffer := make([]byte, bufferSize)
	keyIndex := 0
	for {
		n, readErr := input.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			for i := range chunk {
				chunk[i] ^= key[keyIndex]
				keyIndex = (keyIndex + 1) % len(key)
			}
			written, err := output.Write(chunk)
			if err != nil {
				return err
			}
			if written != len(chunk) {
				return io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

// EncryptDecryptStreamWithProgress reads from input, XORs each byte with key, writes to output,
// and updates the progress bar as bytes are processed.
func EncryptDecryptStreamWithProgress(input io.Reader, output io.Writer, key []byte, bar *mpb.Bar) error {
	const bufferSize = 1024 * 1024

	buffer := make([]byte, bufferSize)
	keyIndex := 0
	for {
		n, readErr := input.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			for i := range chunk {
				chunk[i] ^= key[keyIndex]
				keyIndex = (keyIndex + 1) % len(key)
			}
			written, err := output.Write(chunk)
			if err != nil {
				return err
			}
			if written != len(chunk) {
				return io.ErrShortWrite
			}
			// Update progress bar
			bar.IncrInt64(int64(n))
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func samePath(left, right string) (bool, error) {
	absLeft, err := filepath.Abs(left)
	if err != nil {
		return false, fmt.Errorf("resolve inputfile path: %w", err)
	}

	absRight, err := filepath.Abs(right)
	if err != nil {
		return false, fmt.Errorf("resolve outputfile path: %w", err)
	}

	return absLeft == absRight, nil
}
