package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pmenglund/snok/internal/hotrun"
)

func TestHotrunLinkBuildReuseAndRebuild(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()
	snokHome := filepath.Join(tmp, "home")
	snokBinary := filepath.Join(tmp, "snok")
	app := writeHotrunApp(t)

	buildSnok := exec.Command("go", "build", "-o", snokBinary, "./cmd/snok")
	buildSnok.Dir = root
	if output, err := buildSnok.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/snok failed: %v\n%s", err, output)
	}

	link := exec.Command(snokBinary, "link", "foo", app)
	link.Env = append(os.Environ(), "SNOK_HOME="+snokHome)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("snok link failed: %v\n%s", err, output)
	}

	foo := filepath.Join(snokHome, "bin", "foo")
	firstOutput := runCommand(t, foo, "SNOK_HOME="+snokHome)
	if strings.TrimSpace(firstOutput) != "base" {
		t.Fatalf("first output = %q, want base", firstOutput)
	}

	manager, err := hotrun.NewManagerAt(snokHome)
	if err != nil {
		t.Fatal(err)
	}
	firstBinary, built, err := manager.CachedBinary("foo")
	if err != nil {
		t.Fatal(err)
	}
	if built {
		t.Fatal("expected first cached binary to already exist")
	}
	firstInfo, err := os.Stat(firstBinary)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	secondOutput := runCommand(t, foo, "SNOK_HOME="+snokHome)
	if strings.TrimSpace(secondOutput) != "base" {
		t.Fatalf("second output = %q, want base", secondOutput)
	}
	secondInfo, err := os.Stat(firstBinary)
	if err != nil {
		t.Fatal(err)
	}
	if !secondInfo.ModTime().Equal(firstInfo.ModTime()) {
		t.Fatalf("cached binary was rebuilt without source changes: %s != %s", secondInfo.ModTime(), firstInfo.ModTime())
	}

	if err := os.WriteFile(filepath.Join(app, "extra.go"), []byte(strings.Join([]string{
		"package main",
		"func init() { commands = append(commands, \"extra\") }",
		"",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	thirdOutput := runCommand(t, foo, "SNOK_HOME="+snokHome)
	if strings.TrimSpace(thirdOutput) != "base extra" {
		t.Fatalf("third output = %q, want base extra", thirdOutput)
	}
	secondBinary, built, err := manager.CachedBinary("foo")
	if err != nil {
		t.Fatal(err)
	}
	if built {
		t.Fatal("expected rebuilt cached binary to already exist")
	}
	if secondBinary == firstBinary {
		t.Fatalf("cached binary path did not change after source change: %s", firstBinary)
	}
}

func runCommand(t *testing.T, path string, env ...string) string {
	t.Helper()
	cmd := exec.Command(path)
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", path, err, output)
	}
	return string(output)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test file")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

func writeHotrunApp(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/foo\n\ngo 1.22\n",
		"main.go": strings.Join([]string{
			"package main",
			`import ("fmt"; "strings")`,
			`var commands = []string{"base"}`,
			`func main() { fmt.Println(strings.Join(commands, " ")) }`,
			"",
		}, "\n"),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
