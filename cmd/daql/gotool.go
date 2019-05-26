package main

import (
	"os/exec"
	"strings"

	"github.com/mb0/xelf/cor"
)

func gotool(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	return cmd.Output()
}

func gopkg(dir string) (string, error) {
	b, err := gotool(dir, "list", ".")
	if err != nil {
		return "", cor.Errorf("gopkg for %s: %v", dir, err)
	}
	return strings.TrimSpace(string(b)), nil
}
