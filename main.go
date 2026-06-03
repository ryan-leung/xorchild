package main

import (
	"errors"
	"fmt"
	"os"

	"xorchid/xor"
)

const usage = "usage: xorchid.exe keyfile inputfile outputfile"

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 4 {
		return errors.New("expected keyfile, inputfile, and outputfile")
	}

	return xor.EncryptDecryptFile(args[1], args[2], args[3])
}
