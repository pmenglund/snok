package hotrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryReadWrite(t *testing.T) {
	manager := newTestManager(t)
	registry := Registry{Links: map[string]Link{
		"foo": {Name: "foo", Source: filepath.Join(t.TempDir(), "foo")},
	}}
	if err := manager.WriteRegistry(registry); err != nil {
		t.Fatal(err)
	}
	got, err := manager.ReadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if got.Links["foo"].Name != "foo" || got.Links["foo"].Source != registry.Links["foo"].Source {
		t.Fatalf("unexpected registry: %#v", got)
	}
}

func TestFingerprintChangesForNewGoFile(t *testing.T) {
	root := writeModule(t)
	first, err := Fingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "extra.go"), []byte("package main\nfunc init() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := Fingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("fingerprint did not change after adding non-test Go file: %s", first)
	}
}

func TestFingerprintIgnoresTestFiles(t *testing.T) {
	root := writeModule(t)
	first, err := Fingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package main\nfunc TestExample(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := Fingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("fingerprint changed after adding test file: %s != %s", first, second)
	}
}

func TestLinkCreatesManagedSymlink(t *testing.T) {
	manager := newTestManager(t)
	source := writeModule(t)
	launcher := filepath.Join(t.TempDir(), "snok")
	if err := os.WriteFile(launcher, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Link("foo", source, launcher)
	if err != nil {
		t.Fatal(err)
	}
	if result.LinkPath != filepath.Join(manager.BinDir(), "foo") {
		t.Fatalf("unexpected link path: %s", result.LinkPath)
	}
	target, err := os.Readlink(result.LinkPath)
	if err != nil {
		t.Fatal(err)
	}
	if target != launcher {
		t.Fatalf("link target = %s, want %s", target, launcher)
	}
	registry, err := manager.ReadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if registry.Links["foo"].Source != source {
		t.Fatalf("registry source = %s, want %s", registry.Links["foo"].Source, source)
	}
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	manager, err := NewManagerAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func writeModule(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/foo\n\ngo 1.22\n",
		"main.go": strings.Join([]string{
			"package main",
			`import "fmt"`,
			`func main() { fmt.Println("hello") }`,
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
