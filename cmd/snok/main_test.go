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

func TestInitAgentsFileCreatesAgentsFile(t *testing.T) {
	dir := t.TempDir()

	result, err := initAgentsFile(dir, strings.NewReader(""), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "created AGENTS.md" {
		t.Fatalf("result = %q, want created AGENTS.md", result)
	}
	content := readFile(t, filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(content, "# AGENTS.md") {
		t.Fatalf("created AGENTS.md missing title:\n%s", content)
	}
	if !strings.Contains(content, snokAgentsHeading) {
		t.Fatalf("created AGENTS.md missing Snok section:\n%s", content)
	}
	if !strings.Contains(content, "snok.NewTree") {
		t.Fatalf("created AGENTS.md missing command-tree guidance:\n%s", content)
	}
}

func TestInitAgentsFileDeclinesAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	original := "# Existing\n\nKeep this.\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	var prompt strings.Builder

	result, err := initAgentsFile(dir, strings.NewReader("\n"), &prompt)
	if err != nil {
		t.Fatal(err)
	}
	if result != "left AGENTS.md unchanged" {
		t.Fatalf("result = %q, want left AGENTS.md unchanged", result)
	}
	if got := readFile(t, path); got != original {
		t.Fatalf("AGENTS.md changed after declined append:\n%s", got)
	}
	if !strings.Contains(prompt.String(), "Append Snok command-authoring instructions? [y/N]") {
		t.Fatalf("prompt = %q, want append prompt", prompt.String())
	}
}

func TestInitAgentsFileDeclinesAppendOnEOF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	original := "# Existing\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := initAgentsFile(dir, strings.NewReader(""), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "left AGENTS.md unchanged" {
		t.Fatalf("result = %q, want left AGENTS.md unchanged", result)
	}
	if got := readFile(t, path); got != original {
		t.Fatalf("AGENTS.md changed after EOF prompt:\n%s", got)
	}
}

func TestInitAgentsFileAppendsToExistingAgentsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	original := "# Existing\n\nKeep this.\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := initAgentsFile(dir, strings.NewReader("yes\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "updated AGENTS.md" {
		t.Fatalf("result = %q, want updated AGENTS.md", result)
	}
	content := readFile(t, path)
	if !strings.HasPrefix(content, strings.TrimRight(original, "\n")) {
		t.Fatalf("AGENTS.md did not preserve existing content first:\n%s", content)
	}
	if count := strings.Count(content, snokAgentsHeading); count != 1 {
		t.Fatalf("Snok section count = %d, want 1:\n%s", count, content)
	}
}

func TestInitAgentsFileSkipsExistingSnokSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	original := "# Existing\n\n" + renderSnokAgentsSection()
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	var prompt strings.Builder

	result, err := initAgentsFile(dir, strings.NewReader("yes\n"), &prompt)
	if err != nil {
		t.Fatal(err)
	}
	if result != "AGENTS.md already includes Snok command-authoring instructions" {
		t.Fatalf("result = %q, want already initialized message", result)
	}
	if got := readFile(t, path); got != original {
		t.Fatalf("AGENTS.md changed despite existing section:\n%s", got)
	}
	if prompt.Len() != 0 {
		t.Fatalf("prompt = %q, want no prompt for existing section", prompt.String())
	}
}

func TestIsYes(t *testing.T) {
	tests := map[string]bool{
		"y":       true,
		"Y\n":     true,
		"yes":     true,
		" YES \n": true,
		"":        false,
		"\n":      false,
		"n":       false,
		"no":      false,
		"true":    false,
	}
	for input, want := range tests {
		if got := isYes(input); got != want {
			t.Fatalf("isYes(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestSnokInitCommandCreatesAgentsFile(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()
	snokBinary := filepath.Join(tmp, "snok")
	target := filepath.Join(tmp, "repo")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	buildSnok := exec.Command("go", "build", "-o", snokBinary, "./cmd/snok")
	buildSnok.Dir = root
	if output, err := buildSnok.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/snok failed: %v\n%s", err, output)
	}

	init := exec.Command(snokBinary, "init")
	init.Dir = target
	output, err := init.CombinedOutput()
	if err != nil {
		t.Fatalf("snok init failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "created AGENTS.md") {
		t.Fatalf("snok init output = %q, want creation message", output)
	}
	content := readFile(t, filepath.Join(target, "AGENTS.md"))
	if !strings.Contains(content, snokAgentsHeading) {
		t.Fatalf("snok init did not create Snok section:\n%s", content)
	}
}

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

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
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
