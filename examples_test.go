package snok

import (
	"os/exec"
	"strings"
	"testing"
)

func TestModularExampleRunsSelfContainedCommandFile(t *testing.T) {
	cmd := exec.Command("go", "run", "./examples/modular", "greet", "Ada", "--title", "Dr", "--shout")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run ./examples/modular failed: %v\n%s", err, output)
	}
	if got, want := strings.TrimSpace(string(output)), "HELLO, DR ADA!"; got != want {
		t.Fatalf("modular example output = %q, want %q", got, want)
	}
}
