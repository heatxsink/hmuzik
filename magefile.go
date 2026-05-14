//go:build mage

package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const name = "hmuzik"

func init() {
	_ = os.Setenv("MAGEFILE_VERBOSE", "true")
}

func Clean() {
	_ = os.RemoveAll(name)
}

func Vet() error {
	return sh.Run("go", "vet", "./...")
}

func Test() error {
	return sh.Run("go", "test", "-v", "-race", "./...")
}

func Lint() error {
	return sh.Run("golangci-lint", "run", "./...")
}

func Build() error {
	mg.Deps(Clean, Vet)
	return sh.RunWith(map[string]string{
		"GOOS":        runtime.GOOS,
		"GOARCH":      runtime.GOARCH,
		"CGO_ENABLED": "0",
	}, "go", "build", "-o", name, ".")
}

func Install() error {
	mg.Deps(Build)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	return sh.Copy(filepath.Join(binDir, name), name)
}
